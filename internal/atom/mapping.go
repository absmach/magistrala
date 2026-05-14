// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

const (
	KindUser    = "user"
	KindClient  = "client"
	KindChannel = "channel"
	KindRule    = "rule"
	KindAlarm   = "alarm"
	KindReport  = "report"
)

func TenantFromFields(f ObjectFields) Tenant {
	return Tenant{
		ID:        f.ID,
		Name:      f.Name,
		Route:     f.Route,
		Tags:      cloneStrings(f.Tags),
		Status:    tenantStatus(f.Status),
		CreatedBy: f.CreatedBy,
		UpdatedBy: f.UpdatedBy,
		Attributes: compact(Attributes{
			"source":     "magistrala",
			"metadata":   cloneMap(f.Metadata),
			"created_at": timeString(f.CreatedAt),
			"updated_at": timeString(f.UpdatedAt),
		}),
	}
}

func EntityFromFields(f ObjectFields) Entity {
	return Entity{
		ID:       f.ID,
		Kind:     entityKind(f.Kind),
		Name:     f.Name,
		TenantID: f.TenantID,
		Status:   entityStatus(f.Status),
		Attributes: compact(Attributes{
			"source":           "magistrala",
			"magistrala_kind":  f.Kind,
			"tags":             cloneStrings(f.Tags),
			"metadata":         cloneMap(f.Metadata),
			"private_metadata": cloneMap(f.Private),
			"parent_group_id":  f.ParentID,
			"created_at":       timeString(f.CreatedAt),
			"updated_at":       timeString(f.UpdatedAt),
			"updated_by":       f.UpdatedBy,
		}),
	}
}

func tenantStatus(status string) string {
	switch status {
	case "enabled":
		return "active"
	case "disabled":
		return "inactive"
	case "freezed":
		return "frozen"
	case "deleted":
		return "deleted"
	default:
		return status
	}
}

func entityStatus(status string) string {
	switch status {
	case "enabled":
		return "active"
	case "disabled", "deleted":
		return "inactive"
	default:
		return status
	}
}

func entityKind(kind string) string {
	switch kind {
	case KindUser:
		return "human"
	case KindClient:
		return "device"
	default:
		return kind
	}
}

func GroupFromFields(f ObjectFields) Group {
	return Group{
		ID:          f.ID,
		Name:        f.Name,
		TenantID:    f.TenantID,
		Description: f.Description,
		ParentID:    f.ParentID,
		Status:      entityStatus(f.Status),
		Attributes: compact(Attributes{
			"source":     "magistrala",
			"parent_id":  f.ParentID,
			"tags":       cloneStrings(f.Tags),
			"metadata":   cloneMap(f.Metadata),
			"status":     f.Status,
			"created_at": timeString(f.CreatedAt),
			"updated_at": timeString(f.UpdatedAt),
			"updated_by": f.UpdatedBy,
		}),
	}
}

func ResourceFromFields(f ObjectFields) Resource {
	return Resource{
		ID:       f.ID,
		Kind:     f.Kind,
		Name:     f.Name,
		TenantID: f.TenantID,
		OwnerID:  f.OwnerID,
		Attributes: compact(Attributes{
			"source":          "magistrala",
			"status":          f.Status,
			"route":           f.Route,
			"parent_group_id": f.ParentID,
			"tags":            cloneStrings(f.Tags),
			"metadata":        cloneMap(f.Metadata),
			"created_at":      timeString(f.CreatedAt),
			"updated_at":      timeString(f.UpdatedAt),
			"updated_by":      f.UpdatedBy,
		}),
	}
}

func compact(attrs Attributes) Attributes {
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			if val == "" {
				delete(attrs, k)
			}
		case []string:
			if len(val) == 0 {
				delete(attrs, k)
			}
		case map[string]any:
			if len(val) == 0 {
				delete(attrs, k)
			}
		case nil:
			delete(attrs, k)
		}
	}
	return attrs
}

func cloneStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func timeString(t interface {
	IsZero() bool
	Format(string) string
}) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02T15:04:05.999999999Z07:00")
}
