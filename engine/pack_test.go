package engine

import (
	"testing"
)

func BenchmarkPackRecycle(b *testing.B) {
	poolSize := 100
	inChan := make(chan *PipelinePack, poolSize)
	for i := 0; i < poolSize; i++ {
		pack := NewPipelinePack(inChan)
		inChan <- pack
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := <-inChan
		p.Recycle()
	}
}
