#!/bin/bash
#
# This script provides a simple way to wait for a docker service to be healthy
# It is intended to be called from the Makefile to facilitate easier docker command orchestration
#

SERVICE=
FILE=
INTERVAL=2
TIMEOUT=60

usage() {
    echo "Usage: $0 [-f FILE] [-i INTERVAL] [-t TIMEOUT] <service>" >&2
    echo " -f : Specify alternate compose file" >&2
    echo " -i : Specify interval to check service health status in seconds (defaults to 2)" >&2
    echo " -t : Specify timeout in seconds (defaults to 60)" >&2
    exit 1
}

# Get inputs and set configuration
while getopts ":f:i:t:" opt; do
    case $opt in
        f)
            FILE=$OPTARG
            ;;
        i)
            INTERVAL=$OPTARG
            ;;
        t)
            TIMEOUT=$OPTARG
            ;;
        :)
            echo "Error: Option -$OPTARG requires an argument." >&2
            usage
            ;;
        \?)
            echo "Error: Invalid option: -$OPTARG" >&2
            usage
            ;;
    esac
done

shift $((OPTIND - 1))
SERVICE=$@
if [[ -z "$SERVICE" ]]; then
    usage
fi


#Wait for service
echo "Waiting for service '$SERVICE' to be healthy (timeout: ${TIMEOUT}s)..."

start_time=$(date +%s)
while true; do
    current_time=$(date +%s)
    elapsed_time=$((current_time - start_time))

    if [[ $elapsed_time -ge $TIMEOUT ]]; then
        echo "Timeout reached. Service '$SERVICE' did not become healthy within $TIMEOUT seconds."
        echo "Current status:"
        docker inspect -f '{{.State.Health.Status}}' $(docker compose ${FILE:+-f }${FILE} ps -q "$SERVICE")
        exit 1
    fi

    # Get the health status using docker inspect
    HEALTH_STATUS=$(docker inspect -f '{{.State.Health.Status}}' $(docker compose ${FILE:+-f }${FILE} ps -q "$SERVICE") 2>/dev/null)

    if [[ "$HEALTH_STATUS" == "healthy" ]]; then
        echo "Service '$SERVICE' is healthy."
        exit 0
    elif [[ "$HEALTH_STATUS" == "unhealthy" ]]; then
        echo "Service '$SERVICE' is unhealthy. Exiting."
        exit 1
    elif [[ "$HEALTH_STATUS" == "starting" ]]; then
        echo "Service '$SERVICE' is starting, waiting..."
    elif [[ "$HEALTH_STATUS" == "" ]]; then
        # Handle cases where service name might be wrong or no healthcheck configured
        echo "Could not get health status for '$SERVICE'. Check if service name is correct and healthcheck is configured."
        exit 1
    fi

    sleep $INTERVAL
done