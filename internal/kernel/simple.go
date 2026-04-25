package kernel

import (
	"context"
	"fmt"
	"time"

	"babel-runtime/internal/agent"
	"babel-runtime/internal/core/types"
	"babel-runtime/internal/delivery"
	"babel-runtime/internal/mode"
	"babel-runtime/internal/projection"
	"babel-runtime/internal/requirementregistry"
	"babel-runtime/internal/repository"
)

type SimpleEngine struct {
	Repo       repository.Repository
	Router     mode.Router
	Supervisor agent.Supervisor
	Projector  projection.Projector
	Dispatcher delivery.Dispatcher
	Requirements requirementregistry.Registry
	Owner      string
	LeaseTTL   time.Duration
	Now        func() time.Time
}

func (e SimpleEngine) Accept(ctx context.Context, env types.InboundEnvelope) (types.ExecutionTicket, error) {
	if e.Repo == nil || e.Router == nil {
		return types.ExecutionTicket{}, fmt.Errorf("kernel not initialized")
	}
	if env.RuntimeID == "" || env.IdempotencyKey == "" {
		return types.ExecutionTicket{}, fmt.Errorf("missing runtime or idempotency key")
	}

	now := e.now()
	ticket := types.ExecutionTicket{}
	err := e.Repo.RunExecutionTx(ctx, func(ctx context.Context, tx repository.ExecutionTx) error {
		runtime, err := tx.LoadRuntime(ctx, env.RuntimeID)
		if err != nil {
			return err
		}
		if _, err := resolveRuntimeRequirements(ctx, e.Requirements, runtime); err != nil {
			return err
		}

		existing, ok, err := tx.FindExecutionByIdempotency(ctx, env.RuntimeID, env.IdempotencyKey)
		if err != nil {
			return err
		}
		if ok {
			ticket = types.ExecutionTicket{
				ExecutionID: existing.ExecutionID,
				RuntimeID:   existing.RuntimeID,
				Stage:       existing.Stage,
				Reused:      true,
			}
			return nil
		}

		module, err := e.Router.Resolve(ctx, runtime, env)
		if err != nil {
			return err
		}
		command, err := module.BuildCommand(ctx, runtime, env)
		if err != nil {
			return err
		}

		// Accept only establishes the durable execution shell. Canonical state
		// mutation happens later in Resume so recovery can re-enter from a known stage.
		execution := types.ExecutionRecord{
			ExecutionID:        executionIDFor(env),
			RuntimeID:          env.RuntimeID,
			EnvelopeID:         env.EnvelopeID,
			ActorID:            env.UserID,
			Transport:          env.Transport,
			IdempotencyKey:     env.IdempotencyKey,
			ModeID:             module.ModeID(),
			Stage:              types.ExecutionPlanned,
			CommandType:        command.CommandType,
			CommandText:        command.Text,
			LeaseOwner:         e.owner(),
			LeaseExpiresAtUnix: now.Add(e.leaseTTL()).Unix(),
			AcceptedAtUnix:     now.Unix(),
			UpdatedAtUnix:      now.Unix(),
		}
		if err := tx.SaveExecution(ctx, execution); err != nil {
			return err
		}
		if err := tx.AppendEvent(ctx, types.RuntimeEvent{
			EventID:        execution.ExecutionID + ":accepted",
			RuntimeID:      execution.RuntimeID,
			ExecutionID:    execution.ExecutionID,
			Kind:           "execution.accepted",
			Text:           env.Text,
			OccurredAtUnix: now.Unix(),
		}); err != nil {
			return err
		}
		if err := tx.AppendEvent(ctx, types.RuntimeEvent{
			EventID:        execution.ExecutionID + ":planned",
			RuntimeID:      execution.RuntimeID,
			ExecutionID:    execution.ExecutionID,
			Kind:           "execution.planned",
			Text:           command.CommandType,
			OccurredAtUnix: now.Unix(),
		}); err != nil {
			return err
		}

		ticket = types.ExecutionTicket{
			ExecutionID: execution.ExecutionID,
			RuntimeID:   execution.RuntimeID,
			Stage:       execution.Stage,
		}
		return nil
	})
	return ticket, err
}

