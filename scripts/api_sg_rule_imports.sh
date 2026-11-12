#!/bin/bash
# scripts/api_sg_rule_imports.sh

set -e

ENV=${1:?Usage: ./api_sg_rule_imports.sh <env>}

is_prod() {
  [[ " prod " =~ " ${ENV} " ]]
}

# --- Lookups ---

VPC_ID=$(aws ec2 describe-vpcs \
  --filters "Name=tag:Name,Values=bcda-east-${ENV}" \
  --query "Vpcs[0].VpcId" \
  --output text)

APP_SG_ID=$(aws ec2 describe-security-groups \
  --filters \
    "Name=group-name,Values=bcda-api-${ENV}" \
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

WORKER_SG_ID=$(aws ec2 describe-security-groups \
  --filters \
    "Name=group-name,Values=bcda-worker-${ENV}" \
    "Name=vpc-id,Values=${VPC_ID}" \
  --query "SecurityGroups[0].GroupId" \
  --output text)

SSAS_ALB_SG_ID=$(aws ec2 describe-security-groups \
  --filters \
    "Name=group-name,Values=ssas-alb" \
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

echo "VPC ID:       ${VPC_ID}" >&2
echo "App SG:       ${APP_SG_ID}" >&2
echo "ALB SG:       ${ALB_SG_ID}" >&2
echo "Worker SG:    ${WORKER_SG_ID}" >&2
echo "SSAS ALB SG:  ${SSAS_ALB_SG_ID}" >&2
echo "DB SG:        ${DB_SG_ID}" >&2
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

find_rule_ipv6() {
  local GROUP_ID=$1
  local PORT=$2
  local CIDR=$3
  local EGRESS=${4:-false}

  aws ec2 describe-security-group-rules \
    --filters "Name=group-id,Values=${GROUP_ID}" \
    --query "SecurityGroupRules[?IsEgress==\`${EGRESS}\` && FromPort==\`${PORT}\` && CidrIpv6=='${CIDR}'].SecurityGroupRuleId | [0]" \
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

echo "Processing app_sg ingress rules..." >&2
echo "# --- worker_sg ingress rules ---"
echo ""

print_import \
  "aws_vpc_security_group_ingress_rule.app_sg_ingress_01" \
  "$(find_rule_by_sg $APP_SG_ID 3000 $ALB_SG_ID)"

if ! is_prod; then
  print_import \
    "aws_vpc_security_group_ingress_rule.app_sg_ingress_http_01[0]" \
    "$(find_rule_by_sg $APP_SG_ID 3001 $ALB_SG_ID)"
fi

print_import \
  "aws_vpc_security_group_ingress_rule.ssas_sg_ingress_public" \
  "$(find_rule_by_sg $APP_SG_ID 3003 $SSAS_ALB_SG_ID)"

print_import \
  "aws_vpc_security_group_ingress_rule.ssas_sg_ingress_admin" \
  "$(find_rule_by_sg $APP_SG_ID 3004 $SSAS_ALB_SG_ID)"

print_import \
  "aws_vpc_security_group_ingress_rule.peering_app_sg_ingress" \
  "$(find_rule $APP_SG_ID 22 $PLATFORM_CIDR)"

echo "Processing app_sg egress rules..." >&2
echo "# --- app_sg egress rules ---"
echo ""

print_import \
  "aws_vpc_security_group_egress_rule.app_sg_egress" \
  "$(find_rule $APP_SG_ID -1 '0.0.0.0/0' true)"

echo "Processing sg_alb_01 ingress rules..." >&2
echo "# --- sg_alb_01 ingress rules ---"
echo ""

print_import \
  "aws_vpc_security_group_ingress_rule.alb_ingress_01" \
  "$(find_rule $ALB_SG_ID 443 '0.0.0.0/0')"

print_import \
  "aws_vpc_security_group_ingress_rule.alb_ingress_01_ipv6" \
  "$(find_rule_ipv6 $ALB_SG_ID 443 '::/0')"

print_import \
  "aws_vpc_security_group_ingress_rule.alb_ingress_http_01" \
  "$(find_rule $ALB_SG_ID 80 '0.0.0.0/0')"

print_import \
  "aws_vpc_security_group_ingress_rule.alb_ingress_http_01_ipv6" \
  "$(find_rule_ipv6 $ALB_SG_ID 80 '::/0')"

echo "Processing sg_alb_01 egress rules..." >&2
echo "# --- sg_alb_01 egress rules ---"
echo ""

print_import \
  "aws_vpc_security_group_egress_rule.alb_app_egress" \
  "$(find_rule_by_sg $ALB_SG_ID 3000 $APP_SG_ID true)"

print_import \
  "aws_vpc_security_group_egress_rule.alb_worker_egress" \
  "$(find_rule_by_sg $ALB_SG_ID 3000 $WORKER_SG_ID true)"

echo "Processing db ingress from api..." >&2
echo "# --- db ingress from api ---"
echo ""

print_import \
  "aws_vpc_security_group_ingress_rule.db_ingress_from_api" \
  "$(find_rule_by_sg $DB_SG_ID 5432 $APP_SG_ID)"

echo "Done. Review before applying." >&2
