#!/bin/bash
SUDO=""

podman_exists() {
    local name="$1"

    (set -o pipefail && podman ps --all | grep "$name" >/dev/null)
}

podman_stop() {
    local name="$1"

    podman stop "$name" 2>/dev/null
}

podman_restart() {
    local name="$1"

    podman stop "$name" >/dev/null || return 1
    podman start "$name" >/dev/null || return 1
}

podman_rm() {
    local name="$1"

    if ! run_status=$(set -o pipefail && podman inspect -t container "$name" 2>/dev/null | jq .[0].State.Status); then
        echo "not present"
        return 0
    fi

    if [[ $run_status =~ running ]]; then
        podman stop "$name" >/dev/null || return 1
    fi

    podman rm "$name" >/dev/null || return 1
    echo "removed"
}

podman_isrunning() {
    local name="$1"

    run_status=$(set -o pipefail && podman inspect "$name" 2>/dev/null | jq .[0].State.Running) || return 1
    [[ "$run_status" =~ true ]] || return 1 # be explicit
}

podman_isrunning_logs() {
    local name="$1"

    podman_isrunning "$name" || ("" podman logs "$name" && return 1)
}
