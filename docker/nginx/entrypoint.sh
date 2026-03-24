#!/bin/ash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

if [ ! -f /etc/nginx/snippets/mqtt-upstream.conf ] || [ ! -f /etc/nginx/snippets/mqtt-ws-upstream.conf ]; then
      echo "Missing MQTT upstream snippets; cannot start nginx." >&2
      exit 1
fi

envsubst '
    ${MG_NGINX_SERVER_NAME}
    ${MG_AUTH_HTTP_PORT}
    ${MG_DOMAINS_HTTP_PORT}
    ${MG_GROUPS_HTTP_PORT}
    ${MG_USERS_HTTP_PORT}
    ${MG_CLIENTS_HTTP_PORT}
    ${MG_CLIENTS_AUTH_HTTP_PORT}
    ${MG_CHANNELS_HTTP_PORT}
    ${MG_HTTP_ADAPTER_PORT}
    ${MG_NGINX_MQTT_PORT}
    ${MG_NGINX_MQTTS_PORT}
    ${MG_RE_HTTP_PORT}
    ${MG_ALARMS_HTTP_PORT}
    ${MG_REPORTS_HTTP_PORT}
    ${MG_NGINX_AMQP_PORT}' < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf

exec nginx -g "daemon off;"
