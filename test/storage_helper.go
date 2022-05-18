package test

import (
	"strings"
	"xfsgo/storage/badger"
)

type MemStorage struct {
	db      map[string][]byte
	version uint32
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		db:      make(map[string][]byte),
		version: Version,
	}
}

func (st *MemStorage) GetDBPath() string {
	return ""
}

func (st *MemStorage) Set(key string, val []byte) error {
	_ = st.SetData([]byte(key), val)
	return nil
}

func (st *MemStorage) SetData(key []byte, val []byte) error {
	st.db[string(key)] = val
	return nil
}
func (st *MemStorage) NewWriteBatch() *badger.StorageWriteBatch {
	return nil
}
func (st *MemStorage) CommitWriteBatch(batch *badger.StorageWriteBatch) error {
	return nil
}
func (st *MemStorage) Get(key string) ([]byte, error) {
	return st.db[key], nil
}

func (st *MemStorage) GetData(key []byte) (val []byte, err error) {
	return st.db[string(key)], nil
}
func (st *MemStorage) Del(key string) error {
	delete(st.db, key)
	return nil
}

func (st *MemStorage) DelData(key []byte) error {
	delete(st.db, string(key))
	return nil
}

func (st *MemStorage) Close() error { return nil }

func (st *MemStorage) Foreach(fn func(k string, v []byte) error) error {
	return st.ForeachData(func(k []byte, v []byte) error {
		return fn(string(k), v)
	})
}

func (st *MemStorage) ForeachData(fn func(k []byte, v []byte) error) error {
	for key, val := range st.db {
		return fn([]byte(key), val)
	}
	return nil
}

func (st *MemStorage) For(fn func(k []byte, v []byte)) {
	for key, val := range st.db {
		fn([]byte(key), val)
	}
}

func (st *MemStorage) ForIndex(fn func(n int, k []byte, v []byte)) {
	i := 0
	for key, val := range st.db {
		fn(i, []byte(key), val)
		i += 1
	}
}
func (st *MemStorage) ForIndexStar(start int, fn func(n int, k []byte, v []byte)) {
	i := 0
	for key, val := range st.db {
		if i < start {
			continue
		}
		fn(i, []byte(key), val)
		i += 1
	}

}
func (st *MemStorage) PrefixForeach(prefix string, fn func(k string, v []byte) error) error {
	return st.PrefixForeachData([]byte(prefix), func(k []byte, v []byte) error {
		return fn(string(k), v)
	})
}

func (st *MemStorage) PrefixForeachData(prefix []byte, fn func(k []byte, v []byte) error) error {
	for key, val := range st.db {
		if !strings.Contains(key, string(prefix)) {
			continue
		}
		return fn([]byte(key), val)
	}
	return nil
}

func (st *MemStorage) NewIterator() badger.Iterator { return nil }

func (st *MemStorage) GetVersion() uint32 { return st.version }
