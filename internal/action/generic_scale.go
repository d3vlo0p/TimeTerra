package action

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	v1alpha1 "github.com/d3vlo0p/TimeTerra/api/v1alpha1"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ClusterManager interface {
	DescribeCluster(ctx context.Context, identifier string) (writer string, readers []string, err error)
	ModifyInstance(ctx context.Context, identifier string, instanceClass string) error
	CheckInstancesAvailable(ctx context.Context, identifiers []string, instanceClass string) (allAvailable bool, err error)
	FailoverCluster(ctx context.Context, identifier string) error
}

type GenericScaleParams struct {
	InstanceClass string `json:"instanceClass"`
}

type GenericScaleHandler struct {
	Manager ClusterManager
}

func (h *GenericScaleHandler) Execute(ctx context.Context, logger logr.Logger, op *v1alpha1.ActionExecution, identifier string) (ctrl.Result, bool, error) {
	// Parse Parameters
	var params GenericScaleParams
	if err := json.Unmarshal([]byte(op.Spec.Parameters), &params); err != nil {
		logger.Error(err, "Failed to parse parameters")
		return ctrl.Result{}, true, err // Finish with error
	}
	if params.InstanceClass == "" {
		err := fmt.Errorf("instanceClass is empty")
		logger.Error(err, "Invalid parameters")
		return ctrl.Result{}, true, err
	}

	// State Machine
	if op.Status.StateData == nil {
		op.Status.StateData = make(map[string]string)
	}

	if op.Status.Phase == "" {
		op.Status.Phase = "Initialization"
	}

	logger.Info("Executing Scale Phase", "phase", op.Status.Phase)

	switch op.Status.Phase {
	case "Initialization":
		writer, readers, err := h.Manager.DescribeCluster(ctx, identifier)
		if err != nil {
			logger.Error(err, "DescribeCluster failed")
			return ctrl.Result{}, false, err // retry later
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
		_ = json.Unmarshal([]byte(op.Status.StateData["readers"]), &readers)

		for _, reader := range readers {
			err := h.Manager.ModifyInstance(ctx, reader, params.InstanceClass)
			if err != nil {
				logger.Error(err, "Failed to modify reader", "reader", reader)
				return ctrl.Result{}, false, err
			}
		}
		op.Status.Phase = "WaitReplicas"
		return ctrl.Result{RequeueAfter: 30 * time.Second}, false, nil

	case "WaitReplicas":
		var readers []string
		_ = json.Unmarshal([]byte(op.Status.StateData["readers"]), &readers)

		allAvailable, err := h.Manager.CheckInstancesAvailable(ctx, readers, params.InstanceClass)
		if err != nil {
			return ctrl.Result{}, false, err
		}

		if allAvailable {
			logger.Info("All replicas scaled and available. Proceeding to failover.")
			op.Status.Phase = "PerformingFailover"
			return ctrl.Result{Requeue: true}, false, nil
		}

		logger.Info("Waiting for replicas to scale and become available...")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, false, nil

	case "PerformingFailover":
		err := h.Manager.FailoverCluster(ctx, identifier)
		if err != nil {
			logger.Error(err, "Failover failed")
			return ctrl.Result{}, false, err
		}

		op.Status.Phase = "WaitWriterFailover"
		return ctrl.Result{RequeueAfter: 30 * time.Second}, false, nil

	case "WaitWriterFailover":
		writer := op.Status.StateData["writer"]

		currentWriter, _, err := h.Manager.DescribeCluster(ctx, identifier)
		if err != nil {
			return ctrl.Result{}, false, err
		}

		if currentWriter != writer && currentWriter != "" {
			logger.Info("Failover successful.")
			op.Status.Phase = "ScalingWriter"
			return ctrl.Result{Requeue: true}, false, nil
		}

		logger.Info("Waiting for failover to complete...")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, false, nil

	case "ScalingWriter":
		writer := op.Status.StateData["writer"]
		err := h.Manager.ModifyInstance(ctx, writer, params.InstanceClass)
		if err != nil {
			logger.Error(err, "Failed to modify original writer instance", "writer", writer)
			return ctrl.Result{}, false, err
		}

		op.Status.Phase = "WaitOldWriter"
		return ctrl.Result{RequeueAfter: 30 * time.Second}, false, nil

	case "WaitOldWriter":
		writer := op.Status.StateData["writer"]
		allAvailable, err := h.Manager.CheckInstancesAvailable(ctx, []string{writer}, params.InstanceClass)
		if err != nil {
			return ctrl.Result{}, false, err
		}

		if allAvailable {
			logger.Info("Old writer successfully scaled.")
			op.Status.Phase = "Completed"
			return ctrl.Result{}, true, nil // Done
		}

		logger.Info("Waiting for old writer to scale...")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, false, nil

	case "Completed":
		return ctrl.Result{}, true, nil
	}

	return ctrl.Result{}, true, fmt.Errorf("unknown phase %s", op.Status.Phase)
}
