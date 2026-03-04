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

sed -i '/definition domain/,/^definition / {
	/relation group_view_role_users: role#member | team#member/a\
\
	// Magistrala-specific relations\
	relation report_create: role#member | team#member\
	relation report_update: role#member | team#member\
	relation report_read: role#member | team#member\
	relation report_delete: role#member | team#member\
\
	relation report_manage_role: role#member | team#member\
	relation report_add_role_users: role#member | team#member\
	relation report_remove_role_users: role#member | team#member\
	relation report_view_role_users: role#member | team#member\
\
	relation rule_create: role#member | team#member\
	relation rule_update: role#member | team#member\
	relation rule_read: role#member | team#member\
	relation rule_delete: role#member | team#member\
\
	relation rule_manage_role: role#member | team#member\
	relation rule_add_role_users: role#member | team#member\
	relation rule_remove_role_users: role#member | team#member\
	relation rule_view_role_users: role#member | team#member\
\
	relation alarm_create: role#member | team#member\
	relation alarm_update: role#member | team#member\
	relation alarm_read: role#member | team#member\
	relation alarm_delete: role#member | team#member\
\
	relation alarm_manage_role: role#member | team#member\
	relation alarm_add_role_users: role#member | team#member\
	relation alarm_remove_role_users: role#member | team#member\
	relation alarm_view_role_users: role#member | team#member
}' /tmp/modified-supermq.zed

sed -i '/definition team/,/^definition / {
	/relation group_view_role_users: role#member | team#member/a\
\
	relation alarm_create: role#member | team#member\
	relation alarm_update: role#member | team#member\
	relation alarm_read: role#member | team#member\
	relation alarm_delete: role#member | team#member\
\
	relation alarm_manage_role: role#member | team#member\
	relation alarm_add_role_users: role#member | team#member\
	relation alarm_remove_role_users: role#member | team#member\
	relation alarm_view_role_users: role#member | team#member\
\
	relation rule_create: role#member | team#member\
	relation rule_update: role#member | team#member\
	relation rule_read: role#member | team#member\
	relation rule_delete: role#member | team#member\
\
	relation rule_manage_role: role#member | team#member\
	relation rule_add_role_users: role#member | team#member\
	relation rule_remove_role_users: role#member | team#member\
	relation rule_view_role_users: role#member | team#member\
\
	relation report_create: role#member | team#member\
	relation report_update: role#member | team#member\
	relation report_read: role#member | team#member\
	relation report_delete: role#member | team#member\
\
	relation report_manage_role: role#member | team#member\
	relation report_add_role_users: role#member | team#member\
	relation report_remove_role_users: role#member | team#member\
	relation report_view_role_users: role#member | team#member
}' /tmp/modified-supermq.zed

sed -i '/permission group_view_role_users_permission = group_view_role_users + team->group_view_role_users + organization->admin/a\
\
	// Magistrala-specific permissions\
	permission report_create_permission = report_create + team->report_create + organization->admin\
	permission report_update_permission = report_update + team->report_update + organization->admin\
	permission report_read_permission = report_read + team->report_read + organization->admin\
	permission report_delete_permission = report_delete + team->report_delete + organization->admin\
\
	permission report_manage_role_permission = report_manage_role + team->report_manage_role + organization->admin\
	permission report_add_role_users_permission = report_add_role_users + team->report_add_role_users + organization->admin\
	permission report_remove_role_users_permission = report_remove_role_users + team->report_remove_role_users + organization->admin\
	permission report_view_role_users_permission = report_view_role_users + team->report_view_role_users + organization->admin\
\
	permission rule_create_permission = rule_create + team->rule_create + organization->admin\
	permission rule_update_permission = rule_update + team->rule_update + organization->admin\
	permission rule_read_permission = rule_read + team->rule_read + organization->admin\
	permission rule_delete_permission = rule_delete + team->rule_delete + organization->admin\
\
	permission rule_manage_role_permission = rule_manage_role + team->rule_manage_role + organization->admin\
	permission rule_add_role_users_permission = rule_add_role_users + team->rule_add_role_users + organization->admin\
	permission rule_remove_role_users_permission = rule_remove_role_users + team->rule_remove_role_users + organization->admin\
	permission rule_view_role_users_permission = rule_view_role_users + team->rule_view_role_users + organization->admin\
\
	permission alarm_create_permission = alarm_create + team->alarm_create + organization->admin\
	permission alarm_update_permission = alarm_update + team->alarm_update + organization->admin\
	permission alarm_read_permission = alarm_read + team->alarm_read + organization->admin\
	permission alarm_delete_permission = alarm_delete + team->alarm_delete + organization->admin\
\
	permission alarm_manage_role_permission = alarm_manage_role + team->alarm_manage_role + organization->admin\
	permission alarm_add_role_users_permission = alarm_add_role_users + team->alarm_add_role_users + organization->admin\
	permission alarm_remove_role_users_permission = alarm_remove_role_users + team->alarm_remove_role_users + organization->admin\
	permission alarm_view_role_users_permission = alarm_view_role_users + team->alarm_view_role_users + organization->admin' /tmp/modified-supermq.zed

sed -i '/permission membership = read + update + enable + disable + delete +$/,/+ organization->admin$/ {
	/group_manage_role + group_add_role_users + group_remove_role_users + group_view_role_users + organization->admin/s/$/ +\
	report_create + report_update + report_read + report_delete +\
	report_manage_role + report_add_role_users + report_remove_role_users + report_view_role_users +\
	rule_create + rule_update + rule_read + rule_delete +\
	rule_manage_role + rule_add_role_users + rule_remove_role_users + rule_view_role_users +\
	alarm_create + alarm_update + alarm_read + alarm_delete +\
	alarm_manage_role + alarm_add_role_users + alarm_remove_role_users + alarm_view_role_users/
}' /tmp/modified-supermq.zed

cat /tmp/modified-supermq.zed /schema-magistrala.zed > /schemas/combined-schema.zed

cp /permission-supermq.yaml /tmp/modified-permission.yaml

sed -i '/list_groups: group_read_permission/a\    - create_alarms: alarm_create_permission\
    - list_alarms: alarm_read_permission\
    - create_rules: rule_create_permission\
    - list_rules: rule_read_permission\
    - create_reports: report_create_permission\
    - list_reports: report_read_permission' /tmp/modified-permission.yaml

cat /permission-magistrala.yaml >> /tmp/modified-permission.yaml

cp /tmp/modified-permission.yaml /schemas/permission.yaml
cp /tmp/modified-permission.yaml /schemas/permission-combined.yaml
