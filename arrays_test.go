package ndr

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

const TestHeader = "01100800cccccccca00400000000000000000200"

func TestParseDimensions(t *testing.T) {
	a := [2][2][2][]SimpleTest{}
	l, ta := parseDimensions(reflect.ValueOf(a))
	assert.Equal(t, 4, len(l), "dimension count not as expected")
	assert.Equal(t, []int{2, 2, 2, 0}, l, "lengths list not as expected")
	assert.Equal(t, "SimpleTest", ta.Name(), "type within array not as expected")
}

func TestMakeSubSlices(t *testing.T) {
	l := []int{2, 5, 3, 1}
	a := new([][][][]uint32)
	v := reflect.ValueOf(a)
	v = v.Elem()
	ty := v.Type()
	s := reflect.MakeSlice(ty, l[0], l[0])
	v.Set(s)
	makeSubSlices(v, l[1:])
	assert.Equal(t, "[[[[0] [0] [0]] [[0] [0] [0]] [[0] [0] [0]] [[0] [0] [0]] [[0] [0] [0]]] [[[0] [0] [0]] [[0] [0] [0]] [[0] [0] [0]] [[0] [0] [0]] [[0] [0] [0]]]]", fmt.Sprintf("%v", *a))
}

func TestDimensionCountFromTag(t *testing.T) {
	var a structWithTagKey
	v := reflect.ValueOf(a)
	d, err := intFromTag(v.Type().Field(0).Tag, "test")
	if err != nil {
		t.Errorf("error getting dimensions from tag: %v", err)
	}
	assert.Equal(t, 3, d, "number of dimensions not as expected")
}

type StructWithArray struct {
	A [4]uint32
}

type StructWithMultiDimArray struct {
	A [2][3][2]uint32
}

type StructWithConformantSlice struct {
	A []uint32 `ndr:"conformant"`
}

type StructWithVaryingSlice struct {
	A []uint32 `ndr:"varying"`
}

type StructWithConformantVaryingSlice struct {
	A []uint32 `ndr:"conformant,varying"`
}

type StructWithMultiDimensionalConformantSlice struct {
	A [][][]uint32 `ndr:"conformant"`
}

// structWithTagKey carries a key:value tag purely so intFromTag can be
// exercised; it is never encoded or decoded, so the key need not be one the
// library recognises.
type structWithTagKey struct {
	A [][][]uint32 `ndr:"conformant,test:3"`
}

type StructWithMultiDimensionalVaryingSlice struct {
	A [][][]uint32 `ndr:"varying"`
}

type StructWithMultiDimensionalConformantVaryingSlice struct {
	A [][][]uint32 `ndr:"conformant,varying"`
}

func TestReadUniDimensionalFixedArray(t *testing.T) {
	hexStr := TestHeader + "01000000020000000300000004000000"
	b, _ := hex.DecodeString(hexStr)
	a := new(StructWithArray)
	dec := NewDecoder(bytes.NewReader(b), true)
	err := dec.Decode(a)
	if err != nil {
		t.Fatalf("%v", err)
	}
	for i := range a.A {
		assert.Equal(t, uint32(i+1), a.A[i], "Value of index %d not as expected", i)
	}
}

func TestReadMultiDimensionalFixedArray(t *testing.T) {
	hexStr := TestHeader + "0100000002000000030000000400000005000000060000000700000008000000090000000a0000000b0000000c0000000d0000000e0000000f000000100000001100000012000000130000001400000015000000160000001700000018000000190000001a0000001b0000001c0000001d0000001e0000001f0000002000000021000000220000002300000024000000"
	b, _ := hex.DecodeString(hexStr)
	a := new(StructWithMultiDimArray)
	dec := NewDecoder(bytes.NewReader(b), true)
	err := dec.Decode(a)
	if err != nil {
		t.Fatalf("%v", err)
	}
	ar := [2][3][2]uint32{
		{
			{1, 2},
			{3, 4},
			{5, 6},
		},
		{
			{7, 8},
			{9, 10},
			{11, 12},
		},
	}
	assert.Equal(t, ar, a.A, "multi-dimensional fixed array not as expected")
}

