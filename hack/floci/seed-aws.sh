#!/usr/bin/env bash
set -euo pipefail

# Configuration
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-test}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-test}"
export AWS_DEFAULT_REGION="${AWS_DEFAULT_REGION:-us-east-1}"
export AWS_ENDPOINT_URL="${AWS_ENDPOINT_URL:-http://localhost:4566}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "=== Floci AWS Mock Resource Provisioner ==="
echo "Endpoint: ${AWS_ENDPOINT_URL}"
echo "Region:   ${AWS_DEFAULT_REGION}"

# Helper to run AWS CLI directly or via devbox
run_aws() {
  if command -v aws >/dev/null 2>&1; then
    aws --endpoint-url "${AWS_ENDPOINT_URL}" "$@"
  elif command -v devbox >/dev/null 2>&1; then
    devbox run -- aws --endpoint-url "${AWS_ENDPOINT_URL}" "$@"
  else
    echo "Error: Neither 'aws' nor 'devbox' found in PATH." >&2
    exit 1
  fi
}

# Verify Floci is running
if ! curl -sf "${AWS_ENDPOINT_URL}" >/dev/null 2>&1; then
  echo "Floci is not responding on ${AWS_ENDPOINT_URL}."
  echo "Attempting to start Floci via 'floci start'..."
  if command -v floci >/dev/null 2>&1; then
    floci start
  else
    echo "Error: 'floci' CLI is not available to start the emulator." >&2
    exit 1
  fi
fi

# 1. EC2 Instance
echo "--- Provisioning EC2 Instance ---"
EC2_ID=$(run_aws ec2 describe-instances \
  --filters "Name=tag:Name,Values=timeterra-demo-ec2" "Name=instance-state-name,Values=pending,running,stopped,stopping" \
  --query "Reservations[0].Instances[0].InstanceId" --output text 2>/dev/null || echo "None")

if [ "${EC2_ID}" = "None" ] || [ -z "${EC2_ID}" ] || [ "${EC2_ID}" = "null" ]; then
  echo "Launching new EC2 instance in Floci..."
  EC2_ID=$(run_aws ec2 run-instances \
    --image-id ami-12345678 \
    --count 1 \
    --instance-type t3.micro \
    --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=timeterra-demo-ec2}]" \
    --query "Instances[0].InstanceId" --output text)
  echo "Created EC2 Instance: ${EC2_ID}"
else
  echo "Found existing EC2 Instance: ${EC2_ID}"
fi

# Update examples/aws/aws_ec2_instance.yaml
if [ -f "${REPO_ROOT}/examples/aws/aws_ec2_instance.yaml" ]; then
  sed -i "s/id: i-[a-zA-Z0-9]*/id: ${EC2_ID}/g" "${REPO_ROOT}/examples/aws/aws_ec2_instance.yaml"
  echo "Updated examples/aws/aws_ec2_instance.yaml with ID ${EC2_ID}"
fi

# 2. Transfer Family Server
echo "--- Provisioning Transfer Family Server ---"
SERVER_ID=$(run_aws transfer list-servers \
  --query "Servers[0].ServerId" --output text 2>/dev/null || echo "None")

if [ "${SERVER_ID}" = "None" ] || [ -z "${SERVER_ID}" ] || [ "${SERVER_ID}" = "null" ]; then
  echo "Creating Transfer Family SFTP Server in Floci..."
  SERVER_ID=$(run_aws transfer create-server \
    --protocols SFTP \
    --endpoint-type PUBLIC \
    --tags Key=Name,Value=timeterra-demo-transfer \
    --query "ServerId" --output text)
  echo "Created Transfer Family Server: ${SERVER_ID}"
else
  echo "Found existing Transfer Family Server: ${SERVER_ID}"
fi

# Update examples/aws/aws_transfer_family.yaml
if [ -f "${REPO_ROOT}/examples/aws/aws_transfer_family.yaml" ]; then
  sed -i "s/id: s-[a-zA-Z0-9]*/id: ${SERVER_ID}/g" "${REPO_ROOT}/examples/aws/aws_transfer_family.yaml"
  echo "Updated examples/aws/aws_transfer_family.yaml with ID ${SERVER_ID}"
fi

# 3. RDS Aurora Cluster & DB Instance
echo "--- Provisioning RDS Aurora Cluster ---"
AURORA_CLUSTER_ID="timeterra-demo-aurora"
if ! run_aws rds describe-db-clusters --db-cluster-identifier "${AURORA_CLUSTER_ID}" >/dev/null 2>&1; then
  echo "Creating RDS Aurora Cluster ${AURORA_CLUSTER_ID}..."
  run_aws rds create-db-cluster \
    --db-cluster-identifier "${AURORA_CLUSTER_ID}" \
    --engine aurora-postgresql \
    --master-username root \
    --master-user-password "Password123!" >/dev/null
  echo "Creating RDS DB Instance ${AURORA_CLUSTER_ID}-inst1..."
  run_aws rds create-db-instance \
    --db-instance-identifier "${AURORA_CLUSTER_ID}-inst1" \
    --db-cluster-identifier "${AURORA_CLUSTER_ID}" \
    --db-instance-class db.t3.medium \
    --engine aurora-postgresql >/dev/null
  echo "Created RDS Aurora Cluster & DB Instance."
else
  echo "Found existing RDS Aurora Cluster: ${AURORA_CLUSTER_ID}"
fi

# 4. DocumentDB Cluster & DB Instance
echo "--- Provisioning DocumentDB Cluster ---"
DOCDB_CLUSTER_ID="timeterra-demo-docdb"
if ! run_aws docdb describe-db-clusters --db-cluster-identifier "${DOCDB_CLUSTER_ID}" >/dev/null 2>&1; then
  echo "Creating DocumentDB Cluster ${DOCDB_CLUSTER_ID}..."
  run_aws docdb create-db-cluster \
    --db-cluster-identifier "${DOCDB_CLUSTER_ID}" \
    --engine docdb \
    --master-username root \
    --master-user-password "Password123!" >/dev/null
  echo "Creating DocumentDB DB Instance ${DOCDB_CLUSTER_ID}-inst1..."
  run_aws docdb create-db-instance \
    --db-instance-identifier "${DOCDB_CLUSTER_ID}-inst1" \
    --db-cluster-identifier "${DOCDB_CLUSTER_ID}" \
    --db-instance-class db.r5.large \
    --engine docdb >/dev/null
  echo "Created DocumentDB Cluster & DB Instance."
else
  echo "Found existing DocumentDB Cluster: ${DOCDB_CLUSTER_ID}"
fi

# 5. Apply mock credentials Secret if Kubernetes is accessible
echo "--- Checking Kubernetes Secret ---"
if command -v kubectl >/dev/null 2>&1; then
  if kubectl cluster-info >/dev/null 2>&1; then
    kubectl apply -f "${REPO_ROOT}/examples/aws/credentials_secret.yaml"
    echo "Applied credentials_secret.yaml to Kubernetes."
  else
    echo "Note: No active Kubernetes cluster reachable. Skipping secret application."
  fi
fi

echo ""
echo "=== Seeding Complete ==="
echo "Mock AWS Resources in Floci:"
echo "  - EC2 Instance:     ${EC2_ID}"
echo "  - Transfer Server:  ${SERVER_ID}"
echo "  - Aurora Cluster:   ${AURORA_CLUSTER_ID}"
echo "  - DocDB Cluster:    ${DOCDB_CLUSTER_ID}"
echo ""
echo "Sample manifests in 'examples/aws/' are ready to use with TimeTerra!"
