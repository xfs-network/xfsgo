package backend

import (
	"encoding/json"
	"math/rand"
	"net"
	"testing"
	"xfsgo/common"
	"xfsgo/p2p/discover"
	"xfsgo/test"
)

var (
	testStatusData = []*statusData{
		{
			Version: 1,
			Network: 1,
			Head:    common.Hash{},
			Height:  0,
			Genesis: common.Hash{},
		},
		randomTestStateData(),
		randomTestStateData(),
		randomTestStateData(),
		randomTestStateData(),
		randomTestStateData(),
		randomTestStateData(),
		randomTestStateData(),
	}
)

func randomTestStateData() *statusData {
	randHead := common.Hash{}
	rand.Read(randHead[:])
	randGenesis := common.Hash{}
	rand.Read(randGenesis[:])
	return &statusData{
		Version: rand.Uint32(),
		Network: rand.Uint32(),
		Head:    randHead,
		Height:  rand.Uint64(),
		Genesis: randGenesis,
	}
}
func TestPeer_Handshake(t *testing.T) {
	selfNodeId := genRandomTestNode()
	remoteNodeId := genRandomTestNode()
	remoteNode := discover.NewNode(net.IPv4(0xff, 0xff, 0xff, 0xff), uint16(1611), uint16(1611), remoteNodeId.nodeId)
	wantStatusData := testStatusData[0]
	handshakeFn := func(mType uint8, data []byte) (uint8, []byte, error) {
		var (
			err    error
			repbuf []byte
		)
		if repbuf, err = json.Marshal(wantStatusData); err != nil {
			return 0, repbuf, err
		}
		return MsgCodeVersion, repbuf, err
	}
	np := test.NewBufferPeer(selfNodeId.nodeId, remoteNode, handshakeFn)
	for _, sd := range testStatusData {
		p := newPeer(np, sd.Version, sd.Network)
		err := p.Handshake(sd.Head, sd.Height, sd.Genesis)
		if sd.Genesis != wantStatusData.Genesis && err == nil {
			t.Fatal("test err")
		} else if sd.Version != wantStatusData.Version && err == nil {
			t.Fatal("test err")
		} else if sd.Network != wantStatusData.Network && err == nil {
			t.Fatal("test err")
		}
		if wantStatusData.Height != p.height {
			t.Fatal("test err")
		} else if wantStatusData.Head != p.head {
			t.Fatal("test err")
		}
	}
}

//func Test_a(t *testing.T) {
//	t.Fatal("abc")
//}