func (e SimpleEngine) Resume(ctx context.Context, executionID string) error {
	if e.Repo == nil || e.Router == nil {
		return fmt.Errorf("kernel not initialized")
	}
	if executionID == "" {
		return fmt.Errorf("missing execution id")
	}

	now := e.now()
	var resumeErr error
	err := e.Repo.RunExecutionTx(ctx, func(ctx context.Context, tx repository.ExecutionTx) error {
		execution, err := tx.LoadExecution(ctx, executionID)
		if err != nil {
			return err
		}
		if isTerminalStage(execution.Stage) {
			return nil
		}

		runtime, err := tx.LoadRuntime(ctx, execution.RuntimeID)
		if err != nil {
			return err
		}
		snapshot, ok, err := tx.LoadRuntimeSnapshot(ctx, execution.RuntimeID)
		if err != nil {
			return err
		}
		var snapshotPtr *types.RuntimeSnapshot
		if ok {
			snapshotCopy := snapshot
			snapshotPtr = &snapshotCopy
		}
		requirements, err := resolveRuntimeRequirements(ctx, e.Requirements, runtime)
		if err != nil {
			return err
		}

		// Resume advances exactly one durable stage per call. Callers that want a
		// synchronous user reply are expected to loop until the execution becomes terminal.
		switch execution.Stage {
		case types.ExecutionAccepted, types.ExecutionPlanned:
			module, err := e.Router.Resolve(ctx, runtime, types.InboundEnvelope{
				RuntimeID: execution.RuntimeID,
				RouteHint: string(execution.ModeID),
			})
			if err != nil {
				return err
			}

			result, err := module.Execute(ctx, types.ModeExecutionInput{
				Runtime:      runtime,
				Execution:    execution,
				Snapshot:     snapshotPtr,
				Requirements: requirements,
				Command: types.ModeCommand{
					CommandType: execution.CommandType,
					Text:        execution.CommandText,
				},
			})
			if err != nil {
				resumeErr = err
				return e.failExecution(ctx, tx, execution, now, err)
			}
			if err := e.applyModeResult(ctx, tx, &runtime, &execution, &snapshotPtr, result, now); err != nil {
				resumeErr = err
				return e.failExecution(ctx, tx, execution, now, err)
			}
			return nil
		case types.ExecutionAwaitingArtifacts:
			if err := e.completeArtifactStage(ctx, tx, runtime, &execution, snapshotPtr, now); err != nil {
				resumeErr = err
				return e.failExecution(ctx, tx, execution, now, err)
			}
			return nil
		case types.ExecutionSettled:
			if err := e.projectExecution(ctx, tx, runtime, &execution, snapshotPtr, now); err != nil {
				resumeErr = err
				return e.failExecution(ctx, tx, execution, now, err)
			}
			return nil
		case types.ExecutionProjected:
			if err := e.deliverExecution(ctx, tx, &execution, now); err != nil {
				resumeErr = err
				return e.failExecution(ctx, tx, execution, now, err)
			}
			return nil
		default:
			return fmt.Errorf("unsupported execution stage: %s", execution.Stage)
		}
	})
	if err != nil {
		return err
	}
	return resumeErr
}

func (e SimpleEngine) applyModeResult(
	ctx context.Context,
	tx repository.ExecutionTx,
	runtime *types.RuntimeRecord,
	execution *types.ExecutionRecord,
	snapshotPtr **types.RuntimeSnapshot,
	result types.ModeExecutionResult,
	now time.Time,
) error {
	// Agent tasks are persisted before the stage advances so a restart never loses
	// the fact that the mode asked for external work.
	if len(result.AgentTasks) > 0 {
		if e.Supervisor == nil {
			return fmt.Errorf("agent supervisor not initialized")
		}
		taskRecords := buildAgentTaskRecords(*execution, result.AgentTasks, now)
		for _, task := range taskRecords {
			if err := tx.SaveAgentTask(ctx, task); err != nil {
				return err
			}
		}
		if err := e.Supervisor.StartTasks(ctx, *execution, result.AgentTasks); err != nil {
			return err
		}
	}

	if result.Snapshot != nil {
		if result.Snapshot.RuntimeID == "" {
			result.Snapshot.RuntimeID = execution.RuntimeID
		}
		if result.Snapshot.ModeID == "" {
			result.Snapshot.ModeID = execution.ModeID
		}
		if result.Snapshot.UpdatedAtUnix == 0 {
			result.Snapshot.UpdatedAtUnix = now.Unix()
		}
		if err := tx.SaveRuntimeSnapshot(ctx, *result.Snapshot); err != nil {
			return err
		}
		runtime.HeadVersion = result.Snapshot.Version
		if err := tx.SaveRuntime(ctx, *runtime); err != nil {
			return err
		}
		snapshotCopy := *result.Snapshot
		*snapshotPtr = &snapshotCopy
	} else if result.RuntimeVersionDelta != 0 {
		runtime.HeadVersion += result.RuntimeVersionDelta
		if err := tx.SaveRuntime(ctx, *runtime); err != nil {
			return err
		}
	}

	nextStage := result.NextStage
	if nextStage == "" {
		nextStage = types.ExecutionSettled
	}
	execution.PendingDelivery = nil
	if len(result.AgentTasks) > 0 && nextStage == types.ExecutionSettled {
		nextStage = types.ExecutionAwaitingArtifacts
	}
	if err := e.persistExecutionStage(ctx, tx, execution, nextStage, now); err != nil {
		return err
	}
	for _, event := range result.Events {
		if event.RuntimeID == "" {
			event.RuntimeID = execution.RuntimeID
		}
		if event.ExecutionID == "" {
			event.ExecutionID = execution.ExecutionID
		}
		if event.OccurredAtUnix == 0 {
			event.OccurredAtUnix = now.Unix()
		}
		if err := tx.AppendEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (e SimpleEngine) completeArtifactStage(
	ctx context.Context,
	tx repository.ExecutionTx,
	runtime types.RuntimeRecord,
	execution *types.ExecutionRecord,
	snapshot *types.RuntimeSnapshot,
	now time.Time,
) error {
	if e.Supervisor == nil {
		return fmt.Errorf("agent supervisor not initialized")
	}

	// Artifact collection may be retried many times. The execution remains in the
	// awaiting_artifacts stage until a durable artifact is available for the mode.
	tasks, err := tx.ListAgentTasksByExecution(ctx, execution.ExecutionID)
	if err != nil {
		return err
	}
	artifacts, err := tx.ListAgentArtifactsByExecution(ctx, execution.ExecutionID)
	if err != nil {
		return err
	}
	if len(artifacts) == 0 {
		pendingTasks := queuedAgentTaskSpecs(tasks)
		if len(pendingTasks) == 0 {
			return e.persistExecutionStage(ctx, tx, execution, types.ExecutionAwaitingArtifacts, now)
		}
		if err := e.Supervisor.StartTasks(ctx, *execution, pendingTasks); err != nil {
			return err
		}
		collected, err := e.Supervisor.Collect(ctx, execution.ExecutionID)
		if err != nil {
			return err
		}
		if len(collected) == 0 {
			return e.persistExecutionStage(ctx, tx, execution, types.ExecutionAwaitingArtifacts, now)
		}
		for _, artifact := range collected {
			if artifact.ExecutionID == "" {
				artifact.ExecutionID = execution.ExecutionID
			}
			if artifact.RuntimeID == "" {
				artifact.RuntimeID = execution.RuntimeID
			}
			if artifact.ArtifactID == "" {
				artifact.ArtifactID = artifact.TaskID + ":artifact"
			}
			if err := tx.SaveAgentArtifact(ctx, artifact); err != nil {
				return err
			}
			artifacts = append(artifacts, artifact)
		}
		for _, task := range tasks {
			if task.Status != types.AgentTaskQueued {
				continue
			}
			if hasArtifactForTask(artifacts, task.TaskID) {
				task.Status = types.AgentTaskCompleted
				task.CompletedAtUnix = now.Unix()
				task.LastError = ""
				if err := tx.SaveAgentTask(ctx, task); err != nil {
					return err
				}
			}
		}
	}

	if len(artifacts) == 0 {
		return e.persistExecutionStage(ctx, tx, execution, types.ExecutionAwaitingArtifacts, now)
	}

	// Once artifacts exist, the same mode re-enters and explicitly folds them
	// back into canonical state through the normal mode execution path.
	module, err := e.Router.Resolve(ctx, runtime, types.InboundEnvelope{
		RuntimeID: execution.RuntimeID,
		RouteHint: string(execution.ModeID),
	})
	if err != nil {
		return err
	}
	requirements, err := resolveRuntimeRequirements(ctx, e.Requirements, runtime)
	if err != nil {
		return err
	}

	result, err := module.Execute(ctx, types.ModeExecutionInput{
		Runtime:      runtime,
		Execution:    *execution,
		Snapshot:     snapshot,
		Requirements: requirements,
		Artifacts:    artifacts,
		Command: types.ModeCommand{
			CommandType: execution.CommandType,
			Text:        execution.CommandText,
		},
	})
	if err != nil {
		return err
	}
	return e.applyModeResult(ctx, tx, &runtime, execution, &snapshot, result, now)
}

func (e SimpleEngine) projectExecution(
	ctx context.Context,
	tx repository.ExecutionTx,
	runtime types.RuntimeRecord,
	execution *types.ExecutionRecord,
	snapshot *types.RuntimeSnapshot,
	now time.Time,
) error {
	execution.PendingDelivery = nil
	if e.Projector == nil || snapshot == nil {
		return e.persistExecutionStage(ctx, tx, execution, types.ExecutionCommitted, now)
	}

	projected, err := e.Projector.Project(ctx, types.ProjectionInput{
		Runtime:   runtime,
		Execution: *execution,
		Snapshot:  snapshot,
	})
	if err != nil {
		return err
	}

	for _, frame := range projected.Frames {
		if frame.RuntimeID == "" {
			frame.RuntimeID = execution.RuntimeID
		}
		if frame.ExecutionID == "" {
			frame.ExecutionID = execution.ExecutionID
		}
		if frame.ModeID == "" {
			frame.ModeID = execution.ModeID
		}
		if frame.CreatedAtUnix == 0 {
			frame.CreatedAtUnix = now.Unix()
		}
		if err := tx.SaveProjectionFrame(ctx, frame); err != nil {
			return err
		}
	}

	// Delivery intents are persisted inside the execution so projected work can
	// survive a restart before the dispatcher sees it.
	plans := normalizeDeliveryPlans(projected.DeliveryPlans, *execution)
	execution.PendingDelivery = plans
	if len(projected.Frames) == 0 && len(plans) == 0 {
		return e.persistExecutionStage(ctx, tx, execution, types.ExecutionCommitted, now)
	}
	return e.persistExecutionStage(ctx, tx, execution, types.ExecutionProjected, now)
}

func (e SimpleEngine) deliverExecution(
	ctx context.Context,
	tx repository.ExecutionTx,
	execution *types.ExecutionRecord,
	now time.Time,
) error {
	if len(execution.PendingDelivery) == 0 || e.Dispatcher == nil {
		execution.PendingDelivery = nil
		return e.persistExecutionStage(ctx, tx, execution, types.ExecutionCommitted, now)
	}

	for _, plan := range execution.PendingDelivery {
		job, err := e.Dispatcher.Enqueue(ctx, plan)
		if err != nil {
			return err
		}
		if err := tx.SaveDeliveryJob(ctx, job); err != nil {
			return err
		}
	}
	execution.PendingDelivery = nil
	return e.persistExecutionStage(ctx, tx, execution, types.ExecutionDelivered, now)
}

func (e SimpleEngine) persistExecutionStage(
	ctx context.Context,
	tx repository.ExecutionTx,
	execution *types.ExecutionRecord,
	stage types.ExecutionStage,
	now time.Time,
) error {
	execution.Stage = stage
	execution.UpdatedAtUnix = now.Unix()
	execution.LastError = ""
	if isTerminalStage(stage) {
		execution.LeaseOwner = ""
		execution.LeaseExpiresAtUnix = 0
	} else {
		execution.LeaseOwner = e.owner()
		execution.LeaseExpiresAtUnix = now.Add(e.leaseTTL()).Unix()
	}
	if err := tx.SaveExecution(ctx, *execution); err != nil {
		return err
	}
	return tx.AppendEvent(ctx, types.RuntimeEvent{
		EventID:        execution.ExecutionID + ":" + string(stage),
		RuntimeID:      execution.RuntimeID,
		ExecutionID:    execution.ExecutionID,
		Kind:           "execution." + string(stage),
		OccurredAtUnix: now.Unix(),
	})
}

func (e SimpleEngine) failExecution(
	ctx context.Context,
	tx repository.ExecutionTx,
	execution types.ExecutionRecord,
	now time.Time,
	cause error,
) error {
	// Failure is durable state too. We do not leave an execution silently parked
	// in a pre-failure stage when the operator needs to inspect what went wrong.
	execution.Stage = types.ExecutionFailed
	execution.LastError = cause.Error()
	execution.PendingDelivery = nil
	execution.LeaseOwner = ""
	execution.LeaseExpiresAtUnix = 0
	execution.UpdatedAtUnix = now.Unix()
	if err := tx.SaveExecution(ctx, execution); err != nil {
		return err
	}
	return tx.AppendEvent(ctx, types.RuntimeEvent{
		EventID:        execution.ExecutionID + ":failed",
		RuntimeID:      execution.RuntimeID,
		ExecutionID:    execution.ExecutionID,
		Kind:           "execution.failed",
		Text:           cause.Error(),
		OccurredAtUnix: now.Unix(),
	})
}

func normalizeDeliveryPlans(plans []types.DeliveryPlan, execution types.ExecutionRecord) []types.DeliveryPlan {
	if len(plans) == 0 {
		return nil
	}

	out := make([]types.DeliveryPlan, 0, len(plans))
	for _, plan := range plans {
		if plan.RuntimeID == "" {
			plan.RuntimeID = execution.RuntimeID
		}
		if plan.ExecutionID == "" {
			plan.ExecutionID = execution.ExecutionID
		}
		if plan.RecipientID == "" {
			plan.RecipientID = execution.ActorID
		}
		out = append(out, plan)
	}
	return out
}

func buildAgentTaskRecords(execution types.ExecutionRecord, specs []types.AgentTaskSpec, now time.Time) []types.AgentTaskRecord {
	records := make([]types.AgentTaskRecord, 0, len(specs))
	for _, task := range specs {
		records = append(records, types.AgentTaskRecord{
			TaskID:        task.TaskID,
			ExecutionID:   execution.ExecutionID,
			RuntimeID:     execution.RuntimeID,
			TaskType:      task.TaskType,
			Input:         task.Input,
			ArtifactType:  task.ArtifactType,
			Status:        types.AgentTaskQueued,
			CreatedAtUnix: now.Unix(),
		})
	}
	return records
}

func queuedAgentTaskSpecs(tasks []types.AgentTaskRecord) []types.AgentTaskSpec {
	specs := make([]types.AgentTaskSpec, 0)
	for _, task := range tasks {
		if task.Status != types.AgentTaskQueued {
			continue
		}
		specs = append(specs, types.AgentTaskSpec{
			TaskID:       task.TaskID,
			TaskType:     task.TaskType,
			RuntimeID:    task.RuntimeID,
			Input:        task.Input,
			ArtifactType: task.ArtifactType,
		})
	}
	return specs
}

func hasArtifactForTask(artifacts []types.AgentArtifact, taskID string) bool {
	for _, artifact := range artifacts {
		if artifact.TaskID == taskID {
			return true
		}
	}
	return false
}

func (e SimpleEngine) owner() string {
	if e.Owner != "" {
		return e.Owner
	}
	return "kernel"
}

func (e SimpleEngine) leaseTTL() time.Duration {
	if e.LeaseTTL > 0 {
		return e.LeaseTTL
	}
	return time.Minute
}

func (e SimpleEngine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func executionIDFor(env types.InboundEnvelope) string {
	if env.EnvelopeID != "" {
		return env.EnvelopeID
	}
	return env.RuntimeID + ":" + env.IdempotencyKey
}

func isTerminalStage(stage types.ExecutionStage) bool {
	switch stage {
	case types.ExecutionCommitted, types.ExecutionDelivered, types.ExecutionFailed:
		return true
	default:
		return false
	}
}

var _ Engine = (*SimpleEngine)(nil)
