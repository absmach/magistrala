// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"encoding/csv"
	"io"
	"os"

	"github.com/absmach/magistrala/pkg/errors"
)

const (
	columns = 2
	path    = "/store/nodes.csv"
)

var (
	errNotFound  = errors.New("file not found")
	errWriteFile = errors.New("failed de write file")
	errOpenFile  = errors.New("failed to open file")
	errReadFile  = errors.New("failed to read file")
	errEmptyLine = errors.New("empty or incomplete line found in file")
)

// Node represents an OPC-UA node.
type Node struct {
	ServerURI string
	NodeID    string
}

// Save stores a successful subscription.
func Save(serverURI, nodeID string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return errors.Wrap(errWriteFile, err)
	}
	csvWriter := csv.NewWriter(file)
	err = csvWriter.Write([]string{serverURI, nodeID})
	csvWriter.Flush()
	if err != nil {
		return errors.Wrap(errWriteFile, err)
	}

	return nil
}

// ReadAll returns all stored subscriptions.
func ReadAll() ([]Node, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, errors.Wrap(errNotFound, err)
	}

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, errors.Wrap(errOpenFile, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	nodes := []Node{}
	for {
		l, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(errReadFile, err)
		}

		if len(l) < columns {
			return nil, errEmptyLine
		}

		nodes = append(nodes, Node{l[0], l[1]})
	}

	return nodes, nil
}
