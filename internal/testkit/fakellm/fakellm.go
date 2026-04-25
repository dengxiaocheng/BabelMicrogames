package fakellm

import (
	"context"

	"babel-runtime/internal/llm"
)

type Renderer struct {
	Response llm.SoloRenderResponse
	Err      error
	Calls    int
}

func (r *Renderer) RenderSolo(ctx context.Context, req llm.SoloRenderRequest) (llm.SoloRenderResponse, error) {
	r.Calls++
	_ = ctx
	_ = req
	if r.Err != nil {
		return llm.SoloRenderResponse{}, r.Err
	}
	return r.Response, nil
}

