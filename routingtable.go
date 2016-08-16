package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"math/big"
	"math/rand"
	"time"

	"github.com/rliu054/magnetsearch/util"
)

// TODO: put all constants to a config/JSON file
// doesn't make sense to scatter them around.
const (
	maxNodesPerBucket     int = 8
	maxNumOfBuckets       int = 160
	maxNumOfSearchResults int = 8
)

// Bucket consists of at most K nodes
type Bucket struct {
	min, max    *big.Int
	nodes       []*Contact
	lastUpdated time.Time
}

// NewBucket initializes and returns pointer to a new bucket
func NewBucket(min, max *big.Int) *Bucket {
	return &Bucket{
		min:         min,
		max:         max,
		lastUpdated: time.Now(),
	}
}

func (b *Bucket) contains(node *Contact) bool {
	for i, n := range b.nodes {
		if n.id.String() == node.id.String() {
			b.nodes[i] = node
			b.lastUpdated = time.Now()
			return true
		}
	}
	return false
}

func (b *Bucket) insert(node *Contact) {
	b.nodes = append(b.nodes, node)
	b.lastUpdated = time.Now()
}

// randid generates a random nodeid in range [min, max]
func (b *Bucket) randID() []byte {
	// var d *big.Int
	d := big.NewInt(0)
	d.Sub(b.max, b.min)

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	z := util.Biadd(b.min, d.Rand(random, d))
	ret := make([]byte, 20)
	for idx, b := range z.Bytes() {
		ret[idx] = b
	}
	return ret
}

// RoutingTable maintains active neighbours in DHT network.
type RoutingTable struct {
	id            Identifier
	buckets       []*Bucket
	numOfContacts int // number of contacts in routing table
}

// NewRoutingTable returns a new routing table with given id.
// REFACTOR: we can do better.
func NewRoutingTable(id Identifier) *RoutingTable {
	bucket := NewBucket(big.NewInt(0), big.NewInt(0).Lsh(util.Binew(1), 160))
	buckets := make([]*Bucket, 1)
	buckets[0] = bucket

	table := &RoutingTable{
		id:            id,
		buckets:       buckets,
		numOfContacts: 0,
	}

	// if we have this persisted
	// data, err := getDBSession().loadNodeInfo(id)
	// if err == nil && len(data) > 0 {
	// 	err = table.loadRouting(bytes.NewBuffer(data))
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }
	return table
}

func (table *RoutingTable) print() {
	log.Printf("#routing table [id = %s] has %d buckets, %d nodes",
		table.id.hexString(), len(table.buckets), table.numOfContacts)

	for i, b := range table.buckets {
		log.Printf("##bucket%d, min=%v, max=%v, lastUpdated %v", i, b.min, b.max, b.lastUpdated)
		for _, n := range b.nodes {
			log.Printf("##node %s", n)
		}
	}
}

// LoadRouting reads data from reader, attempts to decode data into node contacts and
// inserts them into current routing table.
// TODO: get rid of magic number, return bytes read and error, return is very ugly.
func (table *RoutingTable) loadRouting(reader io.Reader) error {
	buf := bufio.NewReader(reader)
	data := make([]byte, 24)
	_, err := buf.Read(data)
	if err != nil {
		return err
	}

	var length uint32
	err = binary.Read(bytes.NewBuffer(data[20:24]), binary.LittleEndian, &length)
	if err != nil {
		return err
	}

	stream := make([]byte, length)
	_, err = buf.Read(stream)
	if err != nil {
		return err
	}
	nodes := decodeContacts(stream)
	log.Printf("%d nodes loaded from database", len(nodes))
	for _, node := range nodes {
		table.insertNode(node)
	}

	// TODO: get rid of these shitty returns
	return nil
}