func TestReadUniDimensionalConformantArray(t *testing.T) {
	hexStr := TestHeader + "0400000001000000020000000300000004000000"
	b, _ := hex.DecodeString(hexStr)
	a := new(StructWithConformantSlice)
	dec := NewDecoder(bytes.NewReader(b), true)
	err := dec.Decode(a)
	if err != nil {
		t.Fatalf("%v", err)
	}
	for i := range a.A {
		assert.Equal(t, uint32(i+1), a.A[i], "Value of index %d not as expected", i)
	}
}

func TestReadMultiDimensionalConformantArray(t *testing.T) {
	hexStr := TestHeader + "0200000003000000020000000100000002000000030000000400000005000000060000000700000008000000090000000a0000000b0000000c0000000d0000000e0000000f000000100000001100000012000000130000001400000015000000160000001700000018000000190000001a0000001b0000001c0000001d0000001e0000001f0000002000000021000000220000002300000024000000"
	b, _ := hex.DecodeString(hexStr)
	a := new(StructWithMultiDimensionalConformantSlice)
	dec := NewDecoder(bytes.NewReader(b), true)
	err := dec.Decode(a)
	if err != nil {
		t.Fatalf("%v", err)
	}
	ar := [][][]uint32{
		{
			{1, 2},
			{3, 4},
			{5, 6},
		},
		{
			{7, 8},
			{9, 10},
			{11, 12},
		},
	}
	assert.Equal(t, ar, a.A, "multi-dimensional conformant array not as expected")
}

func TestReadUniDimensionalVaryingArray(t *testing.T) {
	hexStr := TestHeader + "000000000400000001000000020000000300000004000000"
	b, _ := hex.DecodeString(hexStr)
	a := new(StructWithVaryingSlice)
	dec := NewDecoder(bytes.NewReader(b), true)
	err := dec.Decode(a)
	if err != nil {
		t.Fatalf("%v", err)
	}
	for i := range a.A {
		assert.Equal(t, uint32(i+1), a.A[i], "Value of index %d not as expected", i)
	}
}

func TestReadMultiDimensionalVaryingArray(t *testing.T) {
	hexStr := TestHeader + "0000000002000000000000000300000000000000020000000100000002000000030000000400000005000000060000000700000008000000090000000a0000000b0000000c0000000d0000000e0000000f000000100000001100000012000000130000001400000015000000160000001700000018000000190000001a0000001b0000001c0000001d0000001e0000001f0000002000000021000000220000002300000024000000"
	b, _ := hex.DecodeString(hexStr)
	a := new(StructWithMultiDimensionalVaryingSlice)
	dec := NewDecoder(bytes.NewReader(b), true)
	err := dec.Decode(a)
	if err != nil {
		t.Fatalf("%v", err)
	}
	ar := [][][]uint32{
		{
			{1, 2},
			{3, 4},
			{5, 6},
		},
		{
			{7, 8},
			{9, 10},
			{11, 12},
		},
	}
	assert.Equal(t, ar, a.A, "multi-dimensional conformant varying array not as expected")
}

func TestReadUniDimensionalConformantVaryingArray(t *testing.T) {
	hexStr := TestHeader + "04000000000000000400000001000000020000000300000004000000"
	b, _ := hex.DecodeString(hexStr)
	a := new(StructWithConformantVaryingSlice)
	dec := NewDecoder(bytes.NewReader(b), true)
	err := dec.Decode(a)
	if err != nil {
		t.Fatalf("%v", err)
	}
	for i := range a.A {
		assert.Equal(t, uint32(i+1), a.A[i], "Value of index %d not as expected", i)
	}
}

