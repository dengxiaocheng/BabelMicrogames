package kernel

import (
	"context"

	"babel-runtime/internal/core/types"
	"babel-runtime/internal/requirementregistry"
)

func resolveRuntimeRequirements(
	ctx context.Context,
	registry requirementregistry.Registry,
	runtime types.RuntimeRecord,
) (types.RuntimeRequirements, error) {
	if registry == nil {
		return types.RuntimeRequirements{}, nil
	}
	var requirements types.RuntimeRequirements
	if runtime.RulesetID != "" {
		ruleset, err := registry.ResolveRuleset(ctx, runtime.RulesetID)
		if err != nil {
			return types.RuntimeRequirements{}, err
		}
		requirements.Ruleset = &ruleset
	}
	if runtime.PromptPackID != "" {
		promptPack, err := registry.ResolvePromptPack(ctx, runtime.PromptPackID)
		if err != nil {
			return types.RuntimeRequirements{}, err
		}
		requirements.PromptPack = &promptPack
	}
	if runtime.GameplayAssetID != "" {
		gameplayAsset, err := registry.ResolveGameplayAsset(ctx, runtime.GameplayAssetID)
		if err != nil {
			return types.RuntimeRequirements{}, err
		}
		requirements.GameplayAsset = &gameplayAsset
	}
	return requirements, nil
}
