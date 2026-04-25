package types

type SoloRenderEnvelope struct {
	SessionID string               `json:"session_id"`
	Frame     RenderFrame          `json:"frame"`
	Delta     StepDelta            `json:"delta"`
}

