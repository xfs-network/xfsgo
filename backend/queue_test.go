package backend

import (
	"crypto/rand"
	"testing"
	"xfsgo/common"
)

func randomHashes(n int) []common.Hash {
	arr := make([]common.Hash, n)
	for i := 0; i < n; i++ {
		var hash common.Hash
		_, _ = rand.Read(hash[:])
		arr[i] = hash
	}
	return arr
}
func TestSyncQueue_Insert(t *testing.T) {
	q := newSyncQueue()
	hashes := randomHashes(10)
	q.Insert(hashes)
}
