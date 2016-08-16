package main

import (
	"log"
	"math/big"
	"testing"

	"github.com/rliu054/magnetsearch/util"
)

var testids = []string{
	"0000000000000000000000000000000000000001",
	"0000000000000000000000000000000000000010",
	"0000000000000000000000000000000000000100",
	"0000000000000000000000000000000000001000",
	"0000000000000000000000000000000000010000",
	"0000000000000000000000000000000000100000",
	"0000000000000000000000000000000001000000",
	"0000000000000000000000000000000010000000",

	"0000000000000000000000000000000011111111",

	"0000000000000000000000000000000100000000",
	"0000000000000000000000000000001000000000",
	"0000000000000000000000000000010000000000",
	"0000000000000000000000000000100000000000",
	"0000000000000000000000000001000000000000",
	"0000000000000000000000000010000000000000",
	"0000000000000000000000000100000000000000",
	"0000000000000000000000001000000000000000",
}

func TestSearchNode(t *testing.T) {
	id := randID()
	table := NewRoutingTable(id)
	c := NewContact(id)
	table.insertNode(c)

	results := table.findLocalClosest(id)
	if results[0].id.hexString() != id.hexString() {
		t.Errorf("expected search result: %s, got:  %s",
			id.hexString(), results[0].id.hexString())
	}
}

func TestSearchNodeMulti(t *testing.T) {
	id := hexToID("0000000000000000000000000000000011111111")
	table := NewRoutingTable(id)

	for _, v := range testids {
		c := NewContact(hexToID(v))
		table.insertNode(c)
	}

	table.print()

	results := table.findLocalClosest(id)

	for i := range results {
		log.Printf("%s", results[i])
	}

	if len(results) < maxNumOfSearchResults {
		t.Errorf("expected %d nodes returned, got: %d nodes",
			maxNumOfSearchResults, len(results))
	}
}

func TestSearchNodeInBucket(t *testing.T) {
	bucket := NewBucket(big.NewInt(0), big.NewInt(0).Lsh(util.Binew(1), 160))
	c := NewContact(randID())

	if bucket.contains(c) == true {
		t.Errorf("expected if bucket contains node %v, got:  %v",
			false, bucket.contains(c))
	}

	bucket.insert(c)
	if bucket.contains(c) == false {
		t.Errorf("expected if bucket contains node %v, got:  %v",
			true, bucket.contains(c))
	}
}

func TestSplitBucket(t *testing.T) {
	table := NewRoutingTable(randID())
	for i := 0; i < 2*maxNodesPerBucket; i++ {
		c := NewContact(randID())
		table.insertNode(c)
	}

	if table.numOfContacts != 2*maxNodesPerBucket {
		t.Errorf("expected total %d nodes, got: %d nodes",
			2*maxNodesPerBucket, table.numOfContacts)
	}
}
