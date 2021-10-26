package keto

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/auth"
	acl "github.com/ory/keto/proto/ory/keto/acl/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestIsSubjectSet(t *testing.T) {
	cases := []struct {
		desc       string
		subjectSet string
		result     bool
	}{
		{
			desc:       "check valid subject set",
			subjectSet: "namespace:object#relation",
			result:     true,
		},
		{
			desc:       "check invalid subject set, missing namespace field",
			subjectSet: ":object#relation",
			result:     false,
		},
		{
			desc:       "check invalid subject set, missing object field",
			subjectSet: "namespace:#relation",
			result:     false,
		},
		{
			desc:       "check invalid subject set, missing relation field",
			subjectSet: "namespace:object#",
			result:     false,
		},
		{
			desc:       "check invalid subject set, empty subject set",
			subjectSet: ":#",
			result:     false,
		},
		{
			desc:       "check invalid subject set, missing subject set identifier",
			subjectSet: "namespace:#relation",
			result:     false,
		},
		{
			desc:       "check invalid subject set, missing object field",
			subjectSet: "namespace:object",
			result:     false,
		},
		{
			desc:       "check invalid subject set, unexpected object field",
			subjectSet: "namespace:object@relation",
			result:     false,
		},
	}

	for _, tc := range cases {
		iss := isSubjectSet(tc.subjectSet)
		assert.Equal(t, iss, tc.result, fmt.Sprintf("%s expected to be %v, got %v\n", tc.desc, tc.result, iss))
	}

}

func TestGetSubject(t *testing.T) {
	p1 := auth.PolicyReq{Subject: "subject", Object: "object", Relation: "relation"}
	s1 := getSubject(p1)
	ref1 := s1.GetRef()
	_, ok := ref1.(*acl.Subject_Id)
	assert.True(t, ok, fmt.Errorf("subject reference of %#v is expected to be (*acl.Subject_Id), got %T", p1, ref1))

	p2 := auth.PolicyReq{Subject: "members:group#access", Object: "object", Relation: "relation"}
	s2 := getSubject(p2)
	ref2 := s2.GetRef()
	_, ok = ref2.(*acl.Subject_Set)
	assert.True(t, ok, fmt.Errorf("subject reference of %#v is expected to be (*acl.Subject_Set), got %T", p2, ref2))
}