func TestReadMultiDimensionalConformantVaryingArray(t *testing.T) {
	hexStr := TestHeader + "0200000003000000020000000000000002000000000000000300000000000000020000000100000002000000030000000400000005000000060000000700000008000000090000000a0000000b0000000c0000000d0000000e0000000f000000100000001100000012000000130000001400000015000000160000001700000018000000190000001a0000001b0000001c0000001d0000001e0000001f0000002000000021000000220000002300000024000000"
	b, _ := hex.DecodeString(hexStr)
	a := new(StructWithMultiDimensionalConformantVaryingSlice)
	dec := NewDecoder(bytes.NewReader(b), true)
	err := dec.Decode(a)
	if err != nil {
		t.Fatalf("%v", err)
	}
	ar := [][][]uint32{
		{
			{1, 2},
			{3, 4},
			{5, 6},
		},
		{
			{7, 8},
			{9, 10},
			{11, 12},
		},
	}
	assert.Equal(t, ar, a.A, "multi-dimensional conformant varying array not as expected")
}

// Round-trip encode/decode tests for arrays

func TestRoundTripUniDimensionalFixedArray(t *testing.T) {
	original := StructWithArray{A: [4]uint32{1, 2, 3, 4}}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(StructWithArray)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.A, decoded.A, "round-trip fixed array mismatch")
}

func TestRoundTripMultiDimensionalFixedArray(t *testing.T) {
	original := StructWithMultiDimArray{A: [2][3][2]uint32{
		{{1, 2}, {3, 4}, {5, 6}},
		{{7, 8}, {9, 10}, {11, 12}},
	}}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(StructWithMultiDimArray)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.A, decoded.A, "round-trip multi-dim fixed array mismatch")
}

func TestRoundTripUniDimensionalConformantArray(t *testing.T) {
	original := StructWithConformantSlice{A: []uint32{1, 2, 3, 4}}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(StructWithConformantSlice)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.A, decoded.A, "round-trip conformant array mismatch")
}

func TestRoundTripUniDimensionalVaryingArray(t *testing.T) {
	original := StructWithVaryingSlice{A: []uint32{1, 2, 3, 4}}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(StructWithVaryingSlice)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.A, decoded.A, "round-trip varying array mismatch")
}

func TestRoundTripUniDimensionalConformantVaryingArray(t *testing.T) {
	original := StructWithConformantVaryingSlice{A: []uint32{1, 2, 3, 4}}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(StructWithConformantVaryingSlice)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.A, decoded.A, "round-trip conformant varying array mismatch")
}

func TestRoundTripMultiDimensionalVaryingArray(t *testing.T) {
	original := StructWithMultiDimensionalVaryingSlice{A: [][][]uint32{
		{{1, 2}, {3, 4}, {5, 6}},
		{{7, 8}, {9, 10}, {11, 12}},
	}}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(StructWithMultiDimensionalVaryingSlice)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.A, decoded.A, "round-trip multi-dim varying array mismatch")
}

func TestRoundTripMultiDimensionalConformantVaryingArray(t *testing.T) {
	original := StructWithMultiDimensionalConformantVaryingSlice{A: [][][]uint32{
		{{1, 2}, {3, 4}, {5, 6}},
		{{7, 8}, {9, 10}, {11, 12}},
	}}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded := new(StructWithMultiDimensionalConformantVaryingSlice)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.A, decoded.A, "round-trip multi-dim conformant varying array mismatch")
}

// -- maxcount on conformant-varying slices --

type StructWithMaxCountConformantVaryingSlice struct {
	A []uint32 `ndr:"conformant,varying,maxcount:1000"`
}

func TestRoundTripMaxCountConformantVaryingArray(t *testing.T) {
	original := StructWithMaxCountConformantVaryingSlice{A: []uint32{1, 2, 3, 4}}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	// Verify wire format: first uint32 is MaxCount=1000, then Offset=0, ActualCount=4
	maxCount := binary.LittleEndian.Uint32(b[0:4])
	offset := binary.LittleEndian.Uint32(b[4:8])
	actualCount := binary.LittleEndian.Uint32(b[8:12])
	assert.Equal(t, uint32(1000), maxCount, "MaxCount should be 1000 from tag")
	assert.Equal(t, uint32(0), offset, "Offset should be 0")
	assert.Equal(t, uint32(4), actualCount, "ActualCount should be slice length")

	// Decode should recover the original data (decoder reads ActualCount elements)
	decoded := new(StructWithMaxCountConformantVaryingSlice)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.A, decoded.A, "round-trip maxcount conformant varying array mismatch")
}

