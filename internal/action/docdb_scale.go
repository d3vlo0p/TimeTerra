package action

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/docdb"
	docdbtypes "github.com/aws/aws-sdk-go-v2/service/docdb/types"
	v1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DocumentDBScaleParams struct {
	InstanceClass string `json:"instanceClass"`
}

type DocumentDbScaleHandler struct {
	Client            client.Client
	OperatorNamespace string
	SecretProvider    func(secret *corev1.Secret, keysRef *v1alpha1.AwsCredentialsKeysRef) (*credentials.StaticCredentialsProvider, error)
}

type docdbClusterManager struct {
	client *docdb.Client
	opts   func(*docdb.Options)
}

func (m *docdbClusterManager) DescribeCluster(ctx context.Context, identifier string) (string, []string, error) {
	descResp, err := m.client.DescribeDBClusters(ctx, &docdb.DescribeDBClustersInput{
		DBClusterIdentifier: &identifier,
	}, m.opts)
	if err != nil {
		return "", nil, err
	}
	if len(descResp.DBClusters) == 0 {
		return "", nil, fmt.Errorf("cluster not found in AWS")
	}

	var cluster *docdbtypes.DBCluster
	for i := range descResp.DBClusters {
		if descResp.DBClusters[i].DBClusterIdentifier != nil && *descResp.DBClusters[i].DBClusterIdentifier == identifier {
			cluster = &descResp.DBClusters[i]
			break
		}
	}
	if cluster == nil {
		cluster = &descResp.DBClusters[0]
	}

	var writer string
	var readers []string

	for _, member := range cluster.DBClusterMembers {
		if member.DBInstanceIdentifier == nil {
			continue
		}
		if member.IsClusterWriter != nil && *member.IsClusterWriter {
			writer = *member.DBInstanceIdentifier
		} else {
			readers = append(readers, *member.DBInstanceIdentifier)
		}
	}

	// Fallback 1: If single instance cluster, treat that instance as writer
	if writer == "" && len(cluster.DBClusterMembers) == 1 && cluster.DBClusterMembers[0].DBInstanceIdentifier != nil {
		writer = *cluster.DBClusterMembers[0].DBInstanceIdentifier
		readers = nil
	}

	// Fallback 2: If no DBClusterMembers, look up instances by cluster identifier filter
	if writer == "" && len(cluster.DBClusterMembers) == 0 {
		instResp, err := m.client.DescribeDBInstances(ctx, &docdb.DescribeDBInstancesInput{
			Filters: []docdbtypes.Filter{
				{
					Name:   aws.String("db-cluster-id"),
					Values: []string{identifier},
				},
			},
		}, m.opts)
		if err == nil && len(instResp.DBInstances) > 0 {
			for _, inst := range instResp.DBInstances {
				if inst.DBInstanceIdentifier == nil {
					continue
				}
				if inst.DBClusterIdentifier == nil || *inst.DBClusterIdentifier != identifier {
					continue
				}
				if writer == "" {
					writer = *inst.DBInstanceIdentifier
				} else {
					readers = append(readers, *inst.DBInstanceIdentifier)
				}
			}
		}
	}

	if writer == "" {
		return "", nil, fmt.Errorf("no writer instance found for cluster %s", identifier)
	}

	return writer, readers, nil
}

func (m *docdbClusterManager) ModifyInstance(ctx context.Context, identifier string, instanceClass string) error {
	_, err := m.client.ModifyDBInstance(ctx, &docdb.ModifyDBInstanceInput{
		DBInstanceIdentifier: aws.String(identifier),
		DBInstanceClass:      aws.String(instanceClass),
		ApplyImmediately:     aws.Bool(true),
	}, m.opts)
	return err
}

func (m *docdbClusterManager) CheckInstancesAvailable(ctx context.Context, identifiers []string, instanceClass string) (bool, error) {
	for _, id := range identifiers {
		desc, err := m.client.DescribeDBInstances(ctx, &docdb.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(id),
		}, m.opts)
		if err != nil {
			return false, err
		}
		if len(desc.DBInstances) == 0 {
			return false, nil
		}
		inst := desc.DBInstances[0]
		if inst.DBInstanceStatus == nil || *inst.DBInstanceStatus != "available" {
			return false, nil
		}
		if inst.DBInstanceClass == nil || *inst.DBInstanceClass != instanceClass {
			return false, nil
		}
	}
	return true, nil
}

func (m *docdbClusterManager) FailoverCluster(ctx context.Context, identifier string) error {
	_, err := m.client.FailoverDBCluster(ctx, &docdb.FailoverDBClusterInput{
		DBClusterIdentifier: &identifier,
	}, m.opts)
	return err
}

func (h *DocumentDbScaleHandler) Execute(ctx context.Context, logger logr.Logger, op *v1alpha1.ActionExecution) (ctrl.Result, bool, error) {
	// 1. Fetch Target Resource
	target := &v1alpha1.AwsDocumentDBCluster{}
	err := h.Client.Get(ctx, client.ObjectKey{Name: op.Spec.TargetResource.Name, Namespace: op.Spec.TargetResource.Namespace}, target)
	if err != nil {
		logger.Error(err, "Failed to fetch TargetResource")
		return ctrl.Result{}, false, err
	}

	// 2. AWS Config
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

	opts := func(o *docdb.Options) {
		o.Region = clusterInfo.Region
		if target.Spec.ServiceEndpoint != nil {
			o.BaseEndpoint = target.Spec.ServiceEndpoint
		}
	}
	docdbClient := docdb.NewFromConfig(cfg)

	mgr := &docdbClusterManager{
		client: docdbClient,
		opts:   opts,
	}

	genericHandler := &GenericScaleHandler{
		Manager: mgr,
	}

	return genericHandler.Execute(ctx, logger, op, clusterInfo.Identifier)
}
