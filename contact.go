package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"time"
)

const (
	Good          = iota
	Questionable1 = iota
	Questionable2 = iota
	Bad           = iota
)

// Contact has all information needed to communicate with a remote node.
// Contact implements Stringer interface.
type Contact struct {
	id       Identifier
	ip       net.IP
	port     int
	status   uint8
	lastSeen time.Time
}

// NewContact returns a
func NewContact(id Identifier) *Contact {
	// conn, _ := net.ListenUDP("udp", nil)
	// localAddr := conn.LocalAddr().(*net.UDPAddr)

	return &Contact{
		id:       id,
		lastSeen: time.Now(),
	}
}

func (c *Contact) String() string {
	s := fmt.Sprintf("[id=%s, ip=%s, port=%d, status=%d]",
		c.id.hexString(), c.ip, c.port, c.status)
	return s
}

// Contacts is made up of serveral contacts and a target.
// Contacts implemented sort.Interface.
// TODO: this confused even myself.
type Contacts struct {
	target      Identifier
	contactList []*Contact
}

func (cs *Contacts) print() {
	for _, c := range cs.contactList {
		log.Printf("%s", c)
	}
}

func (cs *Contacts) Len() int {
	return len(cs.contactList)
}

func (cs *Contacts) Less(i, j int) bool {
	di := fmt.Sprintf("%x", distance(cs.contactList[i].id, cs.target))
	dj := fmt.Sprintf("%x", distance(cs.contactList[j].id, cs.target))
	return di < dj
}

func (cs *Contacts) Swap(i, j int) {
	cs.contactList[i], cs.contactList[j] = cs.contactList[j], cs.contactList[i]
}

// func (c *Contacts) Encode() []byte {
// 	b := bytes.NewBuffer(nil)
// 	for _, c := range c.ContactList {

// 	}
// }

//TODO: we should these methods rather than functions
func encodeContacts(nodes []*Contact) []byte {
	b := bytes.NewBuffer(nil)
	for _, n := range nodes {
		encodeContact(b, n)
	}
	return b.Bytes()
}

func encodeContact(b *bytes.Buffer, c *Contact) {
	b.Write(c.id)
	encodeAddr(b, c.ip, c.port)
}

func encodeAddr(b *bytes.Buffer, ip net.IP, port int) {
	b.Write(ip.To4())
	b.WriteByte(byte((port & 0xFF00) >> 8))
	b.WriteByte(byte(port & 0xFF))
}

// decodeContacts decodes byte slice into a list of node contacts.
// TODO: There's quite a bit of stupidity here.
func decodeContacts(data []byte) []*Contact {
	var contacts []*Contact
	for j := 0; j < len(data); j = j + 26 {
		if j+26 > len(data) {
			break
		}
		//TODO: this is stupid
		kn := data[j : j+26]
		c := &Contact{
			id:       Identifier(kn[0:20]),
			ip:       kn[20:24],
			port:     int(kn[24:26][0])<<8 + int(kn[24:26][1]),
			status:   Good,
			lastSeen: time.Now(),
		}
		contacts = append(contacts, c)
	}
	return contacts
}
