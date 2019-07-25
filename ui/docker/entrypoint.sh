#!/bin/sh

if [ -n "$MF_UI_PORT" ]; then
    sed -i -e "s/MF_UI_PORT/$MF_UI_PORT/" /etc/nginx/conf.d/default.conf
else
    sed -i -e "s/MF_UI_PORT/3000/" /etc/nginx/conf.d/default.conf
fi

exec nginx -g "daemon off;"
