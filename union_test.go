package ndr

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testUnionSelected1Enc    = "0100000001"
	testUnionSelected2Enc    = "020000000200"
	testUnionSelected1NonEnc = "010000000100000001"
	testUnionSelected2NonEnc = "02000000020000000200"
)

type testUnionEncapsulated struct {
	Tag    uint32 `ndr:"unionTag,encapsulated"`
	Value1 uint8  `ndr:"unionField"`
	Value2 uint16 `ndr:"unionField"`
}

type testUnionNonEncapsulated struct {
	Tag    uint32 `ndr:"unionTag"`
	Value1 uint8  `ndr:"unionField"`
	Value2 uint16 `ndr:"unionField"`
}

func (u testUnionEncapsulated) SwitchFunc(tag interface{}) string {
	t := tag.(uint32)
	switch t {
	case 1:
		return "Value1"
	case 2:
		return "Value2"
	}
	return ""
}

func (u testUnionNonEncapsulated) SwitchFunc(tag interface{}) string {
	t := tag.(uint32)
	switch t {
	case 1:
		return "Value1"
	case 2:
		return "Value2"
	}
	return ""
}

func Test_encodeUnionEncapsulated(t *testing.T) {
	var tests = []struct {
		Hex string
		Tag uint32
		V1  uint8
		V2  uint16
	}{
		{testUnionSelected1Enc, uint32(1), uint8(1), uint16(0)},
		{testUnionSelected2Enc, uint32(2), uint8(0), uint16(2)},
	}

	for i, test := range tests {
		s := testUnionEncapsulated{Tag: test.Tag, Value1: test.V1, Value2: test.V2}
		enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
		b, err := enc.Encode(&s)
		if err != nil {
			t.Fatalf("test %d: encode error: %v", i+1, err)
		}
		expected, _ := hex.DecodeString(test.Hex)
		assert.Equal(t, expected, b, "Encoded bytes not as expected for test: %d", i+1)
	}
}

func Test_encodeUnionNonEncapsulated(t *testing.T) {
	var tests = []struct {
		Hex string
		Tag uint32
		V1  uint8
		V2  uint16
	}{
		{testUnionSelected1NonEnc, uint32(1), uint8(1), uint16(0)},
		{testUnionSelected2NonEnc, uint32(2), uint8(0), uint16(2)},
	}

	for i, test := range tests {
		s := testUnionNonEncapsulated{Tag: test.Tag, Value1: test.V1, Value2: test.V2}
		enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
		b, err := enc.Encode(&s)
		if err != nil {
			t.Fatalf("test %d: encode error: %v", i+1, err)
		}
		expected, _ := hex.DecodeString(test.Hex)
		assert.Equal(t, expected, b, "Encoded bytes not as expected for test: %d", i+1)
	}
}

func Test_roundTripUnionEncapsulated(t *testing.T) {
	var tests = []struct {
		Tag uint32
		V1  uint8
		V2  uint16
	}{
		{uint32(1), uint8(1), uint16(0)},
		{uint32(2), uint8(0), uint16(2)},
	}

	for i, test := range tests {
		original := testUnionEncapsulated{Tag: test.Tag, Value1: test.V1, Value2: test.V2}
		enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
		b, err := enc.Encode(&original)
		if err != nil {
			t.Fatalf("test %d: encode error: %v", i+1, err)
		}

		decoded := new(testUnionEncapsulated)
		dec := NewDecoder(bytes.NewReader(b), false)
		err = dec.Decode(decoded)
		if err != nil {
			t.Fatalf("test %d: decode error: %v", i+1, err)
		}
		assert.Equal(t, original.Tag, decoded.Tag, "Tag mismatch for test: %d", i+1)
		assert.Equal(t, original.Value1, decoded.Value1, "Value1 mismatch for test: %d", i+1)
		assert.Equal(t, original.Value2, decoded.Value2, "Value2 mismatch for test: %d", i+1)
	}
}

func Test_roundTripUnionNonEncapsulated(t *testing.T) {
	var tests = []struct {
		Tag uint32
		V1  uint8
		V2  uint16
	}{
		{uint32(1), uint8(1), uint16(0)},
		{uint32(2), uint8(0), uint16(2)},
	}

	for i, test := range tests {
		original := testUnionNonEncapsulated{Tag: test.Tag, Value1: test.V1, Value2: test.V2}
		enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
		b, err := enc.Encode(&original)
		if err != nil {
			t.Fatalf("test %d: encode error: %v", i+1, err)
		}

		decoded := new(testUnionNonEncapsulated)
		dec := NewDecoder(bytes.NewReader(b), false)
		err = dec.Decode(decoded)
		if err != nil {
			t.Fatalf("test %d: decode error: %v", i+1, err)
		}
		assert.Equal(t, original.Tag, decoded.Tag, "Tag mismatch for test: %d", i+1)
		assert.Equal(t, original.Value1, decoded.Value1, "Value1 mismatch for test: %d", i+1)
		assert.Equal(t, original.Value2, decoded.Value2, "Value2 mismatch for test: %d", i+1)
	}
}

