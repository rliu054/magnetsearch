package main

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"log"

	_ "github.com/mattn/go-sqlite3" //sqlite driver
)

const (
	driver     = "sqlite3"
	datasource = "magnet.db"
)

type Persist struct {
	db *sql.DB
}

var session *Persist

// GetDBSession checks for existing database session, if none is
// available it returns a new database session.
func getDBSession() *Persist {
	if session == nil {
		var err error
		session = new(Persist)

		session.db, err = sql.Open(driver, datasource)
		if err != nil {
			log.Fatal(err)
		}
	}
	return session
}

func (p *Persist) addResource(infohash string) error {
	stmt, err := p.db.Prepare("REPLACE INTO Resources(infohash) VALUES(?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(infohash)
	return err
}

func (p *Persist) deleteOldPeers() error {
	stmt, err := p.db.Prepare("DELETE FROM Peers WHERE DATEDIFF(CURRENT_TIMESTAMP(), ctime) >= 1")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec()
	return err
}

func (p *Persist) addPeer(infohash string, peer []byte) error {
	stmt, err := p.db.Prepare("INSERT INTO Peers(infohash, peers) VALUES(?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(infohash, base64.StdEncoding.EncodeToString(peer))
	return err
}

func (p *Persist) loadPeers(infohash string) ([]string, error) {
	stmt, err := p.db.Prepare("SELECT peers FROM Peers WHERE infohash = ? ORDER BY ctime DESC LIMIT 10")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, _ := stmt.Query(infohash)
	defer rows.Close()
	var link string
	var ret []string
	for rows.Next() {
		rows.Scan(&link)
		data, _ := base64.StdEncoding.DecodeString(link)
		ret = append(ret, bytes.NewBuffer(data).String())
		// ret = append(ret, bytes.NewBuffer(data).String())
	}
	return ret, nil
}

func (p *Persist) updateNodeInfo(id Identifier, routing []byte) error {
	stmt, err := p.db.Prepare("REPLACE INTO Nodes(nodeid, routing, utime) VALUES(?, ?, CURRENT_TIMESTAMP)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(id.hexString(), base64.StdEncoding.EncodeToString(routing))
	return err
}

// LoadAllNodeIDs reads from local datastore and returns all previously saved nodes.
func (p *Persist) loadAllNodeIDs() ([]string, error) {
	rows, err := p.db.Query("SELECT nodeid FROM Nodes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var id string
	var ret []string
	for rows.Next() {
		rows.Scan(&id)
		ret = append(ret, id)
	}
	return ret, nil
}

func (p *Persist) loadNodeInfo(id Identifier) ([]byte, error) {
	stmt, err := p.db.Prepare("SELECT routing FROM Nodes WHERE nodeid = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, _ := stmt.Query(id.hexString())
	var routing string
	for rows.Next() {
		rows.Scan(&routing)
	}
	defer rows.Close()

	data, err := base64.StdEncoding.DecodeString(routing)
	if err != nil {
		return nil, err
	}
	return data, nil
}
