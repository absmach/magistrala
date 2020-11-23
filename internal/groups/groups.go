package groups

import (
	"context"
	"time"
)

type Member interface{}
type Metadata map[string]interface{}

type Group struct {
	ID          string
	OwnerID     string
	ParentID    string
	Name        string
	Description string
	Metadata    Metadata
	// Indicates a level in hierarchy from first group node.
	// For a root node level is 1.
	Level int
	// Path is a path in a tree, consisted of group names
	// parentName.childrenName1.childrenName2 .
	Path      string
	Children  []*Group
	CreatedAt time.Time
	UpdatedAt  time.Time
}

type PageMetadata struct {
	Total  uint64
	Offset uint64
	Limit  uint64
	Name   string
}

type GroupPage struct {
	PageMetadata
	Groups []Group
}

type MemberPage struct {
	PageMetadata
	Members []Member
}

type Service interface {
	// CreateGroup creates new  group.
	CreateGroup(ctx context.Context, token string, g Group) (string, error)

	// UpdateGroup updates the group identified by the provided ID.
	UpdateGroup(ctx context.Context, token string, g Group) (Group, error)

	// ViewGroup retrieves data about the group identified by ID.
	ViewGroup(ctx context.Context, token, id string) (Group, error)

	// ListGroups retrieves groups.
	ListGroups(ctx context.Context, token string, level uint64, m Metadata) (GroupPage, error)

	// ListChildren retrieves groups that are children to group identified by parentID
	ListChildren(ctx context.Context, token, parentID string, level uint64, m Metadata) (GroupPage, error)

	// ListParents retrieves groups that are parent to group identified by childID.
	ListParents(ctx context.Context, token, childID string, level uint64, m Metadata) (GroupPage, error)

	// ListMembers retrieves everything that is assigned to a group identified by groupID.
	ListMembers(ctx context.Context, token, groupID string, offset, limit uint64, m Metadata) (MemberPage, error)

	// ListMemberships retrieves all groups for member that is identified with memberID belongs to.
	ListMemberships(ctx context.Context, token, memberID string, offset, limit uint64, m Metadata) (GroupPage, error)

	// RemoveGroup removes the group identified with the provided ID.
	RemoveGroup(ctx context.Context, token, id string) error

	// Assign adds  member with memberID into the group identified by groupID.
	Assign(ctx context.Context, token, memberID, groupID string) error

	// Unassign removes member with memberID from group identified by groupID.
	Unassign(ctx context.Context, token, memberID, groupID string) error
}

type Repository interface {
	// Save group
	Save(ctx context.Context, g Group) (Group, error)

	// Update a group
	Update(ctx context.Context, g Group) (Group, error)

	// Delete a group
	Delete(ctx context.Context, groupID string) error

	// RetrieveByID retrieves group by its id
	RetrieveByID(ctx context.Context, id string) (Group, error)

	// RetrieveAll retrieves all groups.
	RetrieveAll(ctx context.Context, level uint64, m Metadata) (GroupPage, error)

	// RetrieveAllParents retrieves all groups that are ancestors to the group with given groupID.
	RetrieveAllParents(ctx context.Context, groupID string, level uint64, m Metadata) (GroupPage, error)

	// RetrieveAllChildren retrieves all children from group with given groupID up to the hierarchy level.
	RetrieveAllChildren(ctx context.Context, groupID string, level uint64, m Metadata) (GroupPage, error)

	// Retrieves list of groups that member belongs to
	Memberships(ctx context.Context, memberID string, offset, limit uint64, m Metadata) (GroupPage, error)

	// Members retrieves everything that is assigned to a group identified by groupID.
	Members(ctx context.Context, groupID string, offset, limit uint64, m Metadata) (MemberPage, error)

	// Assign adds member to group.
	Assign(ctx context.Context, memberID, groupID string) error

	// Unassign removes a member from a group
	Unassign(ctx context.Context, memberID, groupID string) error
}
