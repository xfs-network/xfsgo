package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"io/ioutil"
	"net"
	"xfsgo/log"
	"xfsgo/p2p/discover"
)

var (
	errHandshakeFailed = errors.New("handshake failed")
)

// Peer to peer connection session
type peerConn struct {
	logger          log.Logger
	inbound         bool
	id              discover.NodeId
	self            discover.NodeId
	server          *server
	key             *ecdsa.PrivateKey
	rw              net.Conn
	version         uint8
	handshakeStatus int
	flag            int
}

func (c *peerConn) serve() {
	// Get the address and port number of the client
	fromAddr := c.rw.RemoteAddr()
	inbound := c.flag&flagInbound != 0
	if inbound {
		if err := c.serverHandshake(); err != nil {
			//c.logger.Errorf("handshake error server from %s: %v", fromAddr, err)
			c.close()
			return
		}
	} else {
		if err := c.clientHandshake(); err != nil {
			//c.logger.Errorf("handshake error client from %s: %v", fromAddr, err)
			c.close()
			return
		}
	}
	c.logger.Debugf("Successfully handshake by p2p transport: addr=%s, id=%s", fromAddr, c.id)
	c.server.addpeer <- c
}

//Client handshake sending method
func (c *peerConn) clientHandshake() error {

	// Whether the handshake status is based on handshake
	if c.handshakeCompiled() {
		return nil
	}
	request := &helloRequestMsg{
		version:   c.version,
		id:        c.self,
		receiveId: c.id,
	}
	//c.logger.Debugf("send hello request version: %d, id: %s, to receiveId: %s", c.version,c.self, c.id)
	_, err := c.rw.Write(request.marshal())
	if err != nil {
		return err
	}
	hello, err := c.readHelloReRequestMsg()

	if err != nil {
		return err
	}
	//c.logger.Debugf("receive hello request reply from node: %s, by version: %d", hello.id, c.version)
	if hello.version != c.version {
		//log.Errorf("handshake check err, got version: %d, want version: %d",
		//	hello.version, c.version)
		return errHandshakeFailed
	}
	gotId := hello.receiveId
	wantId := c.self
	if !bytes.Equal(gotId[:], wantId[:]) {
		//fmt.Errorf("handshake check err got my name: %x, my real name: %x",
		//	gotId[:], wantId[:])
		return errHandshakeFailed
	}
	c.handshakeStatus = 1
	return nil
}

// Service handshake response method
func (c *peerConn) serverHandshake() error {
	// Whether the handshake status is based on handshake
	if c.handshakeCompiled() {
		return nil
	}

	// Read reply data
	hello, err := c.readHelloRequestMsg()
	if err != nil {
		return err
	}
	if hello.version != c.version {
		return errHandshakeFailed
	}

	gotId := hello.receiveId
	wantId := c.self
	if !bytes.Equal(gotId[:], wantId[:]) {
		return errHandshakeFailed
	}
	c.id = hello.id

	reply := &helloReRequestMsg{
		id:        c.self,
		receiveId: hello.id,
		version:   c.version,
	}
	//c.logger.Debugf("send handshake reply to nodeId %s", reply.receiveId)
	if _, err = c.rw.Write(reply.marshal()); err != nil {
		return err
	}
	return nil
}

// Read reply message
func (c *peerConn) readHelloReRequestMsg() (*helloReRequestMsg, error) {
	msg, err := c.readMessage()
	if err != nil {
		return nil, err
	}
	if msg.Type() != typeReHelloRequest {
		return nil, err
	}
	nMsg := new(helloReRequestMsg)
	raw, _ := ioutil.ReadAll(msg.RawReader())
	if !nMsg.unmarshal(raw) {
		return nil, errors.New("parse hello request err")
	}
	return nMsg, nil
}

// Read peer session messages
func (c *peerConn) readHelloRequestMsg() (*helloRequestMsg, error) {
	msg, err := c.readMessage()
	if err != nil {
		return nil, err
	}
	if msg.Type() != typeHelloRequest {
		return nil, err
	}
	nMsg := new(helloRequestMsg)
	raw, _ := ioutil.ReadAll(msg.RawReader())
	if !nMsg.unmarshal(raw) {
		return nil, errors.New("parse hello request err")
	}
	return nMsg, nil
}

// Write peer session messages
func (c *peerConn) writeMessage(mType uint8, data []byte) error {
	cLen := len(data)
	val := make([]byte, cLen+4)
	//logrus.Debugf("Write raw message type=%d, dataLen: %d", mType, len(data))
	binary.LittleEndian.PutUint32(val, uint32(cLen))
	copy(val[4:], data)
	msg := []byte{c.version, mType}
	msg = append(msg, val...)
	_, err := c.rw.Write(msg)
	if err != nil {
		return err
	}
	return nil
}

func (c *peerConn) readMessage() (MessageReader, error) {
	return ReadMessage(c.rw)
}

func (c *peerConn) close() {
	if err := c.rw.Close(); err != nil {
		c.logger.Errorln(err)
	}
}

func (c *peerConn) handshakeCompiled() bool {
	return c.handshakeStatus == 1
}
