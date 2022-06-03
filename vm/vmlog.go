package vm

import (
	"encoding/json"
	"reflect"
	"xfsgo/common"
	"xfsgo/common/ahash"
)

type Logger interface {
	Event(interface{})
	GetEvents() []Event
}
type Event struct {
	Hash    common.Hash
	Address common.Address
	Value   []byte
}
type logger struct {
	events []Event
}

func NewLogger() *logger {
	l := &logger{
		events: make([]Event, 0),
	}
	return l
}
func (l *logger) Event(e interface{}) {
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	etype := reflect.TypeOf(e)
	etypename := etype.Elem().Name()
	namehash := ahash.SHA256Array([]byte(etypename))
	l.events = append(l.events, Event{
		Hash:  namehash,
		Value: data,
	})
}
func (l *logger) GetEvents() []Event {
	return l.events
}
