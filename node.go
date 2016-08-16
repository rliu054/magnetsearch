package main

import (
	"bytes"
	"crypto/sha1"
	"io"
	"log"
	"time"
	// log "github.com/Sirupsen/logrus"
)

const (
	// UDPPacketSize is the default UDP read buffer size
	UDPPacketSize = 1024
	// maxActiveNodes is the max number of active nodes at a given time.
	// Need to bump to higher value when codebase is stable.
	maxActiveNodes = 1
)

// Bootstrappers are well known torrent nodes,
// a node need to contact at least one nodes to join the
// torrent network.
var Bootstrappers = []string{
	"router.bittorrent.com:6881",
	"dht.transmissionbt.com:6881",
	"service.ygrek.org.ua:6881",
	"router.utorrent.com:6881",
	"router.transmission.com:6881",
}

// A Node represents a DHT node in the torrent network.
type Node struct {
	// Info specifies contact information of current node.
	info *Contact

	// Table represents local routing table of current node.
	table *RoutingTable

	// msgC is a channel for sending and receiving krpc messages.
	krpc *KRPC
	msgC chan *KRPCMessage

	// transport is a UDP transport which is used for communication in the DHT network.
	// reqMap is a hashtable which records requests history, so that all
	// requests for one transaction can be correlated.
	transport *UDPTransport
	reqC      chan *Request
	reqMap    map[string]*Request
	tokenMap  map[string]*Contact

	// secret is a random token that changes every 5 min
	secret string

	masterlogger chan string
}

// NewNode returns a new DHT node.
func NewNode(id Identifier, log chan string) *Node {
	return &Node{
		info:         NewContact(id),
		table:        NewRoutingTable(id),
		krpc:         new(KRPC),
		transport:    NewTransport(),
		reqC:         make(chan *Request),
		msgC:         make(chan *KRPCMessage),
		reqMap:       make(map[string]*Request),
		tokenMap:     make(map[string]*Contact),
		masterlogger: log,
	}
	// n.Log = log.New(logger, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	// n.MLog = log.New(mlogger, id.HexString()+" ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	// n.NewMsg = make(chan *KRPCMessage)
	// n.Routing = NewRouting(n)
	// n.krpc = NewKRPC(n)
	// n.nw = NewNetwork(n)
	// // n.master = master
	// n.tokens = make(map[string]*TokenVal)
	// n.Info.LastSeen = time.Now()
	// return n
}

// Start brings a node up and initiates all listeners.
func (node *Node) start() {
	log.Printf("starting node %s", node.info)
	go func() { node.startUDPListener() }()
	go func() { node.startMsgBroker() }()
	go func() { node.startUpdater() }()

	for {
		select {
		// case msg := <-node.masterlogger:
		// 	fmt.Println(msg)
		case <-time.After(time.Minute * 5):
			// node.secret = randID().hexString()
			// getDBSession().deleteOldPeers()
		}
	}
}

// startUDPListener starts a listener for incoming UDP messages,
// after a message is decoded it is sent over to message broker
func (node *Node) startUDPListener() {
	log.Printf("starting UDP listener...")
	buffer := make([]byte, UDPPacketSize)
	for {
		node.transport.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		n, addr, err := node.transport.conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("error occurred while reading from UDP: [%v]", err)
			continue
		}
		log.Printf("received %d bytes of data from network: %s", n, addr)
		message, err := node.krpc.decode(string(buffer), addr)
		if err != nil || message == nil {
			log.Printf("error occurred while decoding data")
		} else {
			// throw to the message broker
			node.msgC <- message
		}
	}

}

// startMsgBroker starts a message broker that listens for two things:
// 1) in case of an outgoing request, save {request_id: request} in a map.
// 2) in case of a krpc message, it checks if it's a response related to
// an existing request, otherwise it invokes a routine to process the query
func (node *Node) startMsgBroker() {
	log.Printf("starting message broker...")
	for {
		select {
		case req := <-node.reqC:
			// log.Printf("msg broker receiving from reqC channel")
			node.reqMap[req.txid] = req
		case msg := <-node.msgC:
			// log.Printf("msg broker receiving from msgC channel")

			//TODO: check if we have req with this transaction id
			if req, ok := node.reqMap[msg.t]; ok {
				// log.Printf("we already have this req with this transaction id")

				req.resp = msg
				req.respC <- req
				delete(node.reqMap, msg.t)
			} else {
				if msg.y == "q" {
					log.Printf("it's a KRPC message")

					// go func() {
					node.processQuery(msg)
					// }()
				}
			}

		case <-time.After(5 * time.Second):
			//TODO: need refresh
		}
	}
}

