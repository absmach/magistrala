package svcutil

import "fmt"

type ExternalOperation int

func (op ExternalOperation) String(operations []string) string {
	if (int(op) < 0) || (int(op) == len(operations)) {
		return fmt.Sprintf("UnknownOperation(%d)", op)
	}
	return operations[op]
}

type ExternalOperationPerm struct {
	opPerm      map[ExternalOperation]Permission
	expectedOps []ExternalOperation
	opNames     []string
}

func NewExternalOperationPerm(expectedOps []ExternalOperation, opNames []string) ExternalOperationPerm {
	return ExternalOperationPerm{
		opPerm:      make(map[ExternalOperation]Permission),
		expectedOps: expectedOps,
		opNames:     opNames,
	}
}

func (eopp ExternalOperationPerm) isKeyRequired(eop ExternalOperation) bool {
	for _, key := range eopp.expectedOps {
		if key == eop {
			return true
		}
	}
	return false
}

func (eopp ExternalOperationPerm) AddOperationPermissionMap(eopMap map[ExternalOperation]Permission) error {
	// First iteration check all the keys are valid, If any one key is invalid then no key should be added.
	for eop := range eopMap {
		if !eopp.isKeyRequired(eop) {
			return fmt.Errorf("%v is not a valid external operation", eop.String(eopp.opNames))
		}
	}
	for eop, perm := range eopMap {
		eopp.opPerm[eop] = perm
	}
	return nil
}

func (eopp ExternalOperationPerm) AddOperationPermission(eop ExternalOperation, perm Permission) error {
	if !eopp.isKeyRequired(eop) {
		return fmt.Errorf("%v is not a valid external operation", eop.String(eopp.opNames))
	}
	eopp.opPerm[eop] = perm
	return nil
}

func (eopp ExternalOperationPerm) Validate() error {
	for eop := range eopp.opPerm {
		if !eopp.isKeyRequired(eop) {
			return fmt.Errorf("ExternalOperationPerm: \"%s\" is not a valid external operation", eop.String(eopp.opNames))
		}
	}
	for _, eeo := range eopp.expectedOps {
		if _, ok := eopp.opPerm[eeo]; !ok {
			return fmt.Errorf("ExternalOperationPerm: \"%s\" external operation is missing", eeo.String(eopp.opNames))
		}
	}
	return nil
}

func (eopp ExternalOperationPerm) GetPermission(eop ExternalOperation) (Permission, error) {
	if perm, ok := eopp.opPerm[eop]; ok {
		return perm, nil
	}
	return "", fmt.Errorf("external operation \"%s\" doesn't have any permissions", eop.String(eopp.opNames))
}
