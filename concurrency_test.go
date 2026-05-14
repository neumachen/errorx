package errorx_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/neumachen/errorx"
)

// TestConcurrentReadsAndMetadataWrites exercises every concurrent-safe
// method simultaneously. Run with -race for the assertion to bite.
func TestConcurrentReadsAndMetadataWrites(t *testing.T) {
	err := errorx.WrapPrefix(fmt.Errorf("root cause"), "ctx", 0)

	const goroutines = 16
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				switch (g + i) % 6 {
				case 0:
					md := json.RawMessage(fmt.Sprintf(`{"g":%d,"i":%d}`, g, i))
					if e := err.SetMetadata(&md); e != nil {
						t.Errorf("SetMetadata: %v", e)
						return
					}
				case 1:
					_ = err.Metadata()
				case 2:
					_ = err.StackFrames()
				case 3:
					_ = err.Stack()
				case 4:
					_ = err.Error()
				case 5:
					if _, e := json.Marshal(err); e != nil {
						t.Errorf("Marshal: %v", e)
						return
					}
				}
			}
		}(g)
	}
	wg.Wait()
}

// TestConcurrentStackFramesIdempotent ensures concurrent first-call resolves
// produce a consistent frame slice.
func TestConcurrentStackFramesIdempotent(t *testing.T) {
	err := errorx.NewError(errors.New("x"))

	const goroutines = 32
	results := make([][]errorx.StackFrame, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			results[g] = err.StackFrames()
		}(g)
	}
	wg.Wait()

	first := results[0]
	if len(first) == 0 {
		t.Fatal("no frames")
	}
	for i, r := range results[1:] {
		if len(r) != len(first) {
			t.Fatalf("goroutine %d: len=%d, want %d", i+1, len(r), len(first))
		}
		for j := range r {
			if r[j].ProgramCounter != first[j].ProgramCounter {
				t.Fatalf("goroutine %d frame %d differs", i+1, j)
			}
		}
	}
}
