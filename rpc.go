package main

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"net"
	"sync/atomic"

	"github.com/zeebo/bencode"
)

type KRPC struct {
	txid uint32 // transaction id
}

type KRPCMessage struct {
	t    string      // transaction ID
	y    string      // message type
	ext  interface{} // extra keys
	addr *net.UDPAddr
}

type Query struct {
	q string                 // method name of query
	a map[string]interface{} // additional arguments
}

type Response struct {
	r map[string]interface{} // named return values
}

type Error struct {
	e []interface{}
	// e *list.List
}

// NewTxID returns a new transaction id.
//TODO: other ways of atomic operations
func (krpc *KRPC) NewTxID() uint32 {
	next := atomic.AddUint32(&krpc.txid, 1)
	return next % math.MaxUint16
}

func (krpc *KRPC) decode(s string, addr *net.UDPAddr) (*KRPCMessage, error) {
	v := make(map[string]interface{})
	if err := bencode.DecodeString(s, &v); err != nil {
		return nil, err
	}

	// log.Printf("decoded message: %v", v)

	// FIXME: this is stupid
	ok := false
	message := new(KRPCMessage)
	message.t, ok = v["t"].(string)
	if !ok {
		log.Printf("error occured while decoding KRPC message")
		return nil, nil
	}

	message.y, ok = v["y"].(string)
	if !ok {
		log.Printf("error occured while decoding KRPC message")
		return nil, nil
	}

	message.addr = addr
	switch message.y {
	case "q":
		query := new(Query)
		query.q = v["q"].(string)
		query.a = v["a"].(map[string]interface{})
		message.ext = query
	case "r":
		resp := new(Response)
		resp.r = v["r"].(map[string]interface{})
		message.ext = resp
	case "e":
		err := new(Error)
		err.e = v["e"].([]interface{})
		message.ext = err
	default:
		log.Printf("invalid response")
		break
	}
	return message, nil
}

func (krpc *KRPC) encodePing(nodeID string) (uint32, string, error) {
	txid := krpc.NewTxID()
	p := make(map[string]interface{})
	p["t"] = fmt.Sprintf("%d", txid)
	p["y"] = "q"
	p["q"] = "ping"
	arg := make(map[string]string)
	arg["id"] = nodeID
	p["a"] = arg
	resp, err := bencode.EncodeString(p)
	return txid, resp, err
}

// EncodePong encodes a pong message into byte stream.
func (krpc *KRPC) encodePong(nodeID string, txID string) (string, error) {
	p := make(map[string]interface{})
	p["t"] = txID
	p["y"] = "r"
	arg := make(map[string]string)
	arg["id"] = nodeID
	p["r"] = arg

	resp, err := bencode.EncodeString(p)
	return resp, err
}

func (krpc *KRPC) encodeGetPeers(nodeID string, infohash Identifier) (uint32, string, error) {
	txid := krpc.NewTxID()
	p := make(map[string]interface{})
	p["t"] = fmt.Sprintf("%d", txid)
	p["y"] = "q"
	p["q"] = "get_peers"
	arg := make(map[string]string)
	arg["id"] = nodeID
	arg["info_hash"] = infohash.String()
	p["a"] = arg
	resp, err := bencode.EncodeString(p)
	return txid, resp, err
}

func (krpc *KRPC) encodeAnnouncePeer(nodeID string, infohash Identifier, port int, token string) (uint32, string, error) {
	txid := krpc.NewTxID()
	p := make(map[string]interface{})
	p["t"] = fmt.Sprintf("%d", txid)
	p["y"] = "q"
	p["q"] = "announce_peer"
	arg := make(map[string]interface{})
	arg["id"] = nodeID
	arg["token"] = token
	arg["port"] = port
	arg["implied_port"] = 0
	arg["info_hash"] = infohash.String()
	p["a"] = arg
	resp, err := bencode.EncodeString(p)
	return txid, resp, err
}

func (encode *KRPC) encodeNodeResult(nodeID string, txid string, token string, nodes []byte) (string, error) {
	p := make(map[string]interface{})
	p["t"] = txid
	p["y"] = "r"
	arg := make(map[string]string)
	arg["id"] = nodeID
	if token != "" {
		arg["token"] = token
	}
	arg["nodes"] = bytes.NewBuffer(nodes).String()
	p["r"] = arg
	resp, err := bencode.EncodeString(p)
	return resp, err
}

// EncodeNodeSearch encodes contacts into byte stream.
func (krpc *KRPC) encodeNodeSearch(txID string, nodeID string, token string, nodes []byte) (string, error) {
	p := make(map[string]interface{})
	p["t"] = txID
	p["y"] = "r"
	arg := make(map[string]string)
	arg["id"] = nodeID
	if token != "" {
		arg["token"] = token
	}
	arg["nodes"] = bytes.NewBuffer(nodes).String()
	// arg["nodes"] = ""

	p["r"] = arg

	resp, err := bencode.EncodeString(p)
	return resp, err
}

func (krpc *KRPC) encodePeerSearch(txid string, nodeID string, token string, peers []string) (string, error) {
	p := make(map[string]interface{})
	p["t"] = txid
	p["y"] = "r"
	arg := make(map[string]interface{})
	arg["id"] = nodeID
	arg["token"] = token
	arg["values"] = peers
	p["r"] = arg

	resp, err := bencode.EncodeString(p)
	return resp, err
}

func (encode *KRPC) encodePeerResult(nodeID string, txid string, token string, peers []string) (string, error) {
	p := make(map[string]interface{})
	p["t"] = txid
	p["y"] = "r"
	arg := make(map[string]interface{})
	arg["id"] = nodeID
	arg["token"] = token
	arg["values"] = peers
	p["r"] = arg
	resp, err := bencode.EncodeString(p)
	return resp, err
}

func (krpc *KRPC) encodeFindNode(nodeID string, target Identifier) (uint32, string, error) {
	txid := krpc.NewTxID()
	v := make(map[string]interface{})
	v["t"] = fmt.Sprintf("%d", txid)
	v["y"] = "q"
	v["q"] = "find_node"
	args := make(map[string]string)
	args["id"] = nodeID
	args["target"] = target.String()
	v["a"] = args
	s, err := bencode.EncodeString(v)
	return txid, s, err
}
