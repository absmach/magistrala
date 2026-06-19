// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

const (
	atomActionRead      = "read"
	atomActionWrite     = "write"
	atomActionDelete    = "delete"
	atomActionManage    = "manage"
	atomActionPublish   = "publish"
	atomActionSubscribe = "subscribe"
	atomActionExecute   = "execute"
)

const (
	atomStatusActive    = "active"
	atomStatusInactive  = "inactive"
	atomStatusEnabled   = "enabled"
	atomStatusDisabled  = "disabled"
	atomStatusFrozen    = "frozen"
	atomStatusSuspended = "suspended"
	atomStatusDeleted   = "deleted"
)

const (
	atomKindDevice = "device"
	atomKindGroup  = "group"
	atomKindHuman  = "human"
)

const (
	atomObjectKindEntity   = "entity"
	atomObjectKindGroup    = "group"
	atomObjectKindResource = "resource"
	atomObjectKindTenant   = "tenant"
)

const atomScopeModeObject = "object"

const atomGraphQLPath = "/graphql"
