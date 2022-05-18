package p2p

import (
	"bytes"
	"errors"
	"io"
	"net"
	"time"
	"xfsgo/log"
	"xfsgo/p2p/discover"
)

const pingloopinterval = 1
const aliveloopinterval = 1
const alivemaxinterval = 10

type encoder interface {
	Encode(obj interface{}) ([]byte, error)
}
type Peer interface {
	Is(flag int) bool
	ID() discover.NodeId
	RemoteNode() *discover.Node
	RemoteAddr() *net.TCPAddr
	Close()
	Run()
	WriteMessage(mType uint8, data []byte) error
	WriteMessageObj(mType uint8, data interface{}) error
	GetProtocolMsgCh() (chan MessageReader, error)
}

type peer struct {
	id       discover.NodeId
	conn     *peerConn
	rw       net.Conn
	close    chan struct{}
	lastTime int64
	readBuf  bytes.Buffer
	ps       []Protocol
	quit     chan struct{}
	psCh     chan MessageReader
	encoder  encoder
	logger   log.Logger
}

// create peer [Peer to peer connection session,Network protocol]
func newPeer(conn *peerConn, ps []Protocol, en encoder) Peer {
	p := &peer{
		conn:    conn,
		id:      conn.id,
		rw:      conn.rw,
		logger:  conn.logger,
		ps:      ps,
		close:   make(chan struct{}),
		psCh:    make(chan MessageReader),
		encoder: en,
	}
	now := time.Now()
	p.lastTime = now.Unix()
	return p
}
func (p *peer) RemoteNode() *discover.Node {
	addr := p.RemoteAddr()
	return discover.NewNode(addr.IP, uint16(addr.Port), uint16(addr.Port), p.id)
}
func (p *peer) RemoteAddr() *net.TCPAddr {
	addr := p.rw.RemoteAddr().(*net.TCPAddr)
	return addr
}
func (p *peer) ID() discover.NodeId {
	return p.id
}

func (p *peer) Is(flag int) bool {
	return p.conn.flag&flag != 0
}

// Read heartbeat message
func (p *peer) readLoop() {
	for {
		select {
		case <-p.close:
			return
		default:
		}
		msg, err := ReadMessage(p.rw)
		if err != nil {
			p.Close()
			return
		}
		p.handle(msg)
	}
}

func (p *peer) handle(msg MessageReader) {
	data, err := msg.ReadAll()
	if err != nil {
		return
	}
	//p.logger.Infof("peer handle message type %d, data: %s", msg.Type(), string(data))
	switch msg.Type() {
	case typePingMsg:
		//p.logger.Debugln("receive heartbeat request")
		err = p.conn.writeMessage(typePongMsg, []byte("hello"))
		if err != nil {
			p.Close()
		}
	case typePongMsg:
		//p.logger.Debugln("receive response of heartbeat and update alive time")
		now := time.Now()
		p.lastTime = now.Unix()
	default:
		bodyBs := msg.RawReader()
		cpy := &messageReader{
			raw:   bodyBs,
			mType: msg.Type(),
			data:  bytes.NewReader(data),
		}
		p.psCh <- cpy
		_, _ = io.Copy(&p.readBuf, bodyBs)
	}
}

func (p *peer) GetProtocolMsgCh() (chan MessageReader, error) {
	select {
	case _ = <-p.close:
		return nil, errors.New("peer closed")
	default:
	}
	return p.psCh, nil
}

func (p *peer) WriteMessage(mType uint8, bs []byte) error {
	select {
	case _ = <-p.close:
		return errors.New("peer closed")
	default:
	}
	return p.conn.writeMessage(mType, bs)
}

func (p *peer) WriteMessageObj(mType uint8, obj interface{}) error {
	bs, err := p.encoder.Encode(obj)
	if err != nil {
		return err
	}
	return p.WriteMessage(mType, bs)
}

func (p *peer) pingLoop() {
	ping := time.NewTicker(pingloopinterval * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-ping.C:
			if err := p.conn.writeMessage(typePingMsg, []byte("hello")); err != nil {
				p.Close()
				return
			}
		case <-p.close:
			return
		}
	}
}

func (p *peer) suicide() {
	for {
		select {
		case <-p.close:
			return
		default:
		}
		now := time.Now()
		nowTime := now.Unix()
		interval := nowTime - p.lastTime
		// 10s
		if interval > alivemaxinterval {
			p.Close()
			return
		}
		time.Sleep(aliveloopinterval * time.Second)
	}
}

func (p *peer) Run() {
	defer p.conn.close()
	go p.readLoop()
	go p.pingLoop()
	runProtocol := func() {
		for _, item := range p.ps {
			go func(p *peer, item Protocol) {
				err := item.Run(p)
				if err != nil {
					p.Close()
				}
			}(p, item)
		}
	}
	runProtocol()
	go p.suicide()
	for {
		select {
		case <-p.close:
			return
		}
	}

}

func (p *peer) Close() {
	select {
	case _ = <-p.close:
		return
	default:
	}
	close(p.close)
}