// persist encodes current nodes in routing table and saves to underlying datastore.
func (table *RoutingTable) persist() {
	log.Printf("saving routing table information to database")
	table.print()

	data := bytes.NewBuffer(nil)
	for _, b := range table.buckets {
		for _, n := range b.nodes {
			encodeContact(data, n)
		}
	}

	buf := bytes.NewBuffer(nil)
	buf.Write(table.id)
	binary.Write(buf, binary.LittleEndian, uint32(data.Len()))
	buf.Write(data.Bytes())
	getDBSession().updateNodeInfo(table.id, buf.Bytes())
}

// insertNode first looks for a bucket index and tries to insert
// if bucket is already full, node is either dropped or bucket is
// split into two buckets each with half of the node ID space
func (table *RoutingTable) insertNode(node *Contact) {
	log.Printf("inserting %s to routing table %s", node, table.id.hexString())
	b, idx := table.findBucket(node.id)
	if idx < maxNumOfBuckets {
		if b.contains(node) {
			b.lastUpdated = time.Now()
		} else if len(b.nodes) < maxNodesPerBucket {
			b.insert(node)
			table.numOfContacts++
		} else if idx == len(table.buckets)-1 {
			table.splitBucket(b)
			table.insertNode(node)
		}
	}
}

func (table *RoutingTable) splitBucket(b *Bucket) {
	mid := util.Mid(b.min, b.max)
	var newBucket *Bucket

	if table.id.toInt().Cmp(mid) >= 0 {
		newBucket = NewBucket(mid, b.max)
		b.max = mid
	} else {
		newBucket = NewBucket(b.min, mid)
		b.min = mid
	}
	table.buckets = append(table.buckets, newBucket)

	var contactlist []*Contact
	for _, n := range b.nodes {
		if n.id.toInt().Cmp(newBucket.min) >= 0 && n.id.toInt().Cmp(newBucket.max) < 0 {
			newBucket.nodes = append(newBucket.nodes, n)
		} else {
			contactlist = append(contactlist, n)
		}
	}
	b.nodes = contactlist
}

func (table *RoutingTable) findBucket(id Identifier) (*Bucket, int) {
	idx := table.bucketIdx(id)
	length := len(table.buckets)
	if idx < length {
		return table.buckets[idx], idx
	}
	return table.buckets[length-1], length - 1
}

// // bucketIdx returns [0, 160) if candidate node if different
// // than current node and returns 160 if otherwise
func (table *RoutingTable) bucketIdx(id Identifier) int {
	i := 0
	for ; i < len(table.id); i++ {
		if table.id[i] != id[i] {
			break
		}
	}
	if i == len(table.id) {
		return maxNumOfBuckets - 1
	}

	dist := table.id[i] ^ id[i]
	j := 0
	for ; dist != 0; dist >>= 1 {
		j++
	}
	return 8*i + (8 - j)
}

// FindClosest returns SearchResultNum
func (table *RoutingTable) findLocalClosest(id Identifier) []*Contact {
	log.Printf("searching local routing table for node %s", id.hexString())
	// log.Printf("routing table id %s", table.id.hexString())
	// table.print()

	var result []*Contact
	_, p := table.findBucket(id)
	// log.Printf("search index is %d", p)

	table.findLocalNode(p, 0, &result)
	return result
}

// searchNode is a helper method to FindClosest
func (table *RoutingTable) findLocalNode(p, offset int, result *[]*Contact) {
	if offset >= len(table.buckets) || len(*result) >= maxNumOfSearchResults {
		return
	}

	// log.Printf("currently %d search results, search continues", len(*result))
	if p-offset >= 0 {
		for _, n := range table.buckets[p-offset].nodes {
			if n.status == Good && len(*result) < maxNumOfSearchResults {
				*result = append(*result, n)
			}
		}
	}
	if p+offset < len(table.buckets) && offset > 0 {
		for _, n := range table.buckets[p+offset].nodes {
			if n.status == Good && len(*result) < maxNumOfSearchResults {
				*result = append(*result, n)
			}
		}
	}
	table.findLocalNode(p, offset+1, result)
}
