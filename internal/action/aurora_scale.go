package action

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	v1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AuroraScaleParams struct {
	InstanceClass string `json:"instanceClass"`
}

type AuroraScaleHandler struct {
	Client            client.Client
	OperatorNamespace string
	SecretProvider    func(secret *corev1.Secret, keysRef *v1alpha1.AwsCredentialsKeysRef) (*credentials.StaticCredentialsProvider, error)
}

func (h *AuroraScaleHandler) Execute(ctx context.Context, logger logr.Logger, op *v1alpha1.ActionExecution) (ctrl.Result, bool, error) {
	// 1. Fetch Target Resource
	target := &v1alpha1.AwsRdsAuroraCluster{}
	err := h.Client.Get(ctx, client.ObjectKey{Name: op.Spec.TargetResource.Name, Namespace: op.Spec.TargetResource.Namespace}, target)
	if err != nil {
		logger.Error(err, "Failed to fetch TargetResource")
		return ctrl.Result{}, false, err
	}

	// 2. Parse Parameters
	var params AuroraScaleParams
	if err := json.Unmarshal([]byte(op.Spec.Parameters), &params); err != nil {
		logger.Error(err, "Failed to parse parameters")
		return ctrl.Result{}, true, err // Finish with error
	}
	if params.InstanceClass == "" {
		err := fmt.Errorf("instanceClass is empty")
		logger.Error(err, "Invalid parameters")
		return ctrl.Result{}, true, err
	}

	// 3. AWS Config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error(err, "unable to load SDK config")
		return ctrl.Result{}, true, err
	}
	if target.Spec.Credentials != nil {
		secret := &corev1.Secret{}
		ns := target.Spec.Credentials.Namespace
		if ns == "" {
			ns = h.OperatorNamespace
		}
		err = h.Client.Get(ctx, types.NamespacedName{Name: target.Spec.Credentials.SecretName, Namespace: ns}, secret)
		if err != nil {
			logger.Error(err, "Failed to get Secret resource")
			return ctrl.Result{}, true, err
		}

		if h.SecretProvider != nil {
			cfg.Credentials, err = h.SecretProvider(secret, target.Spec.Credentials.KeysRef)
			if err != nil {
				logger.Error(err, "Failed to configure credentials")
				return ctrl.Result{}, true, err
			}
		}
	}

	if len(target.Spec.DBClusterIdentifiers) == 0 {
		return ctrl.Result{}, true, fmt.Errorf("no DBClusterIdentifiers specified")
	}
	clusterInfo := target.Spec.DBClusterIdentifiers[0] // Taking the first one

	opts := func(o *rds.Options) {
		o.Region = clusterInfo.Region
		if target.Spec.ServiceEndpoint != nil {
			o.BaseEndpoint = target.Spec.ServiceEndpoint
		}
	}
	rdsClient := rds.NewFromConfig(cfg)

	// State Machine
	if op.Status.StateData == nil {
		op.Status.StateData = make(map[string]string)
	}

	if op.Status.Phase == "" {
		op.Status.Phase = "Initialization"
	}

	logger.Info("Executing Aurora Scale Phase", "phase", op.Status.Phase)

	switch op.Status.Phase {
	case "Initialization":
		descResp, err := rdsClient.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{
			DBClusterIdentifier: &clusterInfo.Identifier,
		}, opts)
		if err != nil {
			logger.Error(err, "DescribeDBClusters failed")
			return ctrl.Result{}, false, err // retry later
		}
		if len(descResp.DBClusters) == 0 {
			return ctrl.Result{}, true, fmt.Errorf("cluster not found in AWS")
		}

		cluster := descResp.DBClusters[0]
		var writer string
		var readers []string

		for _, member := range cluster.DBClusterMembers {
			if member.IsClusterWriter != nil && *member.IsClusterWriter {
				writer = *member.DBInstanceIdentifier
			} else {
				readers = append(readers, *member.DBInstanceIdentifier)
			}
		}

		op.Status.StateData["writer"] = writer
		readersJson, _ := json.Marshal(readers)
		op.Status.StateData["readers"] = string(readersJson)

		if len(readers) == 0 {
			logger.Info("No read replicas found. Proceeding directly to scale writer.")
			op.Status.Phase = "ScalingWriter"
		} else {
			op.Status.Phase = "ScalingReplicas"
		}
		return ctrl.Result{Requeue: true}, false, nil

	case "ScalingReplicas":
		var readers []string
		json.Unmarshal([]byte(op.Status.StateData["readers"]), &readers)

		for _, reader := range readers {
			_, err := rdsClient.ModifyDBInstance(ctx, &rds.ModifyDBInstanceInput{
				DBInstanceIdentifier: aws.String(reader),
				DBInstanceClass:      aws.String(params.InstanceClass),
				ApplyImmediately:     aws.Bool(true),
			}, opts)
			if err != nil {
				logger.Error(err, "Failed to modify reader", "reader", reader)
				return ctrl.Result{}, false, err
			}
		}
		op.Status.Phase = "WaitReplicas"
		return ctrl.Result{RequeueAfter: 30 * time.Second}, false, nil

	case "WaitReplicas":
		var readers []string
		json.Unmarshal([]byte(op.Status.StateData["readers"]), &readers)

		allAvailable := true
		for _, reader := range readers {
			desc, err := rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: aws.String(reader),
			}, opts)
			if err != nil {
				return ctrl.Result{}, false, err
			}
			if len(desc.DBInstances) == 0 || *desc.DBInstances[0].DBInstanceStatus != "available" {
				allAvailable = false
				break
			}
			if *desc.DBInstances[0].DBInstanceClass != params.InstanceClass {
				allAvailable = false
				break
			}
		}

		if allAvailable {
			logger.Info("All replicas scaled and available. Proceeding to failover.")
			op.Status.Phase = "PerformingFailover"
			return ctrl.Result{Requeue: true}, false, nil
		}

		logger.Info("Waiting for replicas to scale and become available...")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, false, nil

	case "PerformingFailover":
		_, err := rdsClient.FailoverDBCluster(ctx, &rds.FailoverDBClusterInput{
			DBClusterIdentifier: &clusterInfo.Identifier,
		}, opts)
		if err != nil {
			logger.Error(err, "Failover failed")
			return ctrl.Result{}, false, err
		}

		op.Status.Phase = "WaitWriterFailover"
		return ctrl.Result{RequeueAfter: 30 * time.Second}, false, nil

	case "WaitWriterFailover":
		writer := op.Status.StateData["writer"]

		desc, err := rdsClient.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{
			DBClusterIdentifier: &clusterInfo.Identifier,
		}, opts)
		if err != nil {
			return ctrl.Result{}, false, err
		}

		if len(desc.DBClusters) > 0 {
			for _, member := range desc.DBClusters[0].DBClusterMembers {
				if *member.DBInstanceIdentifier == writer {
					if member.IsClusterWriter != nil && *member.IsClusterWriter == false {
						logger.Info("Failover successful.")
						op.Status.Phase = "ScalingWriter"
						return ctrl.Result{Requeue: true}, false, nil
					}
					break
				}
			}
		}

		logger.Info("Waiting for failover to complete...")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, false, nil

	case "ScalingWriter":
		writer := op.Status.StateData["writer"]
		_, err := rdsClient.ModifyDBInstance(ctx, &rds.ModifyDBInstanceInput{
			DBInstanceIdentifier: aws.String(writer),
			DBInstanceClass:      aws.String(params.InstanceClass),
			ApplyImmediately:     aws.Bool(true),
		}, opts)

		if err != nil {
			logger.Error(err, "Failed to modify original writer instance", "writer", writer)
			return ctrl.Result{}, false, err
		}

		op.Status.Phase = "WaitOldWriter"
		return ctrl.Result{RequeueAfter: 30 * time.Second}, false, nil

	case "WaitOldWriter":
		writer := op.Status.StateData["writer"]
		desc, err := rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(writer),
		}, opts)
		if err != nil {
			return ctrl.Result{}, false, err
		}
		if len(desc.DBInstances) > 0 && *desc.DBInstances[0].DBInstanceStatus == "available" {
			if *desc.DBInstances[0].DBInstanceClass == params.InstanceClass {
				logger.Info("Old writer successfully scaled.")
				op.Status.Phase = "Completed"
				return ctrl.Result{}, true, nil // Done
			}
		}

		logger.Info("Waiting for old writer to scale...")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, false, nil

	case "Completed":
		return ctrl.Result{}, true, nil
	}

	return ctrl.Result{}, true, fmt.Errorf("unknown phase %s", op.Status.Phase)
}
