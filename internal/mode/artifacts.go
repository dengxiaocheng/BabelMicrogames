package mode

import (
	"fmt"

	"babel-runtime/internal/core/types"
)

func assistantTextArtifact(artifacts []types.AgentArtifact) (string, error) {
	for _, artifact := range artifacts {
		if artifact.ArtifactType == "assistant_text" && artifact.Body != "" {
			return artifact.Body, nil
		}
	}
	return "", fmt.Errorf("missing assistant_text artifact")
}
