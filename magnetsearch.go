package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	// session := getDBSession()
	// nodeids, err := session.loadAllNodeIDs()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	var nodeids []string

	master := make(chan string)
	// logger := os.Stdout
	// mlogger, err := os.OpenFile("msg", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0744)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	if len(nodeids) > 0 {
		//TODO: there's a bug here
		log.Printf("reloading peers from database")
		for _, nodeid := range nodeids {
			go func(id string) {
				n := NewNode(hexToID(id), master)
				n.start()
			}(nodeid)
		}
	}

	for i := 0; i < maxActiveNodes-len(nodeids); i++ {
		go func() {
			node := NewNode(randID(), master)
			node.start()
		}()
	}

	// for {
	// 	select {
	// 	case msg := <-master:
	// 		fmt.Println(msg)
	// 	}
	// }
	port := os.Getenv("PORT")
	if port != "" {
		http.ListenAndServe(":"+port, nil)
	} else {
		http.ListenAndServe(":8080", nil)
	}
}
