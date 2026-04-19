package ndr

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================
// Primitive round-trip tests
// ============================================================

type testPrimitives struct {
	A bool
	B uint8
	C uint16
	D uint32
	E uint64
	F int8
	G int16
	H int32
	I int64
}

func TestRoundTripPrimitives(t *testing.T) {
	original := testPrimitives{
		A: true,
		B: 0xFF,
		C: 0x1234,
		D: 0xDEADBEEF,
		E: 0x123456789ABCDEF0,
		F: -42,
		G: -1234,
		H: -100000,
		I: -9876543210,
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testPrimitives)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original, *decoded, "primitives round-trip mismatch")
}

type testFloats struct {
	A float32
	B float64
}

func TestRoundTripFloats(t *testing.T) {
	original := testFloats{
		A: 3.14,
		B: 2.718281828459045,
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testFloats)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original, *decoded, "floats round-trip mismatch")
}

// Test alignment: fields of different sizes interleaved
type testAlignment struct {
	A uint8
	B uint32
	C uint8
	D uint16
	E uint8
	F uint64
}

func TestRoundTripAlignment(t *testing.T) {
	original := testAlignment{A: 1, B: 2, C: 3, D: 4, E: 5, F: 6}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testAlignment)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original, *decoded, "alignment round-trip mismatch")
}

// ============================================================
// Pointer round-trip tests
// ============================================================

func TestRoundTripEmbeddedPointers(t *testing.T) {
	original := testEmbeddingPointer{
		A: testEmbeddedPointer{
			C: testEmbeddedPointer2{F: 4, G: 5},
			D: 2,
			E: 3,
		},
		B: 1,
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testEmbeddingPointer)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.B, decoded.B)
	assert.Equal(t, original.A.D, decoded.A.D)
	assert.Equal(t, original.A.E, decoded.A.E)
	assert.Equal(t, original.A.C.F, decoded.A.C.F)
	assert.Equal(t, original.A.C.G, decoded.A.C.G)
}

// Top-level ref pointer (no fullpointer tag): referent written inline, no pointer ID
type testTopLevelRefPtr struct {
	A uint32 `ndr:"toplevel"`
	B uint32
}

func TestRoundTripTopLevelRefPointer(t *testing.T) {
	original := testTopLevelRefPtr{A: 42, B: 99}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testTopLevelRefPtr)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original, *decoded, "top-level ref pointer round-trip mismatch")
}

// Top-level full pointer: pointer ID + referent inline
type testTopLevelFullPtr struct {
	A uint32 `ndr:"toplevel,fullpointer"`
	B uint32
}

func TestRoundTripTopLevelFullPointer(t *testing.T) {
	original := testTopLevelFullPtr{A: 42, B: 99}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testTopLevelFullPtr)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original, *decoded, "top-level full pointer round-trip mismatch")
}

// Top-level full pointer with a conformant varying string (like go-smb LsarGetUserName)
type testTopLevelStringPtr struct {
	Name string `ndr:"toplevel,fullpointer,conformant,varying"`
}

func TestRoundTripTopLevelStringPointer(t *testing.T) {
	original := testTopLevelStringPtr{Name: "testuser"}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testTopLevelStringPtr)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.Name, decoded.Name, "top-level string pointer round-trip mismatch")
}

// Embedded pointer to a struct with its own embedded pointers
type testNestedPtrInner struct {
	X uint32 `ndr:"pointer"`
	Y uint32
}

type testNestedPtrOuter struct {
	Inner testNestedPtrInner `ndr:"pointer"`
	Z     uint32
}

func TestRoundTripNestedPointerDeferral(t *testing.T) {
	original := testNestedPtrOuter{
		Inner: testNestedPtrInner{X: 10, Y: 20},
		Z:     30,
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testNestedPtrOuter)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.Z, decoded.Z)
	assert.Equal(t, original.Inner.X, decoded.Inner.X)
	assert.Equal(t, original.Inner.Y, decoded.Inner.Y)
}

