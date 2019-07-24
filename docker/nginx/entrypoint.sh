#!/bin/ash

if [ -n "$MF_UI_PORT" ]; then
    sed  -e "s/MF_UI_PORT/$MF_UI_PORT/" /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf
else
    sed  -e "s/MF_UI_PORT/3000/" /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf
fi

exec nginx -g "daemon off;"
