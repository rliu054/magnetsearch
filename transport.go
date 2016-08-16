package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

// UDPTransport is a transport for UDP messages.
type UDPTransport struct {
	conn *net.UDPConn
}

// NewTransport returns a new UDP transport.
// This transport is used for all network io.
// TODO: this looks problematic
func NewTransport() *UDPTransport {
	c, err := net.ListenUDP("udp", nil)
	if err != nil {
		log.Fatal(err)
	}

	return &UDPTransport{
		conn: c,
	}
}

func (t *UDPTransport) writeMsgUDP(m []byte, addr *net.UDPAddr) (int, error) {
	n, err := t.conn.WriteToUDP(m, addr)
	if err != nil || n == 0 {
		log.Printf("Error occurred while sending UDP message, %d bytes have been sent", n)
	}
	return n, err
}

// Request represents a UDP message.
type Request struct {
	info  *Contact
	txid  string
	resp  *KRPCMessage
	respC chan *Request
}

// NewRequest returns a new Request based on contact info and identifier.
func NewRequest(c *Contact, id uint32) *Request {
	return &Request{
		info:  c,
		txid:  fmt.Sprintf("%d", id),
		resp:  new(KRPCMessage),
		respC: make(chan *Request, 1),
	}
}

// CheckResponses takes several requests and checks their response channel
// for possible responses, t is the timeout interval
func checkResponses(reqs []*Request, t time.Duration) chan *Request {
	// log.Printf("checking for responses to previous requests")
	ch := make(chan *Request, len(reqs))

	for _, r := range reqs {
		go func(r *Request) {
			// log.Printf("Wait response #%s", r.txid)
			select {
			case resp := <-r.respC:
				ch <- resp
				// log.Printf("response #%s received", r.txid)
				return
			case <-time.After(t):
				// log.Printf("Wait timeout #%s", r.txid)
				ch <- nil
				return
			}
		}(r)
	}
	return ch
}
