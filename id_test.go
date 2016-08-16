package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestGenerateID(t *testing.T) {
	id := randID()
	fmt.Println(id.String())
}

func TestHexToID(t *testing.T) {
	s1 := "1111111111111111111111111111111111111110"
	id := hexToID(s1)
	s2 := id.hexString()

	if s1 != s2 {
		t.Errorf("expected: %s, got: %s", s1, s2)
	}
}

func TestIDToHex(t *testing.T) {
	id1 := randID()
	s := id1.hexString()
	id2 := hexToID(s)

	if !bytes.Equal(id1, id2) {
		t.Errorf("expected: %s, got: %s", id1, id2)
	}
}

func TestCalcDistance(t *testing.T) {
	id1 := randID()
	id2 := randID()

	dist12 := distance(id1, id2)
	dist21 := distance(id2, id1)

	if !bytes.Equal(dist12, dist21) {
		t.Errorf("expected %v == %v", dist12, dist21)
	}
}
