package main

import (
	"fmt"
	"log"
	"net"
	"sort"
	"time"
)

const (
	expirationInterval = time.Minute * 25
	maxActiveSearch    = 8
	// maxNodesPerBucket  = 8
)

// startUpdater refreshes DHT node's routing table
// every 10 seconds, it also bootstraps a node
// by search for its own id
func (node *Node) startUpdater() {
	log.Printf("starting updater...")

	if node.table.numOfContacts == 0 {
		node.searchNodes(node.info.id)
	} else {
		// node.refreshTable()
	}

	for {
		// node.table.print()
		// node.table.persist()
		select {
		case <-time.After(60 * time.Second):
			node.refreshTable()
		}
	}
}

func (node *Node) refreshTable() {
	log.Printf("refreshing local routing table")
	node.searchNodes(randID())

	// for _, b := range node.table.buckets {
	// 	//TODO: fix check if expired
	// 	if b.lastUpdated.Add(expirationInterval).Before(time.Now()) || len(b.nodes) == 0 {
	// 		log.Printf("%v", b.lastUpdated.Add(expirationInterval).Before(time.Now()))
	// 		log.Printf("%d", len(b.nodes))
	// 		id := b.randID()
	// 		node.searchNodes(id)
	// 		// node.table.print()
	// 		node.table.persist()
	// 	}
	// }
}

func (node *Node) searchNodes(target Identifier) {
	log.Printf("searching for node %s in network", target.hexString())

	q := &SearchQueue{
		visited: make(map[string]byte),
		results: Contacts{target: target},
	}

	var startNodes []*Contact

	//TODO: we can check number of contacts here
	// if len(node.table.buckets) > 0 {
	// 	startNodes = node.table.findLocalClosest(target)
	// }

	if node.table.numOfContacts == 0 {
		log.Printf("routing table is empty, bootstrapping from well-know nodes")

		for _, host := range Bootstrappers {
			addr, err := net.ResolveUDPAddr("udp", host)
			if err != nil {
				log.Fatalf("error occurred while resolving UDP addr %s\n", err)
			}
			startNodes = append(startNodes, &Contact{randID(), addr.IP, addr.Port, Good, time.Now()})
			// log.Printf("bootstrapped from %s\n", host)
		}
	} else {
		log.Printf("searching for starter nodes from local routing table")
		startNodes = node.table.findLocalClosest(target)
	}
	q.add(startNodes)
	node.search(q)

	//TODO: what the heck
	// bucket, _ := node.Routing.findBucket(sr.target)
	// bucket.Nodes = nil

	for _, n := range q.results.contactList {
		// log.Printf("%s", n)

		//TODO: better way to describe status: sent, received, responded
		if flag, ok := q.visited[n.id.hexString()]; ok && flag&3 == 3 {
			node.table.insertNode(n)
		}
	}
}

func (node *Node) search(q *SearchQueue) {
	reqs := node.sendFindNode(q)

	if len(reqs) > 0 {
		ch := checkResponses(reqs, time.Second*10)
		for i := 0; i < len(reqs); i++ {
			req := <-ch
			if req == nil {
				continue
			}
			if resp, ok := req.resp.ext.(*Response); ok {
				if nodestr, ok := resp.r["nodes"].(string); ok {
					// log.Printf("received response!!!")
					q.visited[req.info.id.hexString()] |= 2

					// log.Printf("see what happend here: %d", q.visited[req.info.id.hexString()])

					// q.visited[req.SN.ID.HexString()] |= 2
					nodes := decodeContacts([]byte(nodestr))
					// log.Printf("%d nodes received", len(nodes))
					q.add(nodes)
				}
			}
		}
	}
	if q.isCloseEnough() {
		log.Printf("results close enough, search is complete")
		return
	}
	// q.results.print()
	log.Printf("results not close enough, keep searching")
	node.search(q)
}

// findNearest is a helper method for Node.Search
func (node *Node) sendFindNode(q *SearchQueue) []*Request {
	// log.Printf("creating search requests")

	var reqs []*Request
	for _, c := range q.results.contactList {
		//TODO: different status for visited nodes
		if flag, ok := q.visited[c.id.hexString()]; ok && 0 == flag {
			//TODO: skip null nodes

			// construct a find_node request
			// log.Printf("constructing find_node request for %s", c)
			addr := &net.UDPAddr{
				IP:   c.ip,
				Port: c.port,
			}
			txid, data, err := node.krpc.encodeFindNode(node.info.id.String(), q.results.target)
			if err != nil {
				log.Fatalf("error occurred while constructing find node requst to %s\n", q.results.target)
			}

			// r := NewRequest(txid, node, v)
			r := NewRequest(c, txid)
			node.reqC <- r
			q.visited[c.id.hexString()] |= 1

			// log.Printf("Sending request to %s", v)

			_, err = node.transport.writeMsgUDP([]byte(data), addr)
			if err != nil {
				log.Print(err)
				continue
			}

			reqs = append(reqs, r)
			if len(reqs) == maxActiveSearch {
				break
			}
		}
	}
	return reqs
}

// SearchQueue is a BFS search queue.
type SearchQueue struct {
	visited map[string]byte
	results Contacts
	d       string
}

func (q *SearchQueue) add(nodes []*Contact) {
	for _, node := range nodes {
		// log.Printf("adding %s to search queue", node)

		if len(node.id.String()) != 20 {
			log.Printf("node %v is not 20 bytes, skip", node.id.String())
			continue
		}
		if node.id.String() == q.results.target.String() {
			log.Printf("node was found!")
			continue
		}
		if _, ok := q.visited[node.id.hexString()]; ok {
			// log.Printf("node %v has been visited, skip", node.id.hexString())
			continue
		}
		q.visited[node.id.hexString()] = 0
		//TODO: man this is stupid
		q.results.contactList = append(q.results.contactList, node)
		sort.Sort(&q.results)
	}
}

func (q *SearchQueue) isCloseEnough() bool {
	// sr.iterNum++
	if q.results.contactList == nil {
		return false
	}
	cl := q.results.contactList[0]
	if cl.id.hexString() == "" {
		return false
	}
	newd := fmt.Sprintf("%x", distance(q.results.target, cl.id))

	b := false
	if q.d != "" {
		b = (newd >= q.d)
		// sr.ownNode.Log.Printf("Is close enough? %t, %s, %s", b, newd, sr.d)
	}
	q.d = newd
	if b {
		j := 0
		for _, c := range q.results.contactList {
			if j == maxNodesPerBucket*2 {
				break
			}
			if flag, ok := q.visited[c.id.hexString()]; ok && flag == 0 {
				log.Printf("not queried nodes")
				// sr.ownNode.Log.Printf("Not queried nodes")
				return false
			}
			j++
		}
		// sr.ownNode.Log.Printf("Finish searching, %d", sr.iterNum)
	}
	return b
}
