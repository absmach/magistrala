package svcutil

import "fmt"

type Permission string

func (p Permission) String() string {
	return string(p)
}

type Operation int

func (op Operation) String(operations []string) string {
	if (int(op) < 0) || (int(op) == len(operations)) {
		return fmt.Sprintf("UnknownOperation(%d)", op)
	}
	return operations[op]
}

type OperationPerm struct {
	opPerm      map[Operation]Permission
	expectedOps []Operation
	opNames     []string
}

func NewOperationPerm(expectedOps []Operation, opNames []string) OperationPerm {
	return OperationPerm{
		opPerm:      make(map[Operation]Permission),
		expectedOps: expectedOps,
		opNames:     opNames,
	}
}

func (opp OperationPerm) isKeyRequired(op Operation) bool {
	for _, key := range opp.expectedOps {
		if key == op {
			return true
		}
	}
	return false
}

func (opp OperationPerm) AddOperationPermissionMap(opMap map[Operation]Permission) error {
	// First iteration check all the keys are valid, If any one key is invalid then no key should be added.
	for op := range opMap {
		if !opp.isKeyRequired(op) {
			return fmt.Errorf("%v is not a valid operation", op.String(opp.opNames))
		}
	}
	for op, perm := range opMap {
		opp.opPerm[op] = perm
	}
	return nil
}

func (opp OperationPerm) AddOperationPermission(op Operation, perm Permission) error {
	if !opp.isKeyRequired(op) {
		return fmt.Errorf("%v is not a valid operation", op.String(opp.opNames))
	}
	opp.opPerm[op] = perm
	return nil
}

func (opp OperationPerm) Validate() error {
	for op := range opp.opPerm {
		if !opp.isKeyRequired(op) {
			return fmt.Errorf("OperationPerm: \"%s\" is not a valid operation", op.String(opp.opNames))
		}
	}
	for _, eeo := range opp.expectedOps {
		if _, ok := opp.opPerm[eeo]; !ok {
			return fmt.Errorf("OperationPerm: \"%s\" operation is missing", eeo.String(opp.opNames))
		}
	}
	return nil
}

func (opp OperationPerm) GetPermission(op Operation) (Permission, error) {
	if perm, ok := opp.opPerm[op]; ok {
		return perm, nil
	}
	return "", fmt.Errorf("operation \"%s\" doesn't have any permissions", op.String(opp.opNames))
}