// processQuery handles KRPCMessages
func (node *Node) processQuery(m *KRPCMessage) {
	if query, ok := m.ext.(*Query); ok {
		queryNode := &Contact{
			id:       Identifier(query.a["id"].(string)),
			ip:       m.addr.IP,
			port:     m.addr.Port,
			status:   Good,
			lastSeen: time.Now(),
		}

		switch query.q {
		case "ping":
			log.Printf("<========= received ping from %s", queryNode)
			resp, err := node.krpc.encodePong(node.info.id.String(), m.t)
			if err != nil {
				log.Printf("Error while encoding pong")
			}

			log.Printf("=========> sent out ping resp: %v to addr %v", resp, m.addr)
			node.transport.writeMsgUDP([]byte(resp), m.addr)
		case "find_node":
			log.Printf("<========= received find_node from %s", queryNode)

			if target, ok := query.a["target"].(string); ok {
				// search for target in local routing table

				// log.Printf("find_node, target is %s", target)
				closest := node.table.findLocalClosest(Identifier(target))

				// log.Printf("find_node, closest %d nodes", closest)

				nodes := encodeContacts(closest)
				resp, err := node.krpc.encodeNodeSearch(m.t, node.info.id.String(), "", nodes)
				if err != nil {
					log.Printf("Error while encoding search response")
				}

				log.Printf("=========> sent out find_node resp: %v to addr %v", resp, m.addr)
				node.transport.writeMsgUDP([]byte(resp), m.addr)
			}
		case "get_peers":
			log.Printf("<========= received get_peers from %s", queryNode)

			if infohash, ok := query.a["info_hash"].(string); ok {
				// look for infohash from datastore
				ih := Identifier(infohash)
				getDBSession().addResource(ih.hexString())
				token := node.getToken(queryNode)
				peers, _ := getDBSession().loadPeers(ih.hexString())

				if len(peers) > 0 {
					data, _ := node.krpc.encodePeerSearch(m.t, node.info.id.String(), token, peers)
					node.transport.writeMsgUDP([]byte(data), m.addr)
				} else {

					// problem here
					closest := node.table.findLocalClosest(ih)
					log.Printf("******%d closes nodes returned", len(closest))

					nodes := encodeContacts(closest)

					log.Printf("encoded nodes: %v", nodes)
					data, _ := node.krpc.encodeNodeSearch(m.t, node.info.id.String(), token, nodes)

					log.Printf("=========> sent out get_peers resp: %v to get_peers %v", data, m.addr)
					node.transport.writeMsgUDP([]byte(data), m.addr)
				}
			}
		case "announce_peer":
			// if you don't receive announce_peer, it means your get_peers is not properly handled
			log.Fatalf("<========= received announce_peer from %s", queryNode)

			var infohash string
			var token string
			var impliedPort int64
			var port int64
			var ok bool
			// var req *Request

			if infohash, ok = query.a["info_hash"].(string); !ok {
				break
			}
			if token, ok = query.a["token"].(string); !ok {
				break
			}
			if port, ok = query.a["port"].(int64); !ok {
				break
			}
			if impliedPort, ok = query.a["implied_port"].(int64); !ok {
				impliedPort = 0
			}
			if impliedPort > 0 {
				port = int64(m.addr.Port)
			}
			ih := Identifier(infohash)
			c, ok := node.tokenMap[token]

			if ok && c.ip.Equal(queryNode.ip) {
				buf := bytes.NewBufferString("")
				encodeAddr(buf, queryNode.ip, int(port))
				getDBSession().addPeer(ih.hexString(), buf.Bytes())
			}
			if ok {
				delete(node.reqMap, token)
			}
			data, _ := node.krpc.encodePong(node.info.id.String(), m.t)
			node.transport.writeMsgUDP([]byte(data), m.addr)
			break
		}
		node.table.insertNode(queryNode)
	}
}

// func (node *Node) getSimpleToken(c *Contact) string {
// 	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
// 	b := make([]byte, 10)
// 	for i := range b {
// 		b[i] = letterBytes[rand.Intn(len(letterBytes))]
// 	}

// 	node.tokenMap[string(b)] = c
// 	return string(b)
// }

func (node *Node) getToken(c *Contact) string {
	hash := sha1.New()
	io.WriteString(hash, c.ip.String())
	io.WriteString(hash, time.Now().String())
	io.WriteString(hash, node.secret)
	token := bytes.NewBuffer(hash.Sum(nil)).String()

	//TODO: is this a problem?
	node.tokenMap[token] = c
	return token
}