// Union with small discriminant (uint8) and large arm (uint64).
// Per C706 §14.3.9 a non-encapsulated union's external alignment is its
// discriminator's alignment (uint8 → 1); arm alignment is internal, applied
// between the discriminator and the arm, not before the union as a whole.
type testBigArmUnion struct {
	Tag uint8  `ndr:"unionTag"`
	A   uint64 `ndr:"unionField"`
	B   uint32 `ndr:"unionField"`
}

func (u testBigArmUnion) SwitchFunc(tag interface{}) string {
	switch tag.(uint8) {
	case 1:
		return "A"
	case 2:
		return "B"
	}
	return ""
}

type testOuterWithBigUnion struct {
	Pad uint8
	U   testBigArmUnion
}

func TestRoundTripUnionBigArmAlignment(t *testing.T) {
	original := testOuterWithBigUnion{
		Pad: 0x42,
		U:   testBigArmUnion{Tag: 1, A: 0xDEADBEEFCAFEBABE},
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	// Expected wire (non-encap, Convention A — disc written twice):
	//   offset 0: Pad = 0x42
	//   offset 1: Tag1 = 1 (disc, uint8 align = 1, no gap)
	//   offset 2: Tag2 = 1 (second disc write)
	//   offset 3..7: pad (arm A is uint64 → 8-align)
	//   offset 8..15: A = 0xDEADBEEFCAFEBABE (little-endian)
	if b[0] != 0x42 {
		t.Errorf("byte 0: expected 0x42 (Pad), got 0x%02x", b[0])
	}
	if b[1] != 1 {
		t.Errorf("byte 1: expected 1 (Tag1), got 0x%02x", b[1])
	}
	if b[2] != 1 {
		t.Errorf("byte 2: expected 1 (Tag2), got 0x%02x", b[2])
	}
	for i := 3; i < 8; i++ {
		if b[i] != 0 {
			t.Errorf("byte %d: expected 0 (padding), got 0x%02x", i, b[i])
		}
	}
	expectedA := []byte{0xBE, 0xBA, 0xFE, 0xCA, 0xEF, 0xBE, 0xAD, 0xDE}
	for i, e := range expectedA {
		if b[8+i] != e {
			t.Errorf("byte %d of A: expected 0x%02x, got 0x%02x", i, e, b[8+i])
		}
	}

	decoded := new(testOuterWithBigUnion)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.Pad, decoded.Pad)
	assert.Equal(t, original.U.Tag, decoded.U.Tag)
	assert.Equal(t, original.U.A, decoded.U.A)
}

// Multi-dimensional conformant array with different dimension sizes (regression test
// for bug where v.Len() was used for all dimensions instead of each dimension's length).
type testMultiDimDifferent struct {
	Data [][]uint32 `ndr:"conformant"`
}

func TestRoundTripMultiDimConformantDifferentSizes(t *testing.T) {
	original := testMultiDimDifferent{
		Data: [][]uint32{
			{1, 2, 3, 4, 5},
			{6, 7, 8, 9, 10},
			{11, 12, 13, 14, 15},
		},
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	// First uint32 should be outer dim (3), second should be inner dim (5)
	// (not 3 and 3 as the buggy code produced)
	outer := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	inner := uint32(b[4]) | uint32(b[5])<<8 | uint32(b[6])<<16 | uint32(b[7])<<24
	if outer != 3 {
		t.Errorf("outer max count: expected 3, got %d", outer)
	}
	if inner != 5 {
		t.Errorf("inner max count: expected 5, got %d", inner)
	}

	decoded := new(testMultiDimDifferent)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original, *decoded, "multi-dim conformant different sizes round-trip mismatch")
}

// Non-ASCII string test: UTF-16 code unit count must differ from Go byte length.
// "café" = 5 UTF-8 bytes, 4 UTF-16 code units. maxCount must use UTF-16 count.
type testNonASCIIString struct {
	S string `ndr:"conformant,skipnull"`
}

