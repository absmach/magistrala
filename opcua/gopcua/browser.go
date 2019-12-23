// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package gopcua

import (
	"context"
	"fmt"

	opcuaGopcua "github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	uaGopcua "github.com/gopcua/opcua/ua"
	"github.com/mainflux/mainflux/errors"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/opcua"
)

type NodeDef struct {
	NodeID      *uaGopcua.NodeID
	NodeClass   uaGopcua.NodeClass
	BrowseName  string
	Description string
	AccessLevel uaGopcua.AccessLevelType
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
	logger logger.Logger
}

// NewBrowser returns new OPC-UA browser instance.
func NewBrowser(ctx context.Context, log logger.Logger) opcua.Browser {
	return browser{
		ctx:    ctx,
		logger: log,
	}
}

func (c browser) Browse(serverURI, nodeID string) ([]string, error) {
	opts := []opcuaGopcua.Option{
		opcuaGopcua.SecurityMode(uaGopcua.MessageSecurityModeNone),
	}

	oc := opcuaGopcua.NewClient(serverURI, opts...)
	if err := oc.Connect(c.ctx); err != nil {
		return nil, errors.Wrap(errFailedConn, err)
	}
	defer oc.Close()

	n, err := uaGopcua.ParseNodeID(nodeID)
	if err != nil {
		return nil, errors.Wrap(errFailedParseNodeID, err)
	}

	nodeList, err := browse(oc.Node(n), "", 0)
	if err != nil {
		return nil, err
	}

	nodes := []string{}
	for _, s := range nodeList {
		node := fmt.Sprintf("ns=%d;%s", s.NodeID.Namespace(), s.NodeID.String())
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func browse(n *opcuaGopcua.Node, path string, level int) ([]NodeDef, error) {
	if level > 10 {
		return nil, nil
	}

	attrs, err := n.Attributes(
		uaGopcua.AttributeIDNodeClass,
		uaGopcua.AttributeIDBrowseName,
		uaGopcua.AttributeIDDescription,
		uaGopcua.AttributeIDAccessLevel,
		uaGopcua.AttributeIDDataType,
	)
	if err != nil {
		return nil, err
	}

	var def = NodeDef{
		NodeID: n.ID,
	}

	switch err := attrs[0].Status; err {
	case uaGopcua.StatusOK:
		def.NodeClass = uaGopcua.NodeClass(attrs[0].Value.Int())
	default:
		return nil, err
	}

	switch err := attrs[1].Status; err {
	case uaGopcua.StatusOK:
		def.BrowseName = attrs[1].Value.String()
	default:
		return nil, err
	}

	switch err := attrs[2].Status; err {
	case uaGopcua.StatusOK:
		def.Description = attrs[2].Value.String()
	case uaGopcua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	switch err := attrs[3].Status; err {
	case uaGopcua.StatusOK:
		def.AccessLevel = uaGopcua.AccessLevelType(attrs[3].Value.Int())
		def.Writable = def.AccessLevel&uaGopcua.AccessLevelTypeCurrentWrite == uaGopcua.AccessLevelTypeCurrentWrite
	case uaGopcua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	switch err := attrs[4].Status; err {
	case uaGopcua.StatusOK:
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
	case uaGopcua.StatusBadAttributeIDInvalid:
		// ignore
	default:
		return nil, err
	}

	def.Path = join(path, def.BrowseName)

	var nodes []NodeDef
	if def.NodeClass == uaGopcua.NodeClassVariable {
		nodes = append(nodes, def)
	}

	browseChildren := func(refType uint32) error {
		refs, err := n.ReferencedNodes(refType, uaGopcua.BrowseDirectionForward, uaGopcua.NodeClassAll, true)
		if err != nil {
			return err
		}

		for _, rn := range refs {
			children, err := browse(rn, def.Path, level+1)
			if err != nil {
				return err
			}
			nodes = append(nodes, children...)
		}
		return nil
	}

	if err := browseChildren(id.HasComponent); err != nil {
		return nil, err
	}
	if err := browseChildren(id.Organizes); err != nil {
		return nil, err
	}
	if err := browseChildren(id.HasProperty); err != nil {
		return nil, err
	}
	return nodes, nil
}

func join(a, b string) string {
	if a == "" {
		return b
	}
	return a + "." + b
}
