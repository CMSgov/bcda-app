#!/bin/bash
# scripts/ssas_sg_imports.sh

set -e

ENV=${1:?Usage: ./ssas_sg_imports.sh <env>}

# --- Lookups --- #

VPC_ID=$(aws ec2 describe-vpcs \
  --filters "Name=tag:Name,Values=bcda-east-${ENV}" \
  --query "Vpcs[0].VpcId" \
  --output text)

VPC_CIDR=$(aws ec2 describe-vpcs \
  --filters "Name=tag:Name,Values=bcda-east-${ENV}" \
  --query "Vpcs[0].CidrBlock" \
  --output text)

GHA_CIDRS=$(aws ssm get-parameter \
  --name "/bcda/${ENV}/infra/sensitive/ssas_gha_runners_cidr_blocks" \
  --with-decryption \
  --query "Parameter.Value" \
  --output text)

ACO_MS_CIDRS=$(aws ssm get-parameter \
  --name "/bcda/${ENV}/infra/sensitive/ssas_aco_ms_admin_cidr_blocks" \
  --with-decryption \
  --query "Parameter.Value" \
  --output text)

CIDRS_4I_PUBLIC=$(aws ssm get-parameter \
  --name "/bcda/${ENV}/infra/sensitive/ssas_4i_public_cidr_blocks" \
  --with-decryption \
  --query "Parameter.Value" \
  --output text)

CIDRS_4I_ADMIN=$(aws ssm get-parameter \
  --name "/bcda/${ENV}/infra/sensitive/ssas_4i_admin_cidr_blocks" \
  --with-decryption \
  --query "Parameter.Value" \
  --output text)

IHP_CIDRS=$(aws ssm get-parameter \
  --name "/bcda/${ENV}/infra/sensitive/ssas_ihp_cidr_blocks" \
  --with-decryption \
  --query "Parameter.Value" \
  --output text)

SSAS_ALB_SG_ID=$(aws ec2 describe-security-groups \
  --filters \
    "Name=group-name,Values=ssas-alb" \
    "Name=vpc-id,Values=${VPC_ID}" \
  --query "SecurityGroups[0].GroupId" \
  --output text)

APP_SG_ID=$(aws ec2 describe-security-groups \
  --filters \
    "Name=group-name,Values=bcda-api-${ENV}" \
    "Name=vpc-id,Values=${VPC_ID}" \
  --query "SecurityGroups[0].GroupId" \
  --output text)

echo "VPC ID:      ${VPC_ID}" >&2
echo "VPC CIDR:    ${VPC_CIDR}" >&2
echo "ssas_alb SG: ${SSAS_ALB_SG_ID}" >&2
echo "app_sg SG:   ${APP_SG_ID}" >&2
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
    --query "SecurityGroupRules[?IsEgress==\`${EGRESS}\` && ReferencedGroupInfo.GroupId=='${SOURCE_SG}'].SecurityGroupRuleId | [0]" \
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

print_cidr_imports() {
  local RESOURCE_NAME=$1
  local GROUP_ID=$2
  local PORT=$3
  local CIDRS=$4

  # Skip if CIDRS is empty
  [ -z "$CIDRS" ] && return

  IFS=',' read -ra CIDR_LIST <<< "$CIDRS"
  for CIDR in "${CIDR_LIST[@]}"; do
    CIDR=$(echo "$CIDR" | tr -d '[:space:]')
    [ -z "$CIDR" ] && continue
    RULE_ID=$(find_rule "$GROUP_ID" "$PORT" "$CIDR")
    print_import "${RESOURCE_NAME}[\"${CIDR}\"]" "$RULE_ID"
  done
}

echo "Processing ssas_alb ingress rules..." >&2
echo "# --- ssas_alb ingress rules --- #"
echo ""

print_import \
  "aws_vpc_security_group_ingress_rule.ssas_alb_ingress_admin" \
  "$(find_rule_by_sg $SSAS_ALB_SG_ID 444 $APP_SG_ID)"

print_import \
  "aws_vpc_security_group_ingress_rule.ssas_alb_ingress_public" \
  "$(find_rule_by_sg $SSAS_ALB_SG_ID 443 $APP_SG_ID)"

print_import \
  "aws_vpc_security_group_ingress_rule.ssas_alb_ingress_vpc" \
  "$(find_rule $SSAS_ALB_SG_ID 444 $VPC_CIDR)"

print_import \
  "aws_vpc_security_group_ingress_rule.ssas_alb_ingress_vpc_public" \
  "$(find_rule $SSAS_ALB_SG_ID 443 $VPC_CIDR)"

print_cidr_imports \
  "aws_vpc_security_group_ingress_rule.ssas_alb_ingress_gha_runners" \
  "$SSAS_ALB_SG_ID" 444 "$GHA_CIDRS"

print_cidr_imports \
  "aws_vpc_security_group_ingress_rule.ssas_alb_ingress_gha_runners_public" \
  "$SSAS_ALB_SG_ID" 443 "$GHA_CIDRS"

print_cidr_imports \
  "aws_vpc_security_group_ingress_rule.ssas_alb_ingress_aco_ms_admin" \
  "$SSAS_ALB_SG_ID" 444 "$ACO_MS_CIDRS"

print_cidr_imports \
  "aws_vpc_security_group_ingress_rule.ssas_alb_ingress_4i_public" \
  "$SSAS_ALB_SG_ID" 443 "$CIDRS_4I_PUBLIC"

print_cidr_imports \
  "aws_vpc_security_group_ingress_rule.ssas_alb_ingress_4i_admin" \
  "$SSAS_ALB_SG_ID" 444 "$CIDRS_4I_ADMIN"

print_cidr_imports \
  "aws_vpc_security_group_ingress_rule.ssas_alb_ingress_ihp_public" \
  "$SSAS_ALB_SG_ID" 443 "$IHP_CIDRS"

print_cidr_imports \
  "aws_vpc_security_group_ingress_rule.ssas_alb_ingress_ihp_admin" \
  "$SSAS_ALB_SG_ID" 444 "$IHP_CIDRS"

echo "Processing ssas_alb egress rules..." >&2
echo "# --- ssas_alb egress rules --- #"
echo ""

print_import \
  "aws_vpc_security_group_egress_rule.ssas_alb_egress_app" \
  "$(find_rule_by_sg $SSAS_ALB_SG_ID -1 $APP_SG_ID true)"

echo "Processing app_sg ingress rules..." >&2
echo "# --- app_sg ingress rules from ssas_alb --- #"
echo ""

print_import \
  "aws_vpc_security_group_ingress_rule.ssas_sg_ingress_admin" \
  "$(find_rule_by_sg $APP_SG_ID 3004 $SSAS_ALB_SG_ID)"

print_import \
  "aws_vpc_security_group_ingress_rule.ssas_sg_ingress_public" \
  "$(find_rule_by_sg $APP_SG_ID 3003 $SSAS_ALB_SG_ID)"

echo "Done. Review before applying." >&2
