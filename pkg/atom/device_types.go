// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"errors"
	"net/http"
	"sort"
)

// ErrDeviceTypeVersionNotBindable means a bind named a version that is not
// active. Atom only rejects this when it picks the version itself; when the
// caller names one explicitly it binds without checking status, so this client
// checks instead.
var ErrDeviceTypeVersionNotBindable = errors.New("atom: device type version is not active")

// ErrDeviceTypeNotBindable means a bind named a device type that is not
// active. Atom enforces this too; checking here turns it into a typed error.
var ErrDeviceTypeNotBindable = errors.New("atom: device type is not active")

// Atom's Profile carries these for every device type. object_kind narrows it
// to entity profiles and kind to devices, which is also what makes a gateway
// type an ordinary device type rather than a separate catalogue.
const (
	deviceTypeObjectKind = atomObjectKindEntity
	deviceTypeKind       = atomKindDevice
)

// Atom names this concept "profile" on the wire. Aliases map it back to Device
// Type so nothing above this file has to know.
const (
	deviceTypeFields        = `id tenant_id: tenantId key name: displayName description status created_at: createdAt updated_at: updatedAt`
	deviceTypeVersionFields = `id device_type_id: profileId version json_schema: jsonSchema ui_schema: uiSchema status created_at: createdAt`
)

// CreateDeviceType registers a device type. It creates the type only — add a
// capability document with CreateDeviceTypeVersion, which is a separate call
// because Atom has no deleteProfile mutation, so a combined create could not
// roll back a half-built type.
//
// TenantID is required. Atom also supports tenant-less global types, but its
// profiles query cannot ask for "this tenant plus global" — passing no tenant
// returns every tenant's types — so Magistrala does not expose global types
// and always scopes to a tenant.
func (c *Client) CreateDeviceType(ctx context.Context, deviceType DeviceType) (DeviceType, error) {
	if deviceType.TenantID == "" {
		return DeviceType{}, Error{StatusCode: http.StatusBadRequest, Message: "device type requires a tenant"}
	}
	if deviceType.Key == "" {
		return DeviceType{}, Error{StatusCode: http.StatusBadRequest, Message: "device type requires a key"}
	}

	var out struct {
		CreateProfile DeviceType `json:"createProfile"`
	}
	err := c.graphQL(ctx, `mutation CreateDeviceType($input: CreateProfileInput!) {
		createProfile(input: $input) { `+deviceTypeFields+` }
	}`, map[string]any{atomInputKeyInput: deviceTypeCreateInput(deviceType)}, &out)
	return out.CreateProfile, err
}

func (c *Client) GetDeviceType(ctx context.Context, id string) (DeviceType, error) {
	var out struct {
		Profile DeviceType `json:"profile"`
	}
	err := c.graphQL(ctx, `query DeviceType($id: ID!) {
		profile(id: $id) { `+deviceTypeFields+` }
	}`, map[string]any{"id": id}, &out)
	return out.Profile, err
}

// ListDeviceTypes lists a tenant's device types. Global (tenant-less) types
// are not returned: Atom's filter is "this tenant" or "no filter at all", and
// no filter at all crosses tenants.
func (c *Client) ListDeviceTypes(ctx context.Context, q DeviceTypeQuery) (DeviceTypeList, error) {
	if q.TenantID == "" {
		return DeviceTypeList{}, Error{StatusCode: http.StatusBadRequest, Message: "listing device types requires a tenant"}
	}

	var out struct {
		Profiles DeviceTypeList `json:"profiles"`
	}
	err := c.graphQL(ctx, `query DeviceTypes($objectKind: String, $kind: String, $tenantId: ID, $status: String, $limit: Int, $offset: Int) {
		profiles(objectKind: $objectKind, kind: $kind, tenantId: $tenantId, status: $status, limit: $limit, offset: $offset) {
			total
			items { `+deviceTypeFields+` }
		}
	}`, deviceTypeQueryVariables(q), &out)
	return out.Profiles, err
}

// UpdateDeviceType changes a device type's name, description or status. Key,
// tenant and kind are fixed at creation.
//
// Setting Status to anything but active is how a whole device type is retired:
// Atom refuses new bindings to a non-active type and leaves bound devices
// alone.
func (c *Client) UpdateDeviceType(ctx context.Context, id string, deviceType DeviceType) (DeviceType, error) {
	var out struct {
		UpdateProfile DeviceType `json:"updateProfile"`
	}
	err := c.graphQL(ctx, `mutation UpdateDeviceType($id: ID!, $input: UpdateProfileInput!) {
		updateProfile(id: $id, input: $input) { `+deviceTypeFields+` }
	}`, map[string]any{"id": id, atomInputKeyInput: deviceTypeUpdateInput(deviceType)}, &out)
	return out.UpdateProfile, err
}

