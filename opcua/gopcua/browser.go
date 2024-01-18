// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package gopcua

import (
	"context"
	"log/slog"

	"github.com/absmach/magistrala/opcua"
	"github.com/absmach/magistrala/pkg/errors"
	opcuagocpua "github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	uagocpua "github.com/gopcua/opcua/ua"
)

const maxChildrens = 4 // max browsing node children level

// NodeDef represents the node browser responnse.
type NodeDef struct {
	NodeID      *uagocpua.NodeID
	NodeClass   uagocpua.NodeClass
	BrowseName  string
	Description string
	AccessLevel uagocpua.AccessLevelType
	Path        string
	DataType    string
	Writable    bool
	Unit        string
	Scale       string
	Min         string
	Max         string
}

var _ opcua.Browser = (*browser)(nil)

type browser struct {
	ctx    context.Context
	logger *slog.Logger
}

// NewBrowser returns new OPC-UA browser instance.
func NewBrowser(ctx context.Context, log *slog.Logger) opcua.Browser {
	return browser{
		ctx:    ctx,
		logger: log,
	}
}

func (c browser) Browse(serverURI, nodeID string) ([]opcua.BrowsedNode, error) {
	opts := []opcuagocpua.Option{
		opcuagocpua.SecurityMode(uagocpua.MessageSecurityModeNone),
	}

	oc := opcuagocpua.NewClient(serverURI, opts...)
	if err := oc.Connect(c.ctx); err != nil {
		return nil, errors.Wrap(errFailedConn, err)
	}
	defer oc.Close()

	nodeList, err := browse(oc, nodeID, "", 0)
	if err != nil {
		return nil, err
	}

	nodes := []opcua.BrowsedNode{}
	for _, s := range nodeList {
		node := opcua.BrowsedNode{
			NodeID:      s.NodeID.String(),
			DataType:    s.DataType,
			Description: s.Description,
			Unit:        s.Unit,
			Scale:       s.Scale,
			BrowseName:  s.BrowseName,
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func browse(oc *opcuagocpua.Client, nodeID, path string, level int) ([]NodeDef, error) {
	if level > maxChildrens {
		return nil, nil
	}

	nid, err := uagocpua.ParseNodeID(nodeID)
	if err != nil {
		return []NodeDef{}, err
	}
	n := oc.Node(nid)

	attrs, err := n.Attributes(
		uagocpua.AttributeIDNodeClass,
		uagocpua.AttributeIDBrowseName,
		uagocpua.AttributeIDDescription,
		uagocpua.AttributeIDAccessLevel,
		uagocpua.AttributeIDDataType,
	)
	if err != nil {
		return nil, err
	}

	def := NodeDef{
		NodeID: n.ID,
	}

	switch err := attrs[0].Status; err {
	case uagocpua.StatusOK:
		def.NodeClass = uagocpua.NodeClass(attrs[0].Value.Int())
	default:
		return nil, err
	}

	switch err := attrs[1].Status; err {
	case uagocpua.StatusOK:
		def.BrowseName = attrs[1].Value.String()
	default:
		return nil, err
	}

	switch err := attrs[2].Status; err {
	case uagocpua.StatusOK:
		def.Description = attrs[2].Value.String()
	case uagocpua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	switch err := attrs[3].Status; err {
	case uagocpua.StatusOK:
		def.AccessLevel = uagocpua.AccessLevelType(attrs[3].Value.Int())
		def.Writable = def.AccessLevel&uagocpua.AccessLevelTypeCurrentWrite == uagocpua.AccessLevelTypeCurrentWrite
	case uagocpua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	switch err := attrs[4].Status; err {
	case uagocpua.StatusOK:
		switch v := attrs[4].Value.NodeID().IntID(); v {
		case id.DateTime:
			def.DataType = "time.Time"
		case id.Boolean:
			def.DataType = "bool"
		case id.SByte:
			def.DataType = "int8"
		case id.Int16:
			def.DataType = "int16"
		case id.Int32:
			def.DataType = "int32"
		case id.Byte:
			def.DataType = "byte"
		case id.UInt16:
			def.DataType = "uint16"
		case id.UInt32:
			def.DataType = "uint32"
		case id.UtcTime:
			def.DataType = "time.Time"
		case id.String:
			def.DataType = "string"
		case id.Float:
			def.DataType = "float32"
		case id.Double:
			def.DataType = "float64"
		default:
			def.DataType = attrs[4].Value.NodeID().String()
		}
	case uagocpua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	def.Path = join(path, def.BrowseName)

	var nodes []NodeDef
	if def.NodeClass == uagocpua.NodeClassVariable {
		nodes = append(nodes, def)
	}

	bc, err := browseChildren(oc, n, def.Path, level, id.HasComponent)
	if err != nil {
		return nil, err
	}
	nodes = append(nodes, bc...)

	bc, err = browseChildren(oc, n, def.Path, level, id.Organizes)
	if err != nil {
		return nil, err
	}
	nodes = append(nodes, bc...)

	bc, err = browseChildren(oc, n, def.Path, level, id.HasProperty)
	if err != nil {
		return nil, err
	}
	nodes = append(nodes, bc...)

	return nodes, nil
}

func browseChildren(c *opcuagocpua.Client, n *opcuagocpua.Node, path string, level int, typeDef uint32) ([]NodeDef, error) {
	nodes := []NodeDef{}
	refs, err := n.ReferencedNodes(typeDef, uagocpua.BrowseDirectionForward, uagocpua.NodeClassAll, true)
	if err != nil {
		return []NodeDef{}, err
	}

	for _, ref := range refs {
		children, err := browse(c, ref.ID.String(), path, level+1)
		if err != nil {
			return []NodeDef{}, err
		}
		nodes = append(nodes, children...)
	}

	return nodes, nil
}

func join(a, b string) string {
	if a == "" {
		return b
	}
	return a + "." + b
}
