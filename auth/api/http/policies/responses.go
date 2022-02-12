package policies

import "net/http"

type createPolicyRes struct {
	created bool
}

func (res createPolicyRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res createPolicyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res createPolicyRes) Empty() bool {
	return false
}

type deletePoliciesRes struct {
	deleted bool
}

func (res deletePoliciesRes) Code() int {
	if res.deleted {
		return http.StatusNoContent
	}

	return http.StatusOK
}

func (res deletePoliciesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deletePoliciesRes) Empty() bool {
	return true
}

type errorRes struct {
	Err string `json:"error"`
}
