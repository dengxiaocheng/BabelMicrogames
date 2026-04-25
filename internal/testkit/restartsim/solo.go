package restartsim

import (
	"babel-runtime/internal/store"
	"babel-runtime/internal/testkit/harness"
)

type SoloSimulator struct {
	Snapshot store.MemorySnapshot
}

func CaptureSolo(h *harness.SoloHarness) SoloSimulator {
	if h == nil {
		return SoloSimulator{}
	}
	return SoloSimulator{Snapshot: h.Snapshot()}
}

func (s SoloSimulator) Restore() *harness.SoloHarness {
	mem := newStoreFromSnapshot(s.Snapshot)
	return harness.NewSoloHarnessWithStore(mem)
}
