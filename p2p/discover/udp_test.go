package discover

import (
	"bytes"
	"net"
	"testing"
	"xfsgo/common/ahash"
	"xfsgo/common/rawencode"
	"xfsgo/crypto"
)

func Test_udp_encodePacketN(t *testing.T) {
	dataType := uint8(1)
	data := "hello"
	dataraw, err := rawencode.Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	key := crypto.MustGenPrvKey()
	datahash := ahash.SHA256(dataraw)
	wantSign, err := crypto.ECDSASign(datahash, key)
	if err != nil {
		t.Fatal(err)
	}
	remoteKey := crypto.MustGenPrvKey()
	wantRemote := PubKey2NodeId(remoteKey.PublicKey)
	encodedraw, err := encodePacketN(key, dataType, data, wantRemote)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("raw: %x", encodedraw)
	if encodedraw[0] != dataType {
		t.Fatal("invalid packet type")
	}
	if !bytes.Equal(wantSign, encodedraw[1:66]) {
		t.Fatal("invalid packet sign")
	}
	if !bytes.Equal(wantRemote[:], encodedraw[66:130]) {
		t.Fatal("invalid packet remote")
	}
	if len(dataraw) != int(encodedraw[130]) {
		t.Fatal("invalid packet data length")
	}
	if !bytes.Equal(dataraw, encodedraw[131:]) {
		t.Fatal("invalid packet data")
	}
}
func Test_udp_decodePacketHeadN(t *testing.T) {
	dataType := uint8(1)
	data := "hello"
	dataraw, err := rawencode.Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	key := crypto.MustGenPrvKey()
	datahash := ahash.SHA256(dataraw)
	wantSign, err := crypto.ECDSASign(datahash, key)
	if err != nil {
		t.Fatal(err)
	}
	remoteKey := crypto.MustGenPrvKey()
	wantRemote := PubKey2NodeId(remoteKey.PublicKey)
	encodedraw, err := encodePacketN(key, dataType, data, wantRemote)
	buffer := bytes.NewBuffer(encodedraw)
	ph, _, err := decodePacketHeadN(buffer)
	if err != nil {
		t.Fatal(err)
	}
	if dataType != ph.mType {
		t.Fatal("invalid packet type")
	}
	if !bytes.Equal(wantSign, ph.sign[:]) {
		t.Fatal("invalid packet sign")
	}
	if len(dataraw) != int(ph.dataLen) {
		t.Fatal("invalid packet data length")
	}
}
func Test_udp_decodePacketN(t *testing.T) {
	key := crypto.MustGenPrvKey()
	wantNodeId := PubKey2NodeId(key.PublicKey)
	fromAddr := &net.UDPAddr{
		IP:   net.IPv4(0xff, 0xff, 0xff, 0xff),
		Port: 0xffff,
	}
	toAddr := &net.UDPAddr{
		IP:   net.IPv4(0xff, 0xff, 0xff, 0xff),
		Port: 0xffff,
	}
	fromEndpoint := makeEndpoint(fromAddr, uint16(fromAddr.Port))
	toEndpoint := makeEndpoint(toAddr, uint16(toAddr.Port))
	pack := ping{
		Version:    1,
		From:       fromEndpoint,
		To:         toEndpoint,
		Expiration: 100,
	}
	remoteKey := crypto.MustGenPrvKey()
	wantRemote := PubKey2NodeId(remoteKey.PublicKey)
	encodedraw, err := encodePacketN(key, pingPacket, pack, wantRemote)
	buffer := bytes.NewBuffer(encodedraw)
	p, n, err := decodePacketN(buffer, wantRemote)
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("invalid parse packet")
	}
	if !bytes.Equal(wantNodeId[:], n[:]) {
		t.Fatal("invalid packet node id")
	}
}

func Test_udp_encodePacket(t *testing.T) {
	dataType := uint8(1)
	data := "hello"
	dataraw, err := rawencode.Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	key := crypto.MustGenPrvKey()
	datahash := ahash.SHA256(dataraw)
	wantSign, err := crypto.ECDSASign(datahash, key)
	if err != nil {
		t.Fatal(err)
	}
	encodedraw, err := encodePacket(key, dataType, data)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("raw: %x", encodedraw)
	if encodedraw[0] != dataType {
		t.Fatal("invalid packet type")
	}
	if !bytes.Equal(wantSign, encodedraw[1:66]) {
		t.Fatal("invalid packet sign")
	}
	if len(dataraw) != int(encodedraw[66]) {
		t.Fatal("invalid packet data length")
	}
	if !bytes.Equal(dataraw, encodedraw[67:]) {
		t.Fatal("invalid packet data")
	}
}

func Test_udp_decodePacketHead(t *testing.T) {
	dataType := uint8(1)
	data := "hello"
	dataraw, err := rawencode.Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	key := crypto.MustGenPrvKey()
	datahash := ahash.SHA256(dataraw)
	wantSign, err := crypto.ECDSASign(datahash, key)
	if err != nil {
		t.Fatal(err)
	}
	encodedraw, err := encodePacket(key, dataType, data)
	buffer := bytes.NewBuffer(encodedraw)
	ph, _, err := decodePacketHead(buffer)
	if err != nil {
		t.Fatal(err)
	}
	if dataType != ph.mType {
		t.Fatal("invalid packet type")
	}
	if !bytes.Equal(wantSign, ph.sign[:]) {
		t.Fatal("invalid packet sign")
	}
	if len(dataraw) != int(ph.dataLen) {
		t.Fatal("invalid packet data length")
	}
}

func Test_udp_decodePacket(t *testing.T) {
	key := crypto.MustGenPrvKey()
	wantNodeId := PubKey2NodeId(key.PublicKey)
	fromAddr := &net.UDPAddr{
		IP:   net.IPv4(0xff, 0xff, 0xff, 0xff),
		Port: 0xffff,
	}
	toAddr := &net.UDPAddr{
		IP:   net.IPv4(0xff, 0xff, 0xff, 0xff),
		Port: 0xffff,
	}
	fromEndpoint := makeEndpoint(fromAddr, uint16(fromAddr.Port))
	toEndpoint := makeEndpoint(toAddr, uint16(toAddr.Port))
	pack := ping{
		Version:    1,
		From:       fromEndpoint,
		To:         toEndpoint,
		Expiration: 100,
	}
	encodedraw, err := encodePacket(key, pingPacket, pack)
	buffer := bytes.NewBuffer(encodedraw)
	p, n, err := decodePacket(buffer)
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("invalid parse packet")
	}
	if !bytes.Equal(wantNodeId[:], n[:]) {
		t.Fatal("invalid packet node id")
	}
}
