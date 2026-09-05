# TimeTerra Examples & Testing Guide

This directory contains ready-to-test Kubernetes manifests demonstrating the core features of the TimeTerra operator.

Each example file is self-contained and includes both the target Kubernetes resources (Deployments, StatefulSets, HPAs, or Jobs) and the TimeTerra custom resources (`Schedule`, `K8sPodReplicas`, `K8sHpa`, `K8sRunJob`).

---

## Prerequisites

1. A running Kubernetes cluster (e.g. Minikube, Kind, k3d, or remote cluster).
2. TimeTerra CRDs installed:
   ```bash
   make install
   ```
3. TimeTerra operator running (either in-cluster via Helm/manifests or locally via `make run`):
   ```bash
   make run
   ```

---

## Available Examples

### 1. Scaling Deployments (`k8s_deployment_replicas.yaml`)
- **CRDs**: [Schedule](file:///mnt/e/Progetti/vscode/TimeTerra/api/v1alpha1/schedule_types.go), [K8sPodReplicas](file:///mnt/e/Progetti/vscode/TimeTerra/api/v1alpha1/k8spodreplicas_types.go)
- **Target**: Kubernetes `Deployment` (`demo-deployment`)
- **Behavior**: Periodically triggers `scaleup` (3 replicas) and `scalein` (1 replica) based on cron expressions.
- **Run**:
  ```bash
  kubectl apply -f examples/k8s_deployment_replicas.yaml
  ```
- **Observe**:
  ```bash
  kubectl get deployment demo-deployment -w
  kubectl get k8spodreplicas demo-deployment-replicas -o yaml
  ```

### 2. Scaling StatefulSets (`k8s_statefulset_replicas.yaml`)
- **CRDs**: [Schedule](file:///mnt/e/Progetti/vscode/TimeTerra/api/v1alpha1/schedule_types.go), [K8sPodReplicas](file:///mnt/e/Progetti/vscode/TimeTerra/api/v1alpha1/k8spodreplicas_types.go)
- **Target**: Kubernetes `StatefulSet` (`demo-statefulset`) with headless service
- **Behavior**: Alternates replica counts between `scaleup` (2 replicas) and `scalein` (1 replica).
- **Run**:
  ```bash
  kubectl apply -f examples/k8s_statefulset_replicas.yaml
  ```
- **Observe**:
  ```bash
  kubectl get statefulset demo-statefulset -w
  kubectl get k8spodreplicas demo-statefulset-replicas -o yaml
  ```

### 3. Scaling HPA Min/Max Replicas (`k8s_hpa_scaling.yaml`)
- **CRDs**: [Schedule](file:///mnt/e/Progetti/vscode/TimeTerra/api/v1alpha1/schedule_types.go), [K8sHpa](file:///mnt/e/Progetti/vscode/TimeTerra/api/v1alpha1/k8shpa_types.go)
- **Target**: Kubernetes `HorizontalPodAutoscaler` (`demo-hpa`)
- **Behavior**: Adjusts `minReplicas` and `maxReplicas` between peak (`min: 3, max: 10`) and off-peak (`min: 1, max: 4`).
- **Run**:
  ```bash
  kubectl apply -f examples/k8s_hpa_scaling.yaml
  ```
- **Observe**:
  ```bash
  kubectl get hpa demo-hpa -w
  kubectl get k8shpa demo-hpa-scaler -o yaml
  ```

### 4. Running Scheduled Batch Jobs (`k8s_run_job.yaml`)
- **CRDs**: [Schedule](file:///mnt/e/Progetti/vscode/TimeTerra/api/v1alpha1/schedule_types.go), [K8sRunJob](file:///mnt/e/Progetti/vscode/TimeTerra/api/v1alpha1/k8srunjob_types.go)
- **Target**: Kubernetes `batch/v1` `Job` resources created dynamically
- **Behavior**: TimeTerra spawns timestamped Jobs (e.g. `demo-k8srunjob-daily-backup-<timestamp>`) with automatic cleanup (`ttlSecondsAfterFinished: 120`).
- **Run**:
  ```bash
  kubectl apply -f examples/k8s_run_job.yaml
  ```
- **Observe**:
  ```bash
  kubectl get jobs -w
  kubectl get k8srunjob demo-k8srunjob -o yaml
  ```

---

## Cleaning Up

To remove any of the test resources:

```bash
kubectl delete -f examples/k8s_deployment_replicas.yaml
kubectl delete -f examples/k8s_statefulset_replicas.yaml
kubectl delete -f examples/k8s_hpa_scaling.yaml
kubectl delete -f examples/k8s_run_job.yaml
```