// CreateDeviceTypeVersion adds a version to a device type. It never mutates an
// existing version, so devices bound to earlier versions keep validating
// against exactly what they were created against.
//
// Version numbers off Version, or the next unused number when it is zero.
// Deriving it costs a read, and two concurrent creates can still collide — the
// unique constraint turns that into a conflict rather than a silent overwrite.
//
// JSONSchema and UISchema are rendered from Capabilities unless the caller
// supplies JSONSchema directly, which is the escape hatch for schemas the
// capability model cannot express.
//
// Status defaults to active. Atom has no mutation to change it afterwards, so
// a version's status is decided here and only here: publish as
// DeviceTypeVersionStatusDraft to stage a version without making it bindable.
func (c *Client) CreateDeviceTypeVersion(ctx context.Context, deviceTypeID string, version DeviceTypeVersion) (DeviceTypeVersion, error) {
	if deviceTypeID == "" {
		return DeviceTypeVersion{}, Error{StatusCode: http.StatusBadRequest, Message: "device type version requires a device type"}
	}

	if version.JSONSchema == nil {
		jsonSchema, uiSchema, err := BuildCapabilitySchema(version.Capabilities)
		if err != nil {
			return DeviceTypeVersion{}, err
		}
		version.JSONSchema = jsonSchema
		if version.UISchema == nil {
			version.UISchema = uiSchema
		}
	}

	if version.Version == 0 {
		next, err := c.nextDeviceTypeVersion(ctx, deviceTypeID)
		if err != nil {
			return DeviceTypeVersion{}, err
		}
		version.Version = next
	}

	var out struct {
		CreateProfileVersion DeviceTypeVersion `json:"createProfileVersion"`
	}
	err := c.graphQL(ctx, `mutation CreateDeviceTypeVersion($profileId: ID!, $input: CreateProfileVersionInput!) {
		createProfileVersion(profileId: $profileId, input: $input) { `+deviceTypeVersionFields+` }
	}`, map[string]any{
		atomInputKeyProfileID: deviceTypeID,
		atomInputKeyInput:     deviceTypeVersionCreateInput(version),
	}, &out)
	if err != nil {
		return DeviceTypeVersion{}, err
	}
	return withCapabilities(out.CreateProfileVersion)
}

// ListDeviceTypeVersions returns every version of a device type, oldest
// first, each with its stored capability document parsed back out.
func (c *Client) ListDeviceTypeVersions(ctx context.Context, deviceTypeID string) ([]DeviceTypeVersion, error) {
	versions, err := c.listDeviceTypeVersionsRaw(ctx, deviceTypeID)
	if err != nil {
		return nil, err
	}
	for i, version := range versions {
		parsed, err := withCapabilities(version)
		if err != nil {
			return nil, err
		}
		versions[i] = parsed
	}
	return versions, nil
}

// listDeviceTypeVersionsRaw returns every version of a device type, oldest
// first, without parsing any stored capability document. Callers that only
// need a version's identity and status — picking the active one, checking
// whether a named one is bindable, computing the next version number — must
// use this instead of ListDeviceTypeVersions: parsing is not free, and an
// unrelated version's malformed capability document must not fail an
// operation that never reads it.
func (c *Client) listDeviceTypeVersionsRaw(ctx context.Context, deviceTypeID string) ([]DeviceTypeVersion, error) {
	var out struct {
		ProfileVersions []DeviceTypeVersion `json:"profileVersions"`
	}
	err := c.graphQL(ctx, `query DeviceTypeVersions($profileId: ID!) {
		profileVersions(profileId: $profileId) { `+deviceTypeVersionFields+` }
	}`, map[string]any{atomInputKeyProfileID: deviceTypeID}, &out)
	if err != nil {
		return nil, err
	}

	versions := out.ProfileVersions
	sort.Slice(versions, func(i, j int) bool { return versions[i].Version < versions[j].Version })
	return versions, nil
}

// activeDeviceTypeVersionRaw finds the version Atom binds a device to when
// none is named — the highest-numbered active one — without parsing its
// capability document. Callers that only need the version's id or status
// (BindDeviceType's empty-versionID path) must use this instead of
// ActiveDeviceTypeVersion, for the same reason listDeviceTypeVersionsRaw
// exists: a malformed stored declaration on the active version must not fail
// an operation that never reads it.
func (c *Client) activeDeviceTypeVersionRaw(ctx context.Context, deviceTypeID string) (DeviceTypeVersion, error) {
	versions, err := c.listDeviceTypeVersionsRaw(ctx, deviceTypeID)
	if err != nil {
		return DeviceTypeVersion{}, err
	}
	for i := len(versions) - 1; i >= 0; i-- {
		if versions[i].Status == DeviceTypeVersionStatusActive {
			return versions[i], nil
		}
	}
	return DeviceTypeVersion{}, Error{
		StatusCode: http.StatusNotFound,
		Message:    "device type " + deviceTypeID + " has no active version",
	}
}

// ActiveDeviceTypeVersion returns the version Atom binds a device to when none
// is named: the highest-numbered active one. It returns a not-found error when
// the type has no active version, which is also what Atom does on bind.
func (c *Client) ActiveDeviceTypeVersion(ctx context.Context, deviceTypeID string) (DeviceTypeVersion, error) {
	active, err := c.activeDeviceTypeVersionRaw(ctx, deviceTypeID)
	if err != nil {
		return DeviceTypeVersion{}, err
	}
	// Parsed only for the one version actually being returned, not for every
	// version scanned to find it.
	return withCapabilities(active)
}