func TestRoundTripNonASCIIStringMaxCount(t *testing.T) {
	original := testNonASCIIString{S: "café"}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	// First uint32 is maxCount, must be 4 (UTF-16 code units), NOT 5 (UTF-8 bytes)
	maxCount := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
	if maxCount != 4 {
		t.Errorf("maxCount: expected 4 (UTF-16 code units), got %d", maxCount)
	}
	// Next 4 bytes are offset (0), then actualLen (also 4)
	actualLen := uint32(b[8]) | uint32(b[9])<<8 | uint32(b[10])<<16 | uint32(b[11])<<24
	if actualLen != 4 {
		t.Errorf("actualLen: expected 4, got %d", actualLen)
	}

	decoded := new(testNonASCIIString)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.S, decoded.S, "non-ASCII string round-trip mismatch")
}

// Struct alignment: uint8 followed by struct with uint64.
// The inner struct's alignment is 8, so it must start at an 8-byte boundary.
// Without struct-level alignment, the uint64 would end up at offset 2 after the uint8.
type inner8 struct {
	X uint64
	Y uint32
}
type outer8 struct {
	Pad uint8
	Sub inner8
}

func TestRoundTripStructAlignment8(t *testing.T) {
	original := outer8{Pad: 0x42, Sub: inner8{X: 0xDEADBEEFCAFEBABE, Y: 0x12345678}}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	// Expected layout:
	//   offset 0:  Pad = 0x42
	//   offsets 1-7: padding (struct inner8 needs 8-byte alignment)
	//   offset 8:  X = 0xDEADBEEFCAFEBABE (8 bytes)
	//   offset 16: Y = 0x12345678 (4 bytes)
	// Total: 20 bytes
	if len(b) != 20 {
		t.Fatalf("expected 20 bytes, got %d: %x", len(b), b)
	}
	// Verify X starts at offset 8 (little-endian: be ba fe ca ef be ad de)
	expectedX := []byte{0xbe, 0xba, 0xfe, 0xca, 0xef, 0xbe, 0xad, 0xde}
	for i, e := range expectedX {
		if b[8+i] != e {
			t.Errorf("byte %d: expected %02x, got %02x", 8+i, e, b[8+i])
		}
	}

	decoded := new(outer8)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original, *decoded, "struct alignment round-trip mismatch")
}

// Field with fullpointer tag alone (no pointer tag) - should work symmetrically
// on encoder and decoder
type testFullPointerOnly struct {
	A uint32 `ndr:"fullpointer"`
	B uint32
}

func TestRoundTripFullPointerOnlyTag(t *testing.T) {
	original := testFullPointerOnly{A: 42, B: 99}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testFullPointerOnly)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original, *decoded, "fullpointer-only tag round-trip mismatch")
}

// Nil Go pointer with pointer tag
type testGoPointerStruct struct {
	A *uint32 `ndr:"pointer"`
	B uint32
}

func TestRoundTripGoPointerNonNil(t *testing.T) {
	val := uint32(42)
	original := testGoPointerStruct{A: &val, B: 99}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testGoPointerStruct)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.B, decoded.B)
	assert.NotNil(t, decoded.A)
	assert.Equal(t, *original.A, *decoded.A)
}

// ============================================================
// Complex struct patterns (matching go-smb)
// ============================================================

// Pattern: struct with conformant array + other fields (like LsaprTranslatedNames)
type testConformantArrayStruct struct {
	Count uint32
	Items []uint32 `ndr:"conformant"`
}

func TestRoundTripConformantArrayInStruct(t *testing.T) {
	original := testConformantArrayStruct{
		Count: 4,
		Items: []uint32{100, 200, 300, 400},
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testConformantArrayStruct)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original, *decoded, "conformant array struct round-trip mismatch")
}

// Pattern: struct with pointer to conformant array (like LsaprReferencedDomainList).
// IDL [unique, size_is] → pointer,fullpointer,conformant (nullable embedded pointer).
type testPtrConformantArray struct {
	Count uint32
	Items []uint32 `ndr:"pointer,fullpointer,conformant"`
}

