#!/bin/bash
# scripts/worker_sg_rule_imports.sh

set -e

ENV=${1:?Usage: ./worker_sg_rule_imports.sh <env>}

# --- Lookups ---

VPC_ID=$(aws ec2 describe-vpcs \
  --filters "Name=tag:Name,Values=bcda-east-${ENV}" \
  --query "Vpcs[0].VpcId" \
  --output text)

WORKER_SG_ID=$(aws ec2 describe-security-groups \
  --filters \
    "Name=group-name,Values=bcda-worker-${ENV}" \
    "Name=vpc-id,Values=${VPC_ID}" \
  --query "SecurityGroups[0].GroupId" \
  --output text)

ALB_SG_NAME=$([ "$ENV" = "sandbox" ] && echo "bcda-app-lb-${ENV}" || echo "app-alb-${ENV}")

ALB_SG_ID=$(aws ec2 describe-security-groups \
  --filters \
    "Name=group-name,Values=${ALB_SG_NAME}" \
    "Name=vpc-id,Values=${VPC_ID}" \
  --query "SecurityGroups[0].GroupId" \
  --output text)

DB_SG_ID=$(aws ec2 describe-security-groups \
  --filters \
    "Name=group-name,Values=bcda-${ENV}-db" \
    "Name=vpc-id,Values=${VPC_ID}" \
  --query "SecurityGroups[0].GroupId" \
  --output text)

PLATFORM_CIDR=$(aws ssm get-parameter \
  --name "/cdap/sensitive/mgmt-vpc/cidr" \
  --with-decryption \
  --query "Parameter.Value" \
  --output text)

echo "VPC ID:      ${VPC_ID}" >&2
echo "Worker SG:   ${WORKER_SG_ID}" >&2
echo "ALB SG:      ${ALB_SG_ID}" >&2
echo "DB SG:       ${DB_SG_ID}" >&2
echo "" >&2

find_rule() {
  local GROUP_ID=$1
  local PORT=$2
  local CIDR=$3
  local EGRESS=${4:-false}

  aws ec2 describe-security-group-rules \
    --filters "Name=group-id,Values=${GROUP_ID}" \
    --query "SecurityGroupRules[?IsEgress==\`${EGRESS}\` && FromPort==\`${PORT}\` && CidrIpv4=='${CIDR}'].SecurityGroupRuleId | [0]" \
    --output text
}

find_rule_by_sg() {
  local GROUP_ID=$1
  local PORT=$2
  local SOURCE_SG=$3
  local EGRESS=${4:-false}

  aws ec2 describe-security-group-rules \
    --filters "Name=group-id,Values=${GROUP_ID}" \
    --query "SecurityGroupRules[?IsEgress==\`${EGRESS}\` && FromPort==\`${PORT}\` && ReferencedGroupInfo.GroupId=='${SOURCE_SG}'].SecurityGroupRuleId | [0]" \
    --output text
}

print_import() {
  local TO=$1
  local ID=$2

  if [ -n "$ID" ] && [ "$ID" != "None" ] && [ "$ID" != "" ]; then
    echo "import {"
    echo "  to = ${TO}"
    echo "  id = \"${ID}\""
    echo "}"
    echo ""
  else
    echo "# No matching rule found for ${TO}" >&2
  fi
}

echo "Processing worker_sg ingress rules..." >&2
echo "# --- worker_sg ingress rules ---"
echo ""

print_import \
  "aws_vpc_security_group_ingress_rule.worker_sg_ingress_01" \
  "$(find_rule_by_sg $WORKER_SG_ID 3000 $ALB_SG_ID)"

print_import \
  "aws_vpc_security_group_ingress_rule.peering_worker_sg_ingress" \
  "$(find_rule $WORKER_SG_ID 22 $PLATFORM_CIDR)"

echo "Processing worker_sg egress rules..." >&2
echo "# --- worker_sg egress rules ---"
echo ""

print_import \
  "aws_vpc_security_group_egress_rule.worker_sg_egress" \
  "$(find_rule $WORKER_SG_ID -1 '0.0.0.0/0' true)"


echo "Processing db ingress from worker_sg..." >&2
echo "# --- db ingress from worker ---"
echo ""

print_import \
  "aws_vpc_security_group_ingress_rule.db_ingress_from_worker" \
  "$(find_rule_by_sg $DB_SG_ID 5432 $WORKER_SG_ID)"

echo "Done. Review before applying." >&2
