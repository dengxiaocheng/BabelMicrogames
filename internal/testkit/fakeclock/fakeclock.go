package fakeclock

import "babel-runtime/internal/core/types"

type FakeClock struct {
	Now types.WorldClock
}

func New(clock types.WorldClock) *FakeClock {
	return &FakeClock{Now: clock}
}

func (f *FakeClock) Current() types.WorldClock {
	return f.Now
}

func (f *FakeClock) Set(clock types.WorldClock) {
	f.Now = clock
}

