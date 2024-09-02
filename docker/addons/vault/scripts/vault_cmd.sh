#!/usr/bin/bash
# Copyright (c) Abstract Machines
# SPDX-License-Identifier: Apache-2.0

vault() {
    if is_container_running "magistrala-vault"; then
        docker exec -it magistrala-vault vault "$@"
    else
        if which vault &> /dev/null; then
            $(which vault) "$@"
        else
            echo "magistrala-vault container or vault command not found. Please refer to the documentation: https://github.com/absmach/magistrala/blob/main/docker/addons/vault/README.md"
        fi
    fi
}

is_container_running() {
    local container_name="$1"
    if [ "$(docker inspect --format '{{.State.Running}}' "$container_name" 2>/dev/null)" = "true" ]; then
        return 0
    else
        return 1
    fi
}
