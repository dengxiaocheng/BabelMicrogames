package recovery

import "context"

type Report struct {
	RecoveredRuntimes int `json:"recovered_runtimes"`
	StaleCheckpoints  int `json:"stale_checkpoints"`
	RetriedSteps      int `json:"retried_steps"`
	RedeliveredFrames int `json:"redelivered_frames"`
}

type Supervisor interface {
	Sweep(ctx context.Context) (Report, error)
}

