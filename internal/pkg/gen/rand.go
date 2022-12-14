package gen

import (
	"math/rand"
	"sync"
	"time"
)

var defaultRand = rand.New(newSource(time.Now().UnixNano()))

// source implements the rand.Source and rand.Source64 interfaces and is
// safe for concurrent use.
type source struct {
	mu  sync.Mutex
	src rand.Source64
}

func newSource(seed int64) *source {
	return &source{src: rand.NewSource(seed).(rand.Source64)}
}

func (r *source) Int63() int64 {
	r.mu.Lock()
	n := r.src.Int63()
	r.mu.Unlock()

	return n
}

func (r *source) Uint64() uint64 {
	r.mu.Lock()
	n := r.src.Uint64()
	r.mu.Unlock()

	return n
}

func (r *source) Seed(seed int64) {
	r.mu.Lock()
	r.src.Seed(seed)
	r.mu.Unlock()
}
