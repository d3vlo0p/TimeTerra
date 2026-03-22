# TimeTerra: Technical Overview & Guide

TimeTerra is a Kubernetes operator designed to manage time-based (cron) actions on cloud (AWS) and Kubernetes resources. This document serves as a technical compass for AI assistants and developers.

> [!IMPORTANT]
> **Project Foundation**: This project is built using **Operator-SDK**. The directory structure (API, controllers, etc.) follows standard conventions and **must be strictly respected** for compatibility with SDK tools (e.g., `make generate`, `make manifests`). Use **Devbox** to ensure a consistent development environment.

## Development Environment (`devbox`)

This project uses [Devbox](https://www.jetpack.io/devbox/docs/) to manage the development environment and dependencies (Go, Operator-SDK, Make, Task).

-   **Install dependencies**: `devbox install`
-   **Start a shell**: `devbox shell` (all commands like `make`, `operator-sdk`, and `task` will be available)
-   **Run a command directly**: `devbox run <command>` (e.g., `devbox run make build`)

## Core Architecture

The operator's logic is distributed across three main layers:

1.  **Scheduling Layer (`internal/cron`)**:
    -   `ScheduleService`: The "heartbeat" of the operator. It manages a `robfig/cron/v3` instance.
    -   It maintains an internal map linking `Schedule` CRs to specific actions on target resources.
2.  **Reconciliation Layer (`internal/controller`)**:
    -   **Generic Pattern**: Most target controllers use the `reconcileResource` function in `resource_controller.go`.
    -   **Loop Logic**: Watches for changes in target resources and ensures their cron jobs are correctly registered, updated, or removed in the `ScheduleService`.
3.  **Action Layer (Target Controllers)**:
    -   Each target resource controller (e.g., `AwsEc2InstanceReconciler`) implements a `job` callback which is triggered by the cron.
    -   Actions include starting/stopping EC2 instances, scaling HPA, or running K8s Jobs.

## Key Interfaces & Types

Found in `internal/controller/types.go` and `resource_controller.go`:

-   **`Activable`**: Simple `IsActive() bool` check for resources and actions.
-   **`ReconcileResource`**: Interface that target resources must implement to work with the generic `reconcileResource` logic.
-   **`Reconciler`**: Provides access to core services like `ScheduleService`, `NotificationService`, and the `EventRecorder`.

## Project Structure

-   `api/v1alpha1/`: Go type definitions. CRD schemas are defined here.
-   `internal/controller/`: Reconciliation logic.
    -   `resource_controller.go`: Contains the generic reconciliation logic used by target resources.
-   `internal/cron/`: Core scheduling logic.
-   `notification/`: Multi-provider notification system (Slack, MS Teams).
-   `config/`: K8s manifests (CRDs, RBAC, etc.).
-   `test/`: Integration and functional tests.

## Recipes: Adding a New Target Resource

1.  **Generate API and Controller**: `operator-sdk create api --group timeterra --version v1alpha1 --kind <KindName> --resource --controller`
2.  **Define the API**: Update the generated types in `api/v1alpha1/` (ensure they implement `ReconcileResource`).
3.  **Implement Logic**:
    -   Follow the pattern in `internal/controller/awsec2instance_controller.go`.
    -   Use `reconcileResource` in the `Reconcile` loop.
    -   Implement the `job` callback with the actual logic (e.g., AWS SDK calls).
4.  **Register Controller**: Add the new reconciler to `cmd/main.go`.
5.  **Update Manifests**: Run `make manifests generate`.

## Development & Maintenance

-   **Build**: `make build`
-   **Test**: `make test` (runs all unit/envtest tests)
-   **Lint**: `golangci-lint run`
-   **Metrics**: Prometheus metrics are exposed on `:8443/metrics` (secured with RBAC).

## Future Maintenance Notes

-   **Conditions**: Always use `addToConditions` to update resource status.
-   **Notifications**: Trigger notifications via `NotificationService.Send()` within the job callback.
-   **AWS Configuration**: Use `config.LoadDefaultConfig` and handle secrets via the provided utility functions in the controllers.