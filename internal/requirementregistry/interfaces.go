package requirementregistry

import (
	"context"
	"errors"

	"babel-runtime/internal/core/types"
)

var ErrAssetNotFound = errors.New("requirement asset not found")

type Registry interface {
	ResolveRuleset(ctx context.Context, rulesetID string) (types.RulesetBundle, error)
	ResolvePromptPack(ctx context.Context, promptPackID string) (types.PromptPackBundle, error)
	ResolveGameplayAsset(ctx context.Context, assetID string) (types.GameplayAssetBundle, error)
	ResolveContentConstraint(ctx context.Context, constraintID string) (types.ContentConstraintBundle, error)
	ResolveExperimentTrace(ctx context.Context, traceID string) (types.ExperimentTraceBundle, error)
}
