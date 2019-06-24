#!/bin/ash

if [ -n "$UI_PORT" ]; then
    sed -i -e "s/UI_PORT/$UI_PORT/" /etc/nginx/conf.d/default.conf
else
    sed -i -e "s/UI_PORT/3000/" /etc/nginx/conf.d/default.conf
fi

exec nginx -g "daemon off;"
