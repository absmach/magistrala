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
	relation report_generate: role#member | team#member\
\
	relation rule_create: role#member | team#member\
	relation rule_update: role#member | team#member\
	relation rule_read: role#member | team#member\
	relation rule_delete: role#member | team#member\
\
	relation alarm_create: role#member | team#member\
	relation alarm_update: role#member | team#member\
	relation alarm_read: role#member | team#member\
	relation alarm_delete: role#member | team#member' /tmp/modified-supermq.zed

sed -i '/permission group_view_role_users_permission = group_view_role_users + team->group_view_role_users + organization->admin/a\
\
	// Magistrala-specific permissions\
	permission report_create_permission = report_create + organization->admin\
	permission report_update_permission = report_update + organization->admin\
	permission report_read_permission = report_read + organization->admin\
	permission report_delete_permission = report_delete + organization->admin\
	permission report_generate_permission = report_generate + organization->admin\
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

cat /tmp/modified-supermq.zed /schema-magistrala.zed > /schemas/combined-schema.zed

