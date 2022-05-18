package test

import (
	"encoding/json"
	"io"
	"net"
	"time"
	"xfsgo/p2p"
	"xfsgo/p2p/discover"
)

type simplePackReader struct {
	data  []byte
	mtype uint8
}

func newSimplePackReader(mt uint8, data []byte) *simplePackReader {
	return &simplePackReader{
		mtype: mt,
		data:  data,
	}
}
func (pr *simplePackReader) Type() uint8 {
	return pr.mtype
}
func (pr *simplePackReader) Read(p []byte) (int, error) { return 0, nil }
func (pr *simplePackReader) ReadAll() ([]byte, error) {
	return pr.data, nil
}
func (pr *simplePackReader) RawReader() io.Reader {
	return nil
}
func (pr *simplePackReader) DataReader() io.Reader {
	return nil
}

type handleOnPeerData func(mType uint8, data []byte) (uint8, []byte, error)

type BufferPeer struct {
	selfId     discover.NodeId
	remoteNode *discover.Node
	timeTTL    time.Duration
	dataCh     chan p2p.MessageReader
	closed     chan struct{}
	handleFn   handleOnPeerData
}

func NewBufferPeer(selfId discover.NodeId, remoteId *discover.Node, handleFn handleOnPeerData) *BufferPeer {
	return &BufferPeer{
		selfId:     selfId,
		remoteNode: remoteId,
		dataCh:     make(chan p2p.MessageReader),
		closed:     make(chan struct{}),
		handleFn:   handleFn,
	}
}

func (bp *BufferPeer) Is(flag int) bool {
	return false
}
func (bp *BufferPeer) ID() discover.NodeId {
	return bp.selfId
}
func (bp *BufferPeer) RemoteNode() *discover.Node {
	return bp.remoteNode
}
func (bp *BufferPeer) RemoteAddr() *net.TCPAddr {
	return bp.remoteNode.TcpAddr()
}
func (bp *BufferPeer) Close() {
	close(bp.closed)
}
func (bp *BufferPeer) Run() {

}
func (bp *BufferPeer) WriteMessage(mType uint8, data []byte) error {
	time.Sleep(bp.timeTTL)
	select {
	case <-bp.closed:
		return io.EOF
	default:
	}

	var (
		err     error
		repType uint8
		repData []byte
	)
	if bp.handleFn != nil {
		repType, repData, err = bp.handleFn(mType, data)
	} else {
		repType, repData = mType, data
	}
	if err != nil {
		return err
	}
	bp.dataCh <- newSimplePackReader(repType, repData)
	return nil
}
func (bp *BufferPeer) WriteMessageObj(mType uint8, data interface{}) error {
	dataobj, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return bp.WriteMessage(mType, dataobj)
}
func (bp *BufferPeer) GetProtocolMsgCh() (chan p2p.MessageReader, error) {
	select {
	case <-bp.closed:
		return nil, io.EOF
	default:
	}
	return bp.dataCh, nil
}