func TestRoundTripPointerConformantArray(t *testing.T) {
	original := testPtrConformantArray{
		Count: 3,
		Items: []uint32{10, 20, 30},
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testPtrConformantArray)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original, *decoded, "pointer conformant array round-trip mismatch")
}

// Pattern: RPC_UNICODE_STRING as tagged struct (Length, MaxLength, Buffer pointer).
// Buffer is IDL [unique] wchar_t* → pointer,fullpointer (nullable embedded pointer).
type testRPCUnicodeString struct {
	Length    uint16
	MaxLength uint16
	Buffer    string `ndr:"pointer,fullpointer,conformant,varying,skipnull,maxcount:MaxLength"`
}

func TestRoundTripRPCUnicodeString(t *testing.T) {
	s := "hello"
	original := testRPCUnicodeString{
		Length:    uint16(len(s) * 2),       // 10
		MaxLength: uint16((len(s) + 1) * 2), // 12 (includes null terminator space)
		Buffer:    s,
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	// Wire layout:
	//   [0-1]: Length (uint16) = 10
	//   [2-3]: MaxLength (uint16) = 12
	//   [4-7]: pointer refID (uint32)
	// Deferred referent:
	//   [8-11]:  MaxCount (uint32) = MaxLength/2 = 6
	//   [12-15]: Offset (uint32) = 0
	//   [16-19]: ActualCount (uint32) = 5
	maxCount := binary.LittleEndian.Uint32(b[8:12])
	assert.Equal(t, uint32(6), maxCount, "MaxCount should be MaxLength/2")
	actualCount := binary.LittleEndian.Uint32(b[16:20])
	assert.Equal(t, uint32(5), actualCount, "ActualCount should be string length in UTF-16 code units")

	decoded := new(testRPCUnicodeString)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.Length, decoded.Length)
	assert.Equal(t, original.MaxLength, decoded.MaxLength)
	assert.Equal(t, original.Buffer, decoded.Buffer)
}

// Pattern: struct with multiple RPC_UNICODE_STRING fields (like SAM user info)
type testMultiUnicodeStrings struct {
	Name    testRPCUnicodeString
	Comment testRPCUnicodeString
	Count   uint32
}

func TestRoundTripMultipleRPCUnicodeStrings(t *testing.T) {
	original := testMultiUnicodeStrings{
		Name:    testRPCUnicodeString{Length: 8, MaxLength: 10, Buffer: "test"},
		Comment: testRPCUnicodeString{Length: 10, MaxLength: 12, Buffer: "hello"},
		Count:   42,
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testMultiUnicodeStrings)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.Name.Buffer, decoded.Name.Buffer)
	assert.Equal(t, original.Comment.Buffer, decoded.Comment.Buffer)
	assert.Equal(t, original.Count, decoded.Count)
}

// Pattern: union with pointer fields (like SAMPR_USER_INFO_BUFFER)
type testUnionWithPointerFields struct {
	Tag    uint32                 `ndr:"unionTag"`
	Info1  testConformantArrayStruct `ndr:"unionField"`
	Info2  testPrimitives          `ndr:"unionField"`
}

func (u testUnionWithPointerFields) SwitchFunc(tag interface{}) string {
	switch tag.(uint32) {
	case 1:
		return "Info1"
	case 2:
		return "Info2"
	}
	return ""
}

func TestRoundTripUnionWithStructFields(t *testing.T) {
	original := testUnionWithPointerFields{
		Tag: 1,
		Info1: testConformantArrayStruct{
			Count: 2,
			Items: []uint32{55, 66},
		},
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testUnionWithPointerFields)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.Tag, decoded.Tag)
	assert.Equal(t, original.Info1.Count, decoded.Info1.Count)
	assert.Equal(t, original.Info1.Items, decoded.Info1.Items)
}