type StructWithSiblingMaxCountSlice struct {
	MaxSize uint32
	Items   []uint32 `ndr:"conformant,varying,maxcount:MaxSize"`
}

func TestRoundTripSiblingMaxCountSlice(t *testing.T) {
	original := StructWithSiblingMaxCountSlice{
		MaxSize: 500,
		Items:   []uint32{10, 20, 30},
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	// Wire format: MaxSize(uint32) then MaxCount(uint32) Offset(uint32) ActualCount(uint32) elements...
	maxSizeField := binary.LittleEndian.Uint32(b[0:4])
	maxCount := binary.LittleEndian.Uint32(b[4:8])
	offset := binary.LittleEndian.Uint32(b[8:12])
	actualCount := binary.LittleEndian.Uint32(b[12:16])
	assert.Equal(t, uint32(500), maxSizeField, "MaxSize field value")
	assert.Equal(t, uint32(500), maxCount, "MaxCount should equal MaxSize field")
	assert.Equal(t, uint32(0), offset, "Offset should be 0")
	assert.Equal(t, uint32(3), actualCount, "ActualCount should be slice length")

	decoded := new(StructWithSiblingMaxCountSlice)
	dec := NewDecoder(bytes.NewReader(b), false)
	err = dec.Decode(decoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	assert.Equal(t, original.Items, decoded.Items, "round-trip sibling maxcount slice mismatch")
}

// TestResolveSiblingPointerFields directly exercises the two sibling-field
// resolvers that drive `maxcount:FieldName`. MS-RPC [unique] LPDWORD
// parameters land in Go as *uint32; when they appear as size_is targets the
// resolver must dereference them. This regression test covers the pointer
// path for both resolvers (used by dcerpc/msrrp BaseRegEnumValue /
// BaseRegQueryValue).
func TestResolveSiblingPointerFields(t *testing.T) {
	type carrier struct {
		MaxLenU32 *uint32
		MaxLenU16 *uint16
	}

	v32 := uint32(4096)
	v16 := uint16(200)
	c := carrier{MaxLenU32: &v32, MaxLenU16: &v16}
	rv := reflect.ValueOf(c)

	got, err := resolveSiblingFieldAsUint32(rv, "MaxLenU32")
	if err != nil {
		t.Fatalf("resolveSiblingFieldAsUint32(*uint32): %v", err)
	}
	assert.Equal(t, uint32(4096), got, "should dereference *uint32 sibling")

	got, err = resolveSiblingUint16AsCodeUnits(rv, "MaxLenU16")
	if err != nil {
		t.Fatalf("resolveSiblingUint16AsCodeUnits(*uint16): %v", err)
	}
	assert.Equal(t, uint32(100), got, "should dereference *uint16 sibling and divide by 2")

	// Nil pointer must produce a clear error, not a panic.
	nilCarrier := carrier{}
	rv = reflect.ValueOf(nilCarrier)
	if _, err = resolveSiblingFieldAsUint32(rv, "MaxLenU32"); err == nil {
		t.Fatal("expected error for nil *uint32 sibling, got nil")
	}
	if _, err = resolveSiblingUint16AsCodeUnits(rv, "MaxLenU16"); err == nil {
		t.Fatal("expected error for nil *uint16 sibling, got nil")
	}
}

type StructWithUint16SiblingMaxCountSlice struct {
	MaxSize uint16
	Items   []uint32 `ndr:"conformant,varying,maxcount:MaxSize"`
}

func TestSiblingMaxCountSliceUint16NotDividedBy2(t *testing.T) {
	original := StructWithUint16SiblingMaxCountSlice{
		MaxSize: 100,
		Items:   []uint32{1, 2},
	}
	enc := NewEncoder(bytes.NewBuffer([]byte{}), false)
	b, err := enc.Encode(&original)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}

	// MaxSize field is uint16 (2 bytes) + 2 bytes padding to align MaxCount
	// MaxCount should be 100 (raw value), NOT 50 (divided by 2 like string resolver)
	maxCount := binary.LittleEndian.Uint32(b[4:8])
	assert.Equal(t, uint32(100), maxCount, "MaxCount should be raw uint16 value, not divided by 2")
}
