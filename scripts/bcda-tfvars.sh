#!/bin/sh

#
# Helpful script to populate ECS image values into terraform.tfvars when running terraform locally
#
# This script assumes you already have the following variables defined in terraform.tfvars:
#    - api_image_tag - this value will be populated with the image tag of the running API ECS task
#    - ssas_image_tag - this value will be populated with the image tag of the running SSAS ECS task
#    - worker_image_tag - this value will be populated with the image tag of the running worker ECS task

set -e
if [[ -z "$1" ]]; then
    echo "Usage: $0 <environment>" 1>&2
    exit 1
fi
ENV=$1

pushd $(git rev-parse --show-toplevel)/terraform/$ENV
trap popd ERR

API_TASK_ARN=$(aws ecs list-tasks --cluster=bcda-$ENV --service-name=bcda-$ENV-api --query "taskArns[0]" --output text)
API_IMAGE_ARN=$(aws ecs describe-tasks --cluster=bcda-$ENV --tasks=$API_TASK_ARN --query "tasks[0].containers[?name == 'api'].image" --output=text)
API_IMAGE_TAG=${API_IMAGE_ARN##*:}

SSAS_TASK_ARN=$(aws ecs list-tasks --cluster=bcda-$ENV --service-name=bcda-$ENV-ssas --query "taskArns[0]" --output text)
SSAS_IMAGE_ARN=$(aws ecs describe-tasks --cluster=bcda-$ENV --tasks=$SSAS_TASK_ARN --query "tasks[0].containers[?name == 'ssas'].image" --output=text)
SSAS_IMAGE_TAG=${SSAS_IMAGE_ARN##*:}

WORKER_TASK_ARN=$(aws ecs list-tasks --cluster=bcda-$ENV --service-name=bcda-$ENV-worker --query "taskArns[0]" --output text)
WORKER_IMAGE_ARN=$(aws ecs describe-tasks --cluster=bcda-$ENV --tasks=$WORKER_TASK_ARN --query "tasks[0].containers[?name == 'worker'].image" --output=text)
WORKER_IMAGE_TAG=${WORKER_IMAGE_ARN##*:}

sed -i '.bak' \
    -e "s/^api_image_tag.*$/api_image_tag=\"$API_IMAGE_TAG\"/" \
    -e "s/^ssas_image_tag.*$/ssas_image_tag=\"$SSAS_IMAGE_TAG\"/" \
    -e "s/^worker_image_tag.*$/worker_image_tag=\"$WORKER_IMAGE_TAG\"/" \
    terraform.tfvars
