# TimeTerra

TimeTerra is a cron scheduling kubernetes operator that allows you to define a schedule for executing actions at specific times and associate those actions with different resources

## Description

With TimeTerra, you can create a schedule using cron expressions on which you can implement the desired actions on different types of kubernetes and cloud resources.

### Supported Actions

- **Set the number of replicas** for Deployments and StatefulSets.
- **Set the minimum and maximum number of replicas** for Horizontal Pod Autoscaling.
- **Start and stop AWS RDS clusters**.
- **Start and stop AWS DocumentDB clusters**.
- **Start and stop AWS Ec2 instances**.
- **Start and stop AWS Transfer Family servers**.

### Use Cases

- Reduce the cloud cost during periods of non-use
- Prepare the environment to support expected peaks

## Getting Started

To use TimeTerra, follow these steps:

1. Install TimeTerra on your cluster.
2. Define a schedule with the desired actions.
3. Associate each action with a valid cron expression.
4. Define through the available resources (CRs) the actions defined in the schedule.

### To Deploy on the cluster
**Install the CRDs:**

```sh
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/core.timeterra.d3vlo0p.dev_awsdocumentdbclusters.yaml
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/core.timeterra.d3vlo0p.dev_awsec2instances.yaml
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/core.timeterra.d3vlo0p.dev_awsrdsauroraclusters.yaml
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/core.timeterra.d3vlo0p.dev_awstransferfamilies.yaml
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/core.timeterra.d3vlo0p.dev_k8shpas.yaml
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/core.timeterra.d3vlo0p.dev_k8spodreplicas.yaml
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/core.timeterra.d3vlo0p.dev_schedules.yaml
```

**Install the operator with Helm:**

```sh
helm upgrade -n timeterra --create-namespace --install operator oci://ghcr.io/d3vlo0p/timeterra
```

### To Uninstall
**Unistall the helm chart**

```sh
helm uninstall -n timeterra operator
kubectl delete namespace timeterra
```

**Delete the CRDs from the cluster:**

```sh
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/core.timeterra.d3vlo0p.dev_awsdocumentdbclusters.yaml
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/core.timeterra.d3vlo0p.dev_awsec2instances.yaml
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/core.timeterra.d3vlo0p.dev_awsrdsauroraclusters.yaml
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/core.timeterra.d3vlo0p.dev_awstransferfamilies.yaml
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/core.timeterra.d3vlo0p.dev_k8shpas.yaml
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/core.timeterra.d3vlo0p.dev_k8spodreplicas.yaml
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/core.timeterra.d3vlo0p.dev_schedules.yaml
```

### Define a Schedule

```yaml
apiVersion: core.timeterra.d3vlo0p.dev/v1alpha1
kind: Schedule
metadata:
  name: schedule-sample
spec:
  actions:
    scaleup:
      cron: '0/4 * * * *'
    scalein:
      cron: '2/4 * * * *'
---
apiVersion: core.timeterra.d3vlo0p.dev/v1alpha1
kind: K8sPodReplicas
metadata:
  name: k8spodreplicas-sample
spec:
  enabled: True
  resourceType: Deployment
  labelSelector:
    matchLabels:
      app.kubernetes.io/name: hello-world
  namespaces:
    - default
  schedule: schedule-sample
  actions:
    scalein:
      replicas: 1
    scaleup:
      replicas: 2
```
more samples [here](./config/samples/)

## License

Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

