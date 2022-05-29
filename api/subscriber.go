package api

import (
	"github.com/google/uuid"
	"sync"
	"xfsgo"
)

type subscriber struct {
	s    *EventSubscriber
	id   uuid.UUID
	conn xfsgo.RPCConn
	typ  int
}

func (s *subscriber) handleClose(code int, text string) error {
	s.s.mu.Lock()
	defer s.s.mu.Unlock()
	if s.typ == 0 {
		delete(s.s.newBlockSubscriber, s.id)
	}
	return nil
}

type EventSubscriber struct {
	eventBus           *xfsgo.EventBus
	mu                 sync.Mutex
	newBlockSubscriber map[uuid.UUID]*subscriber
}

func NewEventSubscriber(eventBus *xfsgo.EventBus) *EventSubscriber {
	sub := &EventSubscriber{
		eventBus:           eventBus,
		newBlockSubscriber: make(map[uuid.UUID]*subscriber),
	}
	sub.start()
	return sub
}

type BlockSubScribeRequest struct {
}

type EventSubScribeResponse struct {
	Subscription string `json:"subscription"`
}

type UnsubscribeRequest struct {
	Subscription string `json:"subscription"`
}

func (s *EventSubscriber) SubscribeNewBlock(
	conn xfsgo.RPCConn, request BlockSubScribeRequest,
	response **EventSubScribeResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := uuid.New()
	sub := &subscriber{
		s:    s,
		id:   id,
		conn: conn,
	}
	s.newBlockSubscriber[id] = sub
	conn.SetCloseHandler(sub.handleClose)
	responsev := &EventSubScribeResponse{
		Subscription: id.String(),
	}
	*response = responsev
	return nil
}
func (s *EventSubscriber) UnsubscribeNewBlock(conn xfsgo.RPCConn, request UnsubscribeRequest, status *int) error {
	if request.Subscription == "" {
		return nil
	}
	id, err := uuid.Parse(request.Subscription)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.newBlockSubscriber[id]; !exists {
		return nil
	}
	delete(s.newBlockSubscriber, id)
	*status = 1
	return nil
}
func (s *EventSubscriber) broadcastNewBlockEvent(block *xfsgo.Block) {
	for id, sub := range s.newBlockSubscriber {
		if sub == nil {
			return
		}
		_ = sub.conn.SendMessage(id, block)
	}
}
func (s *EventSubscriber) handleNewBlockEvent(ss *xfsgo.Subscription, data interface{}) {
	event, ok := data.(xfsgo.ChainHeadEvent)
	if !ok {
		return
	}
	s.broadcastNewBlockEvent(event.Block)
}
func (s *EventSubscriber) start() {
	chainHeadEventSub := s.eventBus.Subscript(xfsgo.ChainHeadEvent{})
	go chainHeadEventSub.AddLListener(s.handleNewBlockEvent)
}
