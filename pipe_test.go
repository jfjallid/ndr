package ndr

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testPipe = "04000000010000000200000003000000040000000300000001000000020000000300000000000000"

type structWithPipe struct {
	A []uint32 `ndr:"pipe"`
}

func TestRoundTripPipe(t *testing.T) {
	original := structWithPipe{A: []uint32{1, 2, 3, 4, 1, 2, 3}}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	decoded := new(structWithPipe)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.A, decoded.A, "round-trip pipe mismatch")
}

func TestFillPipe(t *testing.T) {
	hexStr := TestHeader + testPipe
	b, _ := hex.DecodeString(hexStr)
	a := new(structWithPipe)
	dec := NewDecoder(bytes.NewReader(b), true)
	err := dec.Decode(a)
	if err != nil {
		t.Fatalf("%v", err)
	}
	tp := []uint32{1, 2, 3, 4, 1, 2, 3}
	assert.Equal(t, tp, a.A, "Value of pipe not as expected")
}
