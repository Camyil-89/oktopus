#!/bin/sh
set -eu

workers=${OKTOPUS_NGINX_WORKERS:-2}
case "$workers" in
    auto) ;;
    0*|*[!0-9]*)
        echo "OKTOPUS_NGINX_WORKERS must be a positive integer or auto" >&2
        exit 1
        ;;
esac

sed -i "s/^worker_processes[[:space:]].*;/worker_processes  $workers;/" /etc/nginx/nginx.conf
