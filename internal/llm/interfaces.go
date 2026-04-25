package llm

import "context"

type Renderer interface {
	RenderSolo(ctx context.Context, req SoloRenderRequest) (SoloRenderResponse, error)
}

