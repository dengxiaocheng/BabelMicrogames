package restartsim

import "babel-runtime/internal/store"

func newStoreFromSnapshot(snapshot store.MemorySnapshot) *store.MemoryStore {
	return store.NewMemoryStoreFromSnapshot(snapshot)
}
