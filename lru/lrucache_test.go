package lru

import (
	"fmt"
	"testing"
	"xfsgo/common/ahash"
)

func makeKey(s string) (out [32]byte) {
	copy(out[:], ahash.SHA256([]byte(s)))
	return
}
func TestCache_Get(t *testing.T) {
	size := 1000
	cache := NewCache(size)
	setCache := func(num int) {
		for i := 0; i < num; i++ {
			key := makeKey(fmt.Sprintf("key%d", i))
			cache.Put(key, []byte(fmt.Sprintf("val%d", i)))
		}
	}
	setCache(size)
	for i := 0; i < size; i++ {
		keystr := fmt.Sprintf("key%d", i)
		key := makeKey(keystr)
		val, exists := cache.Get(key)
		if !exists || val == nil {
			t.Fatalf("not found key: %s", keystr)
		}
	}
	key := makeKey(fmt.Sprintf("key%d", size+1))
	val, exists := cache.Get(key)
	if exists || val != nil {
		t.Fatalf("Invalid query")
	}
}
