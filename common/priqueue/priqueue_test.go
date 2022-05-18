package priqueue

import (
	"testing"
	"xfsgo/assert"
)

func TestPriqueue_Push(t *testing.T) {
	blocksize := 10
	max := 100
	q := New(blocksize)
	for i := 0; i < max; i++ {
		q.Push(i, i)
	}
	assert.Equal(t, q.cont.size, max)
}
