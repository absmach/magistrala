// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"time"

	api "github.com/absmach/magistrala/api/http"
	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
)

const atomAlarmListLimit uint64 = 1000

type atomResourceStore interface {
	CreateResource(ctx context.Context, resource atom.Resource) (atom.Resource, error)
	UpdateResource(ctx context.Context, id string, resource atom.Resource) (atom.Resource, error)
	GetResource(ctx context.Context, id string) (atom.Resource, error)
	ListResources(ctx context.Context, q atom.Query) (atom.ResourceList, error)
	DeleteResource(ctx context.Context, id string) error
}

type atomRepository struct {
	store atomResourceStore
}

var _ Repository = (*atomRepository)(nil)

func NewAtomRepository(store atomResourceStore) Repository {
	return &atomRepository{store: store}
}

func (repo *atomRepository) CreateAlarm(ctx context.Context, alarm Alarm) (Alarm, error) {
	ok, err := repo.shouldCreateAlarm(ctx, alarm)
	if err != nil {
		return Alarm{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	if !ok {
		return Alarm{}, repoerr.ErrNotFound
	}

	res, err := repo.store.CreateResource(ctx, alarmProjection(alarm))
	if err != nil {
		return Alarm{}, atomRepositoryError(repoerr.ErrCreateEntity, err)
	}

	created, err := alarmFromResource(res)
	if err != nil {
		return Alarm{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return created, nil
}

func (repo *atomRepository) UpdateAlarm(ctx context.Context, alarm Alarm) (Alarm, error) {
	current, err := repo.loadAlarm(ctx, alarm.ID, "", repoerr.ErrUpdateEntity)
	if err != nil {
		return Alarm{}, err
	}

	updated := mergeAlarmUpdate(current, alarm)
	res, err := repo.store.UpdateResource(ctx, updated.ID, alarmProjection(updated))
	if err != nil {
		return Alarm{}, atomRepositoryError(repoerr.ErrUpdateEntity, err)
	}

	out, err := alarmFromResource(res)
	if err != nil {
		return Alarm{}, errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return out, nil
}

func (repo *atomRepository) ViewAlarm(ctx context.Context, alarmID, domainID string) (Alarm, error) {
	return repo.loadAlarm(ctx, alarmID, domainID, repoerr.ErrViewEntity)
}

func (repo *atomRepository) loadAlarm(ctx context.Context, alarmID, domainID string, wrapper error) (Alarm, error) {
	res, err := repo.store.GetResource(ctx, alarmID)
	if err != nil {
		return Alarm{}, atomRepositoryError(wrapper, err)
	}
	if res.Kind != atom.KindAlarm {
		return Alarm{}, repoerr.ErrNotFound
	}
	if domainID != "" && res.TenantID != domainID {
		return Alarm{}, repoerr.ErrNotFound
	}

	alarm, err := alarmFromResource(res)
	if err != nil {
		return Alarm{}, errors.Wrap(wrapper, err)
	}

	return alarm, nil
}

func (repo *atomRepository) ListAllAlarms(ctx context.Context, pm PageMetadata) (AlarmsPage, error) {
	items, err := repo.listAlarms(ctx, pm.DomainID, alarmPageAttributesContains(pm))
	if err != nil {
		return AlarmsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	filtered := make([]Alarm, 0, len(items))
	for _, alarm := range items {
		if matchesAlarmPage(alarm, pm) {
			filtered = append(filtered, alarm)
		}
	}
	sortAlarms(filtered, pm)

	total := uint64(len(filtered))
	start := pm.Offset
	if start > total {
		start = total
	}
	end := total
	if pm.Limit > 0 && start+pm.Limit < end {
		end = start + pm.Limit
	}

	return AlarmsPage{
		Offset: pm.Offset,
		Limit:  pm.Limit,
		Total:  total,
		Alarms: filtered[start:end],
	}, nil
}

func (repo *atomRepository) DeleteAlarm(ctx context.Context, id string) error {
	if _, err := repo.loadAlarm(ctx, id, "", repoerr.ErrRemoveEntity); err != nil {
		return err
	}
	if err := repo.store.DeleteResource(ctx, id); err != nil {
		return atomRepositoryError(repoerr.ErrRemoveEntity, err)
	}

	return nil
}

func (repo *atomRepository) shouldCreateAlarm(ctx context.Context, alarm Alarm) (bool, error) {
	items, err := repo.listAlarms(ctx, alarm.DomainID, alarmIdentityAttributes(alarm))
	if err != nil {
		return false, err
	}

	var latest *Alarm
	for i := range items {
		item := items[i]
		if !sameAlarmState(item, alarm) {
			continue
		}
		if !item.CreatedAt.IsZero() && !alarm.CreatedAt.IsZero() && item.CreatedAt.After(alarm.CreatedAt) {
			continue
		}
		if latest == nil || item.CreatedAt.After(latest.CreatedAt) {
			latest = &item
		}
	}

	if latest == nil {
		return alarm.Status == ActiveStatus, nil
	}
	if latest.Status != alarm.Status {
		return true, nil
	}
	return alarm.Status == ActiveStatus && latest.Severity != alarm.Severity, nil
}

func (repo *atomRepository) listAlarms(ctx context.Context, domainID string, attributes atom.Attributes) ([]Alarm, error) {
	var out []Alarm
	var offset uint64
	for {
		page, err := repo.store.ListResources(ctx, atom.Query{
			Kind:               atom.KindAlarm,
			TenantID:           domainID,
			AttributesContains: attributes,
			Limit:              atomAlarmListLimit,
			Offset:             offset,
		})
		if err != nil {
			return nil, err
		}

		for _, res := range page.Items {
			if res.Kind != atom.KindAlarm {
				continue
			}
			alarm, err := alarmFromResource(res)
			if err != nil {
				return nil, err
			}
			out = append(out, alarm)
		}

		offset += uint64(len(page.Items))
		if len(page.Items) == 0 || offset >= page.Total {
			break
		}
	}

	return out, nil
}

func alarmIdentityAttributes(alarm Alarm) atom.Attributes {
	attrs := atom.Attributes{}
	setAttrIfNotEmpty(attrs, "rule_id", alarm.RuleID)
	setAttrIfNotEmpty(attrs, "channel_id", alarm.ChannelID)
	setAttrIfNotEmpty(attrs, "client_id", alarm.ClientID)
	setAttrIfNotEmpty(attrs, "subtopic", alarm.Subtopic)
	setAttrIfNotEmpty(attrs, "measurement", alarm.Measurement)
	return attrs
}

func alarmPageAttributesContains(pm PageMetadata) atom.Attributes {
	attrs := atom.Attributes{}
	setAttrIfNotEmpty(attrs, "rule_id", pm.RuleID)
	setAttrIfNotEmpty(attrs, "channel_id", pm.ChannelID)
	setAttrIfNotEmpty(attrs, "client_id", pm.ClientID)
	setAttrIfNotEmpty(attrs, "subtopic", pm.Subtopic)
	setAttrIfNotEmpty(attrs, "measurement", pm.Measurement)
	if pm.Status != AllStatus {
		attrs["status"] = pm.Status.String()
	}
	if pm.Severity != math.MaxUint8 {
		attrs["severity"] = pm.Severity
	}
	setAttrIfNotEmpty(attrs, "assignee_id", pm.AssigneeID)
	setAttrIfNotEmpty(attrs, "updated_by", pm.UpdatedBy)
	setAttrIfNotEmpty(attrs, "assigned_by", pm.AssignedBy)
	setAttrIfNotEmpty(attrs, "acknowledged_by", pm.AcknowledgedBy)
	setAttrIfNotEmpty(attrs, "resolved_by", pm.ResolvedBy)
	return attrs
}

func setAttrIfNotEmpty(attrs atom.Attributes, key, value string) {
	if value != "" {
		attrs[key] = value
	}
}

func mergeAlarmUpdate(current, update Alarm) Alarm {
	if update.Status != 0 {
		current.Status = update.Status
	}
	if update.AssigneeID != "" {
		current.AssigneeID = update.AssigneeID
	}
	if !update.AssignedAt.IsZero() {
		current.AssignedAt = update.AssignedAt
	}
	if update.AssignedBy != "" {
		current.AssignedBy = update.AssignedBy
	}
	if update.AcknowledgedBy != "" {
		current.AcknowledgedBy = update.AcknowledgedBy
	}
	if !update.AcknowledgedAt.IsZero() {
		current.AcknowledgedAt = update.AcknowledgedAt
	}
	if update.ResolvedBy != "" {
		current.ResolvedBy = update.ResolvedBy
	}
	if !update.ResolvedAt.IsZero() {
		current.ResolvedAt = update.ResolvedAt
	}
	if update.Metadata != nil {
		current.Metadata = update.Metadata
	}
	current.UpdatedAt = update.UpdatedAt
	current.UpdatedBy = update.UpdatedBy

	return current
}

func alarmFromResource(res atom.Resource) (Alarm, error) {
	attrs := res.Attributes
	alarm := Alarm{
		ID:        res.ID,
		DomainID:  res.TenantID,
		CreatedAt: res.CreatedAt,
		UpdatedAt: res.UpdatedAt,
	}

	var err error
	alarm.RuleID = stringAttr(attrs, "rule_id")
	alarm.ChannelID = stringAttr(attrs, "channel_id")
	alarm.ClientID = stringAttr(attrs, "client_id")
	alarm.Subtopic = stringAttr(attrs, "subtopic")
	alarm.Measurement = stringAttr(attrs, "measurement")
	alarm.Value = stringAttr(attrs, "value")
	alarm.Unit = stringAttr(attrs, "unit")
	alarm.Threshold = stringAttr(attrs, "threshold")
	alarm.Cause = stringAttr(attrs, "cause")
	alarm.AssigneeID = stringAttr(attrs, "assignee_id")
	alarm.UpdatedBy = stringAttr(attrs, "updated_by")
	alarm.AssignedBy = stringAttr(attrs, "assigned_by")
	alarm.AcknowledgedBy = stringAttr(attrs, "acknowledged_by")
	alarm.ResolvedBy = stringAttr(attrs, "resolved_by")

	if alarm.Status, err = statusAttr(attrs); err != nil {
		return Alarm{}, err
	}
	if alarm.Severity, err = uint8Attr(attrs, "severity"); err != nil {
		return Alarm{}, err
	}
	if alarm.CreatedAt, err = timeAttr(attrs, "created_at", alarm.CreatedAt); err != nil {
		return Alarm{}, err
	}
	if alarm.UpdatedAt, err = timeAttr(attrs, "updated_at", alarm.UpdatedAt); err != nil {
		return Alarm{}, err
	}
	if alarm.AssignedAt, err = timeAttr(attrs, "assigned_at", time.Time{}); err != nil {
		return Alarm{}, err
	}
	if alarm.AcknowledgedAt, err = timeAttr(attrs, "acknowledged_at", time.Time{}); err != nil {
		return Alarm{}, err
	}
	if alarm.ResolvedAt, err = timeAttr(attrs, "resolved_at", time.Time{}); err != nil {
		return Alarm{}, err
	}
	alarm.Metadata, err = metadataAttr(attrs)
	if err != nil {
		return Alarm{}, err
	}

	return alarm, nil
}

func matchesAlarmPage(a Alarm, pm PageMetadata) bool {
	if pm.DomainID != "" && a.DomainID != pm.DomainID {
		return false
	}
	if pm.RuleID != "" && a.RuleID != pm.RuleID {
		return false
	}
	if pm.ChannelID != "" && a.ChannelID != pm.ChannelID {
		return false
	}
	if pm.Subtopic != "" && a.Subtopic != pm.Subtopic {
		return false
	}
	if pm.ClientID != "" && a.ClientID != pm.ClientID {
		return false
	}
	if pm.Measurement != "" && a.Measurement != pm.Measurement {
		return false
	}
	if pm.Status != AllStatus && a.Status != pm.Status {
		return false
	}
	if pm.Severity != math.MaxUint8 && a.Severity != pm.Severity {
		return false
	}
	if pm.AssigneeID != "" && a.AssigneeID != pm.AssigneeID {
		return false
	}
	if pm.UpdatedBy != "" && a.UpdatedBy != pm.UpdatedBy {
		return false
	}
	if pm.ResolvedBy != "" && a.ResolvedBy != pm.ResolvedBy {
		return false
	}
	if pm.AcknowledgedBy != "" && a.AcknowledgedBy != pm.AcknowledgedBy {
		return false
	}
	if pm.AssignedBy != "" && a.AssignedBy != pm.AssignedBy {
		return false
	}
	if !pm.CreatedFrom.IsZero() && a.CreatedAt.Before(pm.CreatedFrom) {
		return false
	}
	if !pm.CreatedTo.IsZero() && a.CreatedAt.After(pm.CreatedTo) {
		return false
	}

	return true
}

func sameAlarmState(a, b Alarm) bool {
	return a.DomainID == b.DomainID &&
		a.RuleID == b.RuleID &&
		a.ChannelID == b.ChannelID &&
		a.ClientID == b.ClientID &&
		a.Subtopic == b.Subtopic &&
		a.Measurement == b.Measurement
}

func sortAlarms(items []Alarm, pm PageMetadata) {
	desc := pm.Dir != api.AscDir
	sort.SliceStable(items, func(i, j int) bool {
		it, jt := alarmOrderTime(items[i], pm), alarmOrderTime(items[j], pm)
		if !it.Equal(jt) {
			if desc {
				return it.After(jt)
			}
			return it.Before(jt)
		}
		if desc {
			return items[i].ID > items[j].ID
		}
		return items[i].ID < items[j].ID
	})
}

func alarmOrderTime(a Alarm, pm PageMetadata) time.Time {
	if pm.Order == api.CreatedAtOrder {
		return a.CreatedAt
	}
	if !a.UpdatedAt.IsZero() {
		return a.UpdatedAt
	}
	return a.CreatedAt
}

func stringAttr(attrs atom.Attributes, key string) string {
	if attrs == nil {
		return ""
	}
	if v, ok := attrs[key].(string); ok {
		return v
	}
	return ""
}

func statusAttr(attrs atom.Attributes) (Status, error) {
	if v := stringAttr(attrs, "status"); v != "" {
		return ToStatus(v)
	}
	n, ok, err := numericAttr(attrs, "alarm_status")
	if err != nil || !ok {
		return ActiveStatus, err
	}
	return Status(n), nil
}

func uint8Attr(attrs atom.Attributes, key string) (uint8, error) {
	n, _, err := numericAttr(attrs, key)
	return n, err
}

func numericAttr(attrs atom.Attributes, key string) (uint8, bool, error) {
	if attrs == nil {
		return 0, false, nil
	}
	switch v := attrs[key].(type) {
	case nil:
		return 0, false, nil
	case uint8:
		return v, true, nil
	case uint64:
		if v > math.MaxUint8 {
			return 0, true, repoerr.ErrMalformedEntity
		}
		return uint8(v), true, nil
	case int:
		if v < 0 || v > math.MaxUint8 {
			return 0, true, repoerr.ErrMalformedEntity
		}
		return uint8(v), true, nil
	case float64:
		if v < 0 || v > math.MaxUint8 || v != math.Trunc(v) {
			return 0, true, repoerr.ErrMalformedEntity
		}
		return uint8(v), true, nil
	case json.Number:
		i, err := strconv.ParseUint(v.String(), 10, 8)
		if err != nil {
			return 0, true, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		return uint8(i), true, nil
	default:
		return 0, true, repoerr.ErrMalformedEntity
	}
}

func timeAttr(attrs atom.Attributes, key string, fallback time.Time) (time.Time, error) {
	if attrs == nil {
		return fallback, nil
	}
	switch v := attrs[key].(type) {
	case nil:
		return fallback, nil
	case string:
		if v == "" {
			return time.Time{}, nil
		}
		t, err := time.Parse(time.RFC3339Nano, v)
		if err != nil {
			return time.Time{}, errors.Wrap(repoerr.ErrMalformedEntity, err)
		}
		return t, nil
	case time.Time:
		return v, nil
	default:
		return time.Time{}, repoerr.ErrMalformedEntity
	}
}

func metadataAttr(attrs atom.Attributes) (Metadata, error) {
	if attrs == nil || attrs["metadata"] == nil {
		return nil, nil
	}
	switch v := attrs["metadata"].(type) {
	case map[string]any:
		return Metadata(v), nil
	case atom.Attributes:
		return Metadata(v), nil
	default:
		return nil, repoerr.ErrMalformedEntity
	}
}

func atomRepositoryError(wrapper, err error) error {
	switch {
	case atom.IsNotFound(err) || errors.Contains(err, repoerr.ErrNotFound):
		return repoerr.ErrNotFound
	case atom.IsConflict(err) || errors.Contains(err, repoerr.ErrConflict):
		return errors.Wrap(repoerr.ErrConflict, err)
	default:
		return errors.Wrap(wrapper, err)
	}
}