// Pattern: top-level pointer to struct with conformant array (common in RPC responses)
type testResponseStruct struct {
	Entries uint32
	Names   []testRPCUnicodeString `ndr:"pointer,fullpointer,conformant"`
}

func TestRoundTripResponseWithPointerConformantStructArray(t *testing.T) {
	original := testResponseStruct{
		Entries: 2,
		Names: []testRPCUnicodeString{
			{Length: 8, MaxLength: 8, Buffer: "user"},
			{Length: 10, MaxLength: 10, Buffer: "admin"},
		},
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testResponseStruct)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.Entries, decoded.Entries)
	assert.Equal(t, len(original.Names), len(decoded.Names))
	for i := range original.Names {
		assert.Equal(t, original.Names[i].Buffer, decoded.Names[i].Buffer, "name %d mismatch", i)
		assert.Equal(t, original.Names[i].Length, decoded.Names[i].Length, "length %d mismatch", i)
	}
}

// Pattern: struct with fixed byte array (like context handle or GUID)
type testFixedByteArray struct {
	Handle [20]byte
	Status uint32
}

func TestRoundTripFixedByteArray(t *testing.T) {
	original := testFixedByteArray{
		Handle: [20]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
			0x11, 0x12, 0x13, 0x14},
		Status: 0,
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testFixedByteArray)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original, *decoded, "fixed byte array round-trip mismatch")
}

// Pattern: with NDR headers (like DCERPC type serialization)
type testWithHeaders struct {
	A uint32
	B uint32
}

func TestRoundTripWithHeaders(t *testing.T) {
	original := testWithHeaders{A: 0x12345678, B: 0xABCDEF00}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), true)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testWithHeaders)
	dec := NewDecoder(bytes.NewReader(b), true)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original, *decoded, "headers round-trip mismatch")
}

// Pattern: big-endian encoding with headers. Verifies that common header byte 1
// indicates big-endian (0x00) and all multi-byte fields (header length, buffer
// size, payload) use big-endian byte order.
func TestRoundTripBigEndianWithHeaders(t *testing.T) {
	original := testWithHeaders{A: 0x12345678, B: 0xABCDEF00}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), true)
	enc.SetEndianness(binary.BigEndian)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	// Byte 1 of common header should be 0x00 (big-endian + ASCII)
	if b[1] != 0x00 {
		t.Errorf("common header endianness byte: expected 0x00 (big-endian), got 0x%02x", b[1])
	}
	// Header length at bytes 2-3 should be 00 08 (big-endian uint16)
	if b[2] != 0x00 || b[3] != 0x08 {
		t.Errorf("header length bytes 2-3: expected 00 08, got %02x %02x", b[2], b[3])
	}

	decoded := new(testWithHeaders)
	dec := NewDecoder(bytes.NewReader(b), true)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original, *decoded, "big-endian round-trip mismatch")
}

// Pattern: explicit endianness setting
func TestRoundTripLittleEndian(t *testing.T) {
	original := testWithHeaders{A: 0x12345678, B: 0xABCDEF00}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	enc.SetEndianness(binary.LittleEndian)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testWithHeaders)
	dec := NewDecoder(bytes.NewReader(b), false)
	dec.SetEndianness(binary.LittleEndian)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original, *decoded, "little-endian round-trip mismatch")
}

// Pattern: struct with a mix of top-level pointers and conformant arrays
// Simulates a request with multiple RPC arguments
type testRPCRequest struct {
	SystemName string   `ndr:"toplevel,fullpointer,conformant,varying"`
	Level      uint32
	Items      []uint32 `ndr:"toplevel,conformant"`
}

func TestRoundTripRPCRequestPattern(t *testing.T) {
	original := testRPCRequest{
		SystemName: "\\\\server",
		Level:      1,
		Items:      []uint32{10, 20, 30},
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(testRPCRequest)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.SystemName, decoded.SystemName)
	assert.Equal(t, original.Level, decoded.Level)
	assert.Equal(t, original.Items, decoded.Items)
}
