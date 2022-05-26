package vm

import (
	"bytes"
	"encoding/json"
	"sync"
	"xfsgo/common"
	"xfsgo/common/ahash"
	"xfsgo/core"
	"xfsgo/storage/badger"
)

type LogStorage interface {
	PutAllEvents(tx common.Hash, address common.Address, events []Event)
	SaveEvents(block core.IBlock) error
	GetEventLogs(block common.Hash) ([]*EventObj, bool)
	GetEventLogsByAddress(block common.Hash, address common.Address) ([]*EventObj, bool)
}

type EventObj struct {
	BlockHeight     uint64         `json:"block_number"`
	BlockHash       common.Hash    `json:"block_hash"`
	TransactionHash common.Hash    `json:"transaction_hash"`
	EventHash       common.Hash    `json:"event_hash"`
	EventValue      []byte         `json:"event_value"`
	Address         common.Address `json:"address"`
}

type logStorage struct {
	db     badger.IStorage
	caches map[common.Hash][]Event
	lock   sync.Mutex
}

var (
	blockKeyPrefix   = []byte("blk:")
	eventKeyPrefix   = []byte("event:")
	addressKeyPrefix = []byte("address:")
)

func NewLogStorage(db badger.IStorage) *logStorage {
	return &logStorage{
		db:     db,
		caches: make(map[common.Hash][]Event),
	}
}

func (l *logStorage) PutAllEvents(tx common.Hash, address common.Address, events []Event) {
	l.lock.Lock()
	defer l.lock.Unlock()
	for i, _ := range events {
		events[i].Address = address
	}
	l.caches[tx] = events
}

func (l *logStorage) makeBlockIndexKey(prefix []byte, key []byte) []byte {
	return append(blockKeyPrefix, append(prefix, key...)...)
}
func (l *logStorage) makeEventKey(key []byte) []byte {
	return append(eventKeyPrefix, key...)
}
func (l *logStorage) makeAddressKey(prefix []byte, key []byte) []byte {
	return append(addressKeyPrefix, append(prefix, key...)...)
}
func (l *logStorage) makeHash(block, tx, event common.Hash) common.Hash {
	buf := bytes.NewBuffer(nil)
	buf.Write(block[:])
	buf.Write(tx[:])
	buf.Write(event[:])
	return ahash.SHA256Array(buf.Bytes())
}
func (l *logStorage) SaveEvents(block core.IBlock) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	blockHash := block.HeaderHash()
	objs := make(map[common.Hash]*EventObj)
	for key, events := range l.caches {
		for _, event := range events {
			obj := &EventObj{
				BlockHeight:     block.Height(),
				BlockHash:       blockHash,
				TransactionHash: key,
				EventHash:       event.Hash,
				EventValue:      event.Value,
				Address:         event.Address,
			}
			hash := l.makeHash(blockHash, key, event.Hash)
			objs[hash] = obj
		}
	}
	l.caches = nil
	l.caches = make(map[common.Hash][]Event)
	batchWriter := l.db.NewWriteBatch()
	for key, value := range objs {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		valueAddress := value.Address
		objhash := l.makeEventKey(key[:])
		addressKey := l.makeAddressKey(append(valueAddress[:], []byte(":")...), objhash)
		eventKey := l.makeBlockIndexKey(append(blockHash[:], []byte(":")...), addressKey)
		if err = batchWriter.Put(eventKey, data); err != nil {
			return err
		}
	}
	return l.db.CommitWriteBatch(batchWriter)
}

func (l *logStorage) GetEventLogs(block common.Hash) ([]*EventObj, bool) {
	eventPre := l.makeBlockIndexKey(append(block[:], []byte(":")...), nil)
	evenLogs := make([]*EventObj, 0)
	err := l.db.PrefixForeachData(eventPre, func(k []byte, v []byte) error {
		obj := &EventObj{}
		err := json.Unmarshal(v, obj)
		if err != nil {
			return err
		}
		evenLogs = append(evenLogs, obj)
		return nil
	})
	return evenLogs, err == nil
}
func (l *logStorage) GetEventLogsByAddress(block common.Hash, address common.Address) ([]*EventObj, bool) {
	addressKey := l.makeAddressKey(append(address[:], []byte(":")...), nil)
	eventPre := l.makeBlockIndexKey(append(block[:], []byte(":")...), addressKey)
	evenLogs := make([]*EventObj, 0)
	err := l.db.PrefixForeachData(eventPre, func(k []byte, v []byte) error {
		obj := &EventObj{}
		err := json.Unmarshal(v, obj)
		if err != nil {
			return err
		}
		evenLogs = append(evenLogs, obj)
		return nil
	})
	return evenLogs, err == nil
}
