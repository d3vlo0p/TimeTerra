# TimeTerra

TimeTerra is a cron scheduling kubernetes operator that allows you to define a schedule for executing actions at specific times and associate those actions with different resources

## Description

With TimeTerra, you can create a schedule using cron expressions on which you can implement the desired actions on different types of kubernetes and cloud resources.

### Supported Actions

- **Set the number of replicas** for Deployments and StatefulSets.
- **Set the minimum and maximum number of replicas** for Horizontal Pod Autoscaling.
- **Run Jobs** with your own logic and permissions.
- **Start and stop RDS clusters** on AWS.
- **Start and stop DocumentDB clusters** on AWS.
- **Start and stop Ec2 instances** on AWS.
- **Start and stop Transfer Family servers** on AWS.

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
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_awsdocumentdbclusters.yaml
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_awsec2instances.yaml
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_awsrdsauroraclusters.yaml
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_awstransferfamilies.yaml
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_k8shpas.yaml
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_k8spodreplicas.yaml
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_k8srunjobs.yaml
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_schedules.yaml
kubectl apply -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_notificationpolicies.yaml
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
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_awsdocumentdbclusters.yaml
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_awsec2instances.yaml
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_awsrdsauroraclusters.yaml
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_awstransferfamilies.yaml
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_k8shpas.yaml
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_k8spodreplicas.yaml
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_k8srunjobs.yaml
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_schedules.yaml
kubectl delete -f https://raw.githubusercontent.com/d3vlo0p/TimeTerra/main/config/crd/bases/timeterra.d3vlo0p.dev_notificationpolicies.yaml
```

### Define a Schedule

```yaml
apiVersion: timeterra.d3vlo0p.dev/v1alpha1
kind: Schedule
metadata:
  name: schedule-sample
spec:
  enabled: True
  activePeriods: []
  inactivePeriods: 
    - start: '2024-12-24T00:00:00Z'
      end: '2024-12-26T00:00:00Z'
  actions:
    scaleup:
      cron: '0/4 * * * *'
    scalein:
      cron: '2/4 * * * *'
---
apiVersion: timeterra.d3vlo0p.dev/v1alpha1
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

## AWS

To allow timeterra to be able to call the AWS API there are several methods, including:

On EKS:
- [EKS Pod Identity](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html)
- [IAM roles for service accounts](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)

Others:
- Setting env vars, you can find the list [here](https://docs.aws.amazon.com/sdkref/latest/guide/settings-reference.html#EVarSettings)
- Mounting as volume the [credentials and config files](https://docs.aws.amazon.com/sdkref/latest/guide/file-format.html) in the operator folder [~/.aws/config](https://docs.aws.amazon.com/sdkref/latest/guide/file-location.html)
- Specify the credentials property in the AWS CRs with [this format](api/v1alpha1/aws_types.go), this allows you to read the credentials from a kubernetes secret, you can do your mapping or relaing on the default keys. You can have different credentials for different resources.

  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: aws-credentials-example
    namespace: timeterra
  stringData:
    aws_access_key_id: ASIAIOSFODNN7EXAMPLE
    aws_secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
    aws_session_token: IQoJb3JpZ2luX2IQoJb3JpZ2luX2IQoJb3JpZ2luX2IQoJb3JpZ2luX2IQoJb3JpZVERYLONGSTRINGEXAMPLE
  type: Opaque
  ```

Example of AWS IAM Policy to attach to the IAM user / role.
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "rds",
      "Action": [
        "rds:StartDBCluster",
        "rds:StopDBCluster"
      ],
      "Effect": "Allow",
      "Resource": "*"
    },
    {
      "Sid": "docdb",
      "Action": [
        "docdb-elastic:StartCluster",
        "docdb-elastic:StopCluster"
      ],
      "Effect": "Allow",
      "Resource": "*"
    },
    {
      "Sid": "sftp",
      "Action": [
        "transfer:StartServer",
        "transfer:StopServer"
      ],
      "Effect": "Allow",
      "Resource": "*"
    },
    {
      "Sid": "ec2",
      "Action": [
        "ec2:StartInstances",
        "ec2:StopInstances"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
```

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

