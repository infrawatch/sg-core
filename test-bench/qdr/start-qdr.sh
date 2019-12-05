#!/bin/bash

usage() {
    cat <<-EOM
    Manage a Prometheus container runtime

    Usage:
        $(basename "$0") [-h] start|stop|remove
            start    - Start the Prometheus container
            stop     - Stop the Prometheus container
            remove   - Stop and remove the Prometheus container
            restart  - Restart Prometheus to reload config files

    Options
EOM
    exit 0
}

# shellcheck disable=SC1091
source "../common/podman.sh"

CONTAINER_NAME="qdrouterd"

VERBOSE="false"
export VERBOSE

while getopts ":h" opt; do
    case ${opt} in
    h)
        usage
        exit 0
        ;;
    \?)
        echo "Invalid Option: -$OPTARG" 1>&2
        exit 1
        ;;
    esac
done

if [ "$#" -ne 1 ]; then
    usage
fi

COMMAND=$1
shift

case "$COMMAND" in
start)
    if ! cid=$(podman run -d -p 5672:5672 --network=host --name "$CONTAINER_NAME" quay.io/interconnectedcloud/qdrouterd:1.8.0); then
        printf "Could not start %s container!\n" "$CONTAINER_NAME"
        exit 1
    fi
    podman_isrunning_logs "$CONTAINER_NAME" && printf "Started %s as %s...\n" "$CONTAINER_NAME" "$cid"
    ;;
restart)
    podman_restart "$CONTAINER_NAME" && printf "Restarted %s\n" "$CONTAINER_NAME" || exit 1 
    ;;
stop)
    podman_stop "$CONTAINER_NAME" && printf "Stopped %s\n" "$CONTAINER_NAME" || exit 1
    ;;
remove)
    status=$(podman_rm "$CONTAINER_NAME") && printf "%s %s\n" "$CONTAINER_NAME" "$status" || exit 1
    ;;
isrunning)
    if ! podman_isrunning "$CONTAINER_NAME"; then
        printf "%s is NOT running...\n" "$CONTAINER_NAME"
        exit 1
    else
        printf "%s is running...\n" "$CONTAINER_NAME"
    fi
    ;;
*)
    echo "Unknown command: ${COMMAND}"
    usage
    ;;
esac
