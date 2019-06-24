#!/bin/ash

if [ -n "$UI_PORT" ]; then
    sed  -e "s/UI_PORT/$UI_PORT/" /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf
else
    sed  -e "s/UI_PORT/3000/" /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf
fi

exec nginx -g "daemon off;"
