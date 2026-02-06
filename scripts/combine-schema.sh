#!/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

set -e

if [ ! -f /schema-supermq.zed ]; then 
    echo 'ERROR: /schema-supermq.zed not found'
    exit 1
fi

if [ ! -f /schema-magistrala.zed ]; then 
    echo 'ERROR: /schema-magistrala.zed not found'
    exit 1
fi

mkdir -p /schemas
cp /schema-supermq.zed /tmp/modified-supermq.zed

sed -i '/relation group_view_role_users: role#member | team#member/a\
\
	// Magistrala-specific relations\
	relation report_create: role#member | team#member\
	relation report_update: role#member | team#member\
	relation report_read: role#member | team#member\
	relation report_delete: role#member | team#member\
\
	relation rule_create: role#member | team#member\
	relation rule_update: role#member | team#member\
	relation rule_read: role#member | team#member\
	relation rule_delete: role#member | team#member\
\
	relation alarm_create: role#member | team#member\
	relation alarm_update: role#member | team#member\
	relation alarm_read: role#member | team#member\
	relation alarm_delete: role#member | team#member\
\
	// Domain-level operations for creating and listing Magistrala entities\
	relation create_alarms: role#member | team#member\
	relation list_alarms: role#member | team#member\
	relation create_rules: role#member | team#member\
	relation list_rules: role#member | team#member\
	relation create_reports: role#member | team#member\
	relation list_reports: role#member | team#member' /tmp/modified-supermq.zed

sed -i '/permission group_view_role_users_permission = group_view_role_users + team->group_view_role_users + organization->admin/a\
\
	// Magistrala-specific permissions\
	permission report_create_permission = report_create + organization->admin\
	permission report_update_permission = report_update + organization->admin\
	permission report_read_permission = report_read + organization->admin\
	permission report_delete_permission = report_delete + organization->admin\
\
	permission rule_create_permission = rule_create + organization->admin\
	permission rule_update_permission = rule_update + organization->admin\
	permission rule_read_permission = rule_read + organization->admin\
	permission rule_delete_permission = rule_delete + organization->admin\
\
	permission alarm_create_permission = alarm_create + organization->admin\
	permission alarm_update_permission = alarm_update + organization->admin\
	permission alarm_read_permission = alarm_read + organization->admin\
	permission alarm_delete_permission = alarm_delete + organization->admin' /tmp/modified-supermq.zed

sed -i '/permission membership = read + update + enable + disable + delete +$/,/+ organization->admin$/ {
	/group_manage_role + group_add_role_users + group_remove_role_users + group_view_role_users + organization->admin/s/$/ +\
	report_create + report_update + report_read + report_delete +\
	rule_create + rule_update + rule_read + rule_delete +\
	alarm_create + alarm_update + alarm_read + alarm_delete +\
	create_alarms + list_alarms + create_rules + list_rules + create_reports + list_reports/
}' /tmp/modified-supermq.zed

cat /tmp/modified-supermq.zed /schema-magistrala.zed > /schemas/combined-schema.zed

# Combine permission.yaml files
cp /permission-supermq.yaml /tmp/modified-permission.yaml

# Inject Magistrala domain operations into domains operations section
sed -i '/list_groups: group_read_permission/a\    - create_alarms: alarm_create_permission\
    - list_alarms: alarm_read_permission\
    - create_rules: rule_create_permission\
    - list_rules: rule_read_permission\
    - create_reports: report_create_permission\
    - list_reports: report_read_permission' /tmp/modified-permission.yaml

# Append Magistrala-specific entities (alarm, rule, report) from Magistrala permission.yaml
cat /permission-magistrala.yaml >> /tmp/modified-permission.yaml

# Copy to both locations for different services
cp /tmp/modified-permission.yaml /schemas/permission.yaml
cp /tmp/modified-permission.yaml /schemas/permission-combined.yaml