// BindDeviceType binds an existing device to a device type, optionally to a
// named version. An empty versionID leaves the choice to Atom, which takes the
// highest-numbered active version.
//
// The type and version are checked before the write because Atom only checks
// them itself when it chooses the version — a caller-named version is bound
// whatever its status. Without this check, deprecating a version would not
// stop new bindings to it.
//
// Binding validates attributes against the *previously* bound version, not the
// new one: Atom's update path looks up the schema by the entity's current
// binding. The first write after a bind is what the new schema sees.
func (c *Client) BindDeviceType(ctx context.Context, deviceID, deviceTypeID, versionID string) (Entity, error) {
	if deviceID == "" || deviceTypeID == "" {
		return Entity{}, Error{StatusCode: http.StatusBadRequest, Message: "binding a device type requires a device and a device type"}
	}

	deviceType, err := c.GetDeviceType(ctx, deviceTypeID)
	if err != nil {
		return Entity{}, err
	}
	if deviceType.Status != DeviceTypeStatusActive {
		return Entity{}, ErrDeviceTypeNotBindable
	}

	if versionID == "" {
		// Atom's "leaves the choice to Atom" default only applies to a first
		// bind, with no previous version to fall back to. On an update it
		// instead coalesces a missing profileVersionId to the entity's
		// *currently* bound version (see entityUpdateInput) — so leaving
		// versionID empty here would silently keep whatever version the
		// device was already on, including one belonging to a different
		// device type, instead of picking this type's highest-numbered
		// active version the way the doc comment above promises. Resolving
		// it explicitly keeps both paths consistent.
		active, err := c.activeDeviceTypeVersionRaw(ctx, deviceTypeID)
		if err != nil {
			return Entity{}, err
		}
		versionID = active.ID
	} else {
		versions, err := c.listDeviceTypeVersionsRaw(ctx, deviceTypeID)
		if err != nil {
			return Entity{}, err
		}
		bindable := false
		for _, version := range versions {
			if version.ID != versionID {
				continue
			}
			bindable = version.Status == DeviceTypeVersionStatusActive
			break
		}
		if !bindable {
			return Entity{}, ErrDeviceTypeVersionNotBindable
		}
	}

	return c.UpdateEntity(ctx, deviceID, Entity{
		DeviceTypeID:        deviceTypeID,
		DeviceTypeVersionID: versionID,
	})
}

// nextDeviceTypeVersion is one past the highest existing version number.
func (c *Client) nextDeviceTypeVersion(ctx context.Context, deviceTypeID string) (int, error) {
	versions, err := c.listDeviceTypeVersionsRaw(ctx, deviceTypeID)
	if err != nil {
		return 0, err
	}
	next := 1
	for _, version := range versions {
		if version.Version >= next {
			next = version.Version + 1
		}
	}
	return next, nil
}

// withCapabilities fills in the declaration a stored version encodes, so
// callers never read json_schema themselves.
func withCapabilities(version DeviceTypeVersion) (DeviceTypeVersion, error) {
	capabilities, err := ParseCapabilityDocument(version.JSONSchema, version.UISchema)
	if err != nil {
		return DeviceTypeVersion{}, err
	}
	version.Capabilities = capabilities
	return version, nil
}

func deviceTypeCreateInput(deviceType DeviceType) map[string]any {
	input := map[string]any{
		atomInputKeyObjectKind: deviceTypeObjectKind,
		atomInputKeyKind:       deviceTypeKind,
		"key":                  deviceType.Key,
		"displayName":          deviceType.Name,
	}
	setIfNotEmpty(input, "tenantId", deviceType.TenantID)
	setIfNotEmpty(input, "description", deviceType.Description)
	setIfNotEmpty(input, atomAttributeStatus, deviceType.Status)
	return input
}

func deviceTypeUpdateInput(deviceType DeviceType) map[string]any {
	input := map[string]any{}
	setIfNotEmpty(input, "displayName", deviceType.Name)
	setIfNotEmpty(input, "description", deviceType.Description)
	setIfNotEmpty(input, atomAttributeStatus, deviceType.Status)
	return input
}

func deviceTypeVersionCreateInput(version DeviceTypeVersion) map[string]any {
	input := map[string]any{"version": version.Version}
	if version.JSONSchema != nil {
		input["jsonSchema"] = version.JSONSchema
	}
	if version.UISchema != nil {
		input["uiSchema"] = version.UISchema
	}
	setIfNotEmpty(input, atomAttributeStatus, version.Status)
	return input
}

func deviceTypeQueryVariables(q DeviceTypeQuery) map[string]any {
	vars := map[string]any{
		atomInputKeyObjectKind: deviceTypeObjectKind,
		atomInputKeyKind:       deviceTypeKind,
	}
	setIfNotEmpty(vars, "tenantId", q.TenantID)
	setIfNotEmpty(vars, atomAttributeStatus, q.Status)
	if q.Limit > 0 {
		vars["limit"] = int(q.Limit)
	}
	if q.Offset > 0 {
		vars["offset"] = int(q.Offset)
	}
	return vars
}
