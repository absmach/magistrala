package main

import (
	"encoding/json"
	"fmt"
)

type OperationType uint32

const (
	CreateOp OperationType = iota
	ReadOp
	ListOp
	UpdateOp
	DeleteOp
)

func (ot OperationType) String() string {
	switch ot {
	case CreateOp:
		return "create"
	case ReadOp:
		return "read"
	case ListOp:
		return "list"
	case UpdateOp:
		return "update"
	case DeleteOp:
		return "delete"
	default:
		return fmt.Sprintf("unknown operation type %d", ot)
	}
}

func (ot OperationType) MarshalText() ([]byte, error) {
	return []byte(ot.String()), nil
}

type OperationScope struct {
	Operations map[OperationType]string `json:"operations,omitempty"`
}

type DomainEntityType uint32

const (
	DomainManagementScope DomainEntityType = iota
	DomainGroupsScope
	DomainChannelsScope
	DomainThingsScope
	DomainNullScope
)

func (det DomainEntityType) String() string {
	switch det {
	case DomainManagementScope:
		return "domain_management"
	case DomainGroupsScope:
		return "groups"
	case DomainChannelsScope:
		return "channels"
	case DomainThingsScope:
		return "things"
	case DomainNullScope:
		return "null"
	default:
		return fmt.Sprintf("unknown domain entity type %d", det)
	}
}

func (det DomainEntityType) MarshalText() ([]byte, error) {
	return []byte(det.String()), nil
}

type DomainScope struct {
	Entities map[DomainEntityType]string `json:"entities,omitempty"`
}

func main() {
	// OperationScope works because map keys are encoded correctly
	os := &OperationScope{
		Operations: map[OperationType]string{
			CreateOp: "allowed",
			ReadOp:   "allowed",
		},
	}
	osJSON, _ := json.MarshalIndent(os, "", "  ")
	fmt.Println("OperationScope:", string(osJSON))

	// DomainScope does not work as intended for map keys
	ds := &DomainScope{
		Entities: map[DomainEntityType]string{
			DomainManagementScope: "allowed",
			DomainGroupsScope:     "allowed",
		},
	}
	dsJSON, _ := json.MarshalIndent(ds, "", "  ")
	fmt.Println("DomainScope (incorrect):", string(dsJSON))
}
