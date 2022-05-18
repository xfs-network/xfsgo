package backend

import (
	"bytes"
	"encoding/json"
	"io"
	"xfsgo/p2p/discover"
)

type handlerMgr interface {
	Handle(t uint8, cb handlerFn)
}

type request struct {
	peerId discover.NodeId
	data   []byte
}

func (r *request) reader() io.Reader {
	return bytes.NewBuffer(r.data)
}

func (r *request) dataString() string {
	return string(r.data)
}

func (r *request) jsonObj(v interface{}) error {
	return json.Unmarshal(r.data, v)
}

type handlerFn func(req *request, sp sender) error

type mHandlerMgr struct {
	handlers map[uint8]handlerFn
}

func newHandlerMgr() *mHandlerMgr {
	return &mHandlerMgr{
		handlers: make(map[uint8]handlerFn),
	}
}

func (mgr *mHandlerMgr) Handle(t uint8, cb handlerFn) {
	if _, exists := mgr.handlers[t]; exists {
		return
	}
	mgr.handlers[t] = cb
}

func (mgr *mHandlerMgr) OnMessage(id discover.NodeId, p sender, t uint8, data []byte) error {
	if fn, exists := mgr.handlers[t]; exists {
		rq := &request{
			peerId: id,
			data:   data,
		}
		return fn(rq, p)
	}
	return nil
}
