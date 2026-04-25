package restartsim

import "babel-runtime/internal/testkit/harness"

type MultiplayerSimulator struct {
	SoloSimulator
}

func CaptureMultiplayer(h *harness.MultiplayerHarness) MultiplayerSimulator {
	if h == nil {
		return MultiplayerSimulator{}
	}
	return MultiplayerSimulator{
		SoloSimulator: SoloSimulator{Snapshot: h.Snapshot()},
	}
}

func (s MultiplayerSimulator) Restore() *harness.MultiplayerHarness {
	mem := newStoreFromSnapshot(s.Snapshot)
	return harness.NewMultiplayerHarnessWithStore(mem)
}