func Test_readUnionEncapsulated(t *testing.T) {
	var tests = []struct {
		Hex string
		Tag uint32
		V1  uint8
		V2  uint16
	}{
		{testUnionSelected1Enc, uint32(1), uint8(1), uint16(0)},
		{testUnionSelected2Enc, uint32(2), uint8(0), uint16(2)},
	}

	for i, test := range tests {
		a := new(testUnionEncapsulated)
		hexStr := TestHeader + test.Hex
		b, _ := hex.DecodeString(hexStr)
		dec := NewDecoder(bytes.NewReader(b), true)
		err := dec.Decode(a)
		if err != nil {
			t.Fatalf("test %d: %v", i+1, err)
		}
		assert.Equal(t, test.Tag, a.Tag, "Tag value not as expected for test: %d", i+1)
		assert.Equal(t, test.V1, a.Value1, "Value1 not as expected for test: %d", i+1)
		assert.Equal(t, test.V2, a.Value2, "Value2 value not as expected for test: %d", i+1)

	}
}

// testUnionNonEncBigArm exercises a non-encapsulated union whose selected
// arm has stricter alignment (8) than the discriminator (4). Per C706
// §14.3.9 the wire layout is `[disc1][disc2][pad to arm align][arm]`; the
// external alignment of the union is the discriminator's alignment, not
// max(disc, arms). This verifies the fix for structAlignment applied to
// non-encapsulated union structs.
type testUnionNonEncBigArm struct {
	Tag  uint32                        `ndr:"unionTag"`
	ArmA uint8                         `ndr:"unionField"`
	ArmB testUnionNonEncBigArmBigBody `ndr:"unionField"`
}

type testUnionNonEncBigArmBigBody struct {
	X uint64
	Y uint32
}

func (u testUnionNonEncBigArm) SwitchFunc(tag interface{}) string {
	switch tag.(uint32) {
	case 1:
		return "ArmA"
	case 2:
		return "ArmB"
	}
	return ""
}

// testUnionNonEncBigArmWithPrefix verifies the fix still pads correctly when
// the union is nested inside another struct at a non-8-aligned offset.
type testUnionNonEncBigArmWithPrefix struct {
	Prefix uint16
	U      testUnionNonEncBigArm
}

func Test_encodeUnionNonEncapsulatedBigArm(t *testing.T) {
	// Tag=2 selects ArmB. Wire:
	//   disc1: 02 00 00 00  (offset 0)
	//   disc2: 02 00 00 00  (offset 4, 4-aligned — no pad)
	//   arm pad to 8:       (offset 8 already 8-aligned — no pad)
	//   X:     88 77 66 55 44 33 22 11
	//   Y:     cc bb aa 99
	// Total 20 bytes.
	const expectedHex = "02000000" + "02000000" + "8877665544332211" + "ccbbaa99"
	s := testUnionNonEncBigArm{
		Tag:  2,
		ArmB: testUnionNonEncBigArmBigBody{X: 0x1122334455667788, Y: 0x99AABBCC},
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&s)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	expected, _ := hex.DecodeString(expectedHex)
	assert.Equal(t, expected, b, "Encoded bytes not as expected")
}

func Test_roundTripUnionNonEncapsulatedBigArm(t *testing.T) {
	original := testUnionNonEncBigArm{
		Tag:  2,
		ArmB: testUnionNonEncBigArmBigBody{X: 0x1122334455667788, Y: 0x99AABBCC},
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testUnionNonEncBigArm)
	dec := NewDecoder(bytes.NewReader(b), false)
	if err := dec.Decode(decoded); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.Tag, decoded.Tag)
	assert.Equal(t, original.ArmB, decoded.ArmB)
}

func Test_encodeUnionNonEncapsulatedBigArmNested(t *testing.T) {
	// Prefix(uint16) at offset 0: aa bb
	// pad to union's external alignment = disc_align = 4: 2 bytes
	//   (offset 2 → offset 4)
	// disc1 at offset 4: 02 00 00 00
	// disc2 at offset 8: 02 00 00 00
	// arm pad to 8 at offset 12: 4 bytes  → offset 16
	// X: 88 77 66 55 44 33 22 11
	// Y: cc bb aa 99
	// Confirms union-as-containee alignment is 4 (not 8).
	const expectedHex = "bbaa" + "0000" + "02000000" + "02000000" + "00000000" + "8877665544332211" + "ccbbaa99"
	s := testUnionNonEncBigArmWithPrefix{
		Prefix: 0xaabb,
		U: testUnionNonEncBigArm{
			Tag:  2,
			ArmB: testUnionNonEncBigArmBigBody{X: 0x1122334455667788, Y: 0x99AABBCC},
		},
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&s)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	expected, _ := hex.DecodeString(expectedHex)
	assert.Equal(t, expected, b, "Encoded bytes not as expected")
}

func Test_readUnionNonEncapsulated(t *testing.T) {
	var tests = []struct {
		Hex string
		Tag uint32
		V1  uint8
		V2  uint16
	}{
		{testUnionSelected1NonEnc, uint32(1), uint8(1), uint16(0)},
		{testUnionSelected2NonEnc, uint32(2), uint8(0), uint16(2)},
	}

	for i, test := range tests {
		a := new(testUnionNonEncapsulated)
		hexStr := TestHeader + test.Hex
		b, _ := hex.DecodeString(hexStr)
		dec := NewDecoder(bytes.NewReader(b), true)
		err := dec.Decode(a)
		if err != nil {
			t.Fatalf("test %d: %v", i+1, err)
		}
		assert.Equal(t, test.Tag, a.Tag, "Tag value not as expected for test: %d", i+1)
		assert.Equal(t, test.V1, a.Value1, "Value1 not as expected for test: %d", i+1)
		assert.Equal(t, test.V2, a.Value2, "Value2 value not as expected for test: %d", i+1)

	}
}
