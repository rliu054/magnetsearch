package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"math/big"
	"math/rand"
	"strconv"
	"time"
)

// Identifier represents a unique 160 bit string.
// It can be used as unique identifier for each node.
type Identifier []byte

func (id Identifier) String() string {
	return bytes.NewBuffer(id).String()
}

// toInt returns an id in math.big format.
func (id Identifier) toInt() *big.Int {
	return big.NewInt(0).SetBytes(id)
}

// hexString returns the id in hex format.
func (id Identifier) hexString() string {
	return hex.EncodeToString(id)
}

// distance calculates XOR distance between two identifiers.
// TODO: should we make it a method of Identifier type.
func distance(x, y Identifier) []byte {
	dist := make([]byte, 20)
	for i := 0; i < len(x); i++ {
		dist[i] = x[i] ^ y[i]
	}
	return dist
}

// randID generates a random identifier.
func randID() Identifier {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	h := sha1.New()
	io.WriteString(h, time.Now().String()) // each call returns a different identifier
	io.WriteString(h, string(r.Int()))
	return h.Sum(nil)
}

// hexToID turns a 40 character string to a 20 byte identifier.
func hexToID(s string) Identifier {
	if len(s) != 40 {
		return nil
	}
	id := make([]byte, 20)
	j := 0
	for i := 0; i < len(s); i += 2 {
		n1, _ := strconv.ParseInt(s[i:i+1], 16, 8)
		n2, _ := strconv.ParseInt(s[i+1:i+2], 16, 8)
		id[j] = byte((n1 << 4) + n2)
		j++
	}
	return id
}
