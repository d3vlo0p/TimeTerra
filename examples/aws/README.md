# Testing TimeTerra with Floci Local Cloud Emulator

[Floci](https://floci.io) is a fast, MIT-licensed local cloud emulator supporting AWS, GCP, Azure, and OCI.

TimeTerra natively supports targeting custom cloud endpoints via the `spec.serviceEndpoint` field on its Custom Resources (CRs). This directory provides sample manifests configured to interact with a local Floci AWS instance.

---

## 1. Quickstart

### Start the Floci AWS Emulator
Start the local Floci container on port `4566`:
```bash
task floci:start
# or
task floci:aws:start
```

Verify Floci is healthy:
```bash
task floci:status
```

### Seed Mock Resources
To test TimeTerra, mock resources (EC2 instance, Transfer Family server, RDS Aurora cluster, and DocumentDB cluster) need to exist in Floci. Run the seed task:
```bash
task floci:seed
# or
task floci:aws:seed
```

This task:
1. Uses `aws-cli` (via devbox) against `http://localhost:4566`.
2. Creates:
   - An EC2 instance (tagged `timeterra-demo-ec2`).
   - A Transfer Family SFTP server.
   - An RDS Aurora DB Cluster (`timeterra-demo-aurora`) & DB instance.
   - A DocumentDB Cluster (`timeterra-demo-docdb`) & DB instance.
3. Automatically updates the example CRs with the generated IDs.
4. Deploys the mock `aws-credentials` Secret (`examples/aws/credentials_secret.yaml`) to your local Kubernetes cluster.

---

## 2. Example Manifests

The following sample Custom Resources are provided:

| Manifest | Kind | Description |
| :--- | :--- | :--- |
| [`credentials_secret.yaml`](./credentials_secret.yaml) | `Secret` | Mock AWS credentials (`aws_access_key_id: test`, `aws_secret_access_key: test`). |
| [`aws_ec2_instance.yaml`](./aws_ec2_instance.yaml) | `AwsEc2Instance` | Manages start/stop cycles on the seeded EC2 instance. |
| [`aws_transfer_family.yaml`](./aws_transfer_family.yaml) | `AwsTransferFamily` | Manages start/stop cycles on the seeded SFTP server. |
| [`aws_rds_aurora.yaml`](./aws_rds_aurora.yaml) | `AwsRdsAuroraCluster` | Scales the seeded Aurora PostgreSQL cluster between instance classes. |
| [`aws_documentdb.yaml`](./aws_documentdb.yaml) | `AwsDocumentDBCluster` | Scales the seeded DocumentDB cluster between instance classes. |

---

## 3. Deploying Examples with Kustomize

When TimeTerra is deployed inside a Kubernetes cluster, it needs to reach the Floci emulator on the host machine. You can deploy all examples together using `task`, which automatically discovers the host's IP and uses Kustomize to patch `spec.serviceEndpoint`:

```bash
# Deploy all AWS examples with host IP injected
task floci:deploy-examples
# or
task floci:aws:deploy-examples

# Undeploy examples
task floci:undeploy-examples
```

You can also override the host IP manually if desired:
```bash
HOST_IP=192.168.1.50 task floci:deploy-examples
```

Alternatively, apply individual manifests or use standard Kustomize directly:
```bash
kubectl apply -k examples/aws
```

Observe reconciliation and execution events:
```bash
kubectl describe awsec2instance demo-ec2-instance -n timeterra
```

---

## 4. Teardown & Cleanup

Stop the Floci emulator when finished:
```bash
task floci:stop
```
