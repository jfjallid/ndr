package ndr

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- A1: element counts read from the stream must not drive unbounded allocation ---

type hardConformant struct {
	Data []uint32 `ndr:"conformant"`
}

type hardVarying struct {
	Data []uint32 `ndr:"varying"`
}

type hardConformantVarying struct {
	Data []uint32 `ndr:"conformant,varying"`
}

type hardMultiConformant struct {
	Data [][]uint32 `ndr:"conformant"`
}

type hardPipe struct {
	Data []uint32 `ndr:"pipe"`
}

func TestHostileCountsAreRejected(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		into interface{}
	}{
		{"conformant max", []byte{0xff, 0xff, 0xff, 0xff}, &hardConformant{}},
		{"conformant max moderate", []byte{0x00, 0x00, 0x00, 0x20}, &hardConformant{}},
		{"varying actual count", []byte{0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff}, &hardVarying{}},
		{"conformant varying", []byte{0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff}, &hardConformantVarying{}},
		{"multi-dimensional", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, &hardMultiConformant{}},
		{"pipe chunk", []byte{0xff, 0xff, 0xff, 0xff}, &hardPipe{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Before the bound existed this killed the process outright with
			// "fatal error: runtime: out of memory", which recover() cannot catch.
			err := NewDecoder(bytes.NewReader(tc.in), false).Decode(tc.into)
			require.Error(t, err, "a hostile element count must be rejected")
		})
	}
}

func TestPlausibleCountsStillDecode(t *testing.T) {
	in := hardConformant{Data: []uint32{1, 2, 3}}
	b, err := NewEncoder(bytes.NewBuffer(nil), false).Encode(&in)
	require.NoError(t, err)
	var out hardConformant
	require.NoError(t, NewDecoder(bytes.NewReader(b), false).Decode(&out))
	assert.Equal(t, in.Data, out.Data)
}

func TestSetMaxElements(t *testing.T) {
	in := hardConformant{Data: []uint32{1, 2, 3, 4, 5}}
	b, err := NewEncoder(bytes.NewBuffer(nil), false).Encode(&in)
	require.NoError(t, err)

	dec := NewDecoder(bytes.NewReader(b), false)
	dec.SetMaxElements(2)
	var out hardConformant
	err = dec.Decode(&out)
	require.Error(t, err, "count above the configured bound must be rejected")
	assert.Contains(t, err.Error(), "exceeds the maximum")
}

// --- A2: reflection mismatches must surface as errors, never panics ---

type namedBool bool
type namedU8 uint8
type namedU16 uint16
type namedU32 uint32
type namedU64 uint64
type namedI16 int16
type namedF64 float64
type namedString string

type namedTypes struct {
	A namedBool
	B namedU8
	C namedU16
	D namedU32
	E namedU64
	F namedI16
	G namedF64
	H namedString
}

func TestNamedPrimitiveTypesRoundTrip(t *testing.T) {
	// Only the uint32 case used to convert to the destination type; every other
	// named primitive panicked with "value of type X is not assignable to type Y".
	in := namedTypes{A: true, B: 1, C: 2, D: 3, E: 4, F: -5, G: 6.5, H: "seven"}
	b, err := NewEncoder(bytes.NewBuffer(nil), false).Encode(&in)
	require.NoError(t, err)
	var out namedTypes
	require.NoError(t, NewDecoder(bytes.NewReader(b), false).Decode(&out))
	assert.Equal(t, in, out)
}

type withUnexported struct {
	A uint32
	b uint32
	C uint32
}

func TestUnexportedFieldsAreSkipped(t *testing.T) {
	in := withUnexported{A: 1, C: 2}
	b, err := NewEncoder(bytes.NewBuffer(nil), false).Encode(&in)
	require.NoError(t, err)
	assert.Len(t, b, 8, "the unexported field must not be written to the wire")

	var out withUnexported
	require.NoError(t, NewDecoder(bytes.NewReader(b), false).Decode(&out))
	assert.Equal(t, uint32(1), out.A)
	assert.Equal(t, uint32(2), out.C)
}

// A conformant slice reached without a preceding max count used to index an
// empty slice and panic.
type missingMaxOuter struct {
	Items []missingMaxInner `ndr:"conformant"`
}

type missingMaxInner struct {
	Nested []uint32 `ndr:"conformant"`
}

func TestMissingConformantMaxIsAnError(t *testing.T) {
	b := []byte{0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00}
	var out missingMaxOuter
	err := NewDecoder(bytes.NewReader(b), false).Decode(&out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conformant max count")
}

type unsupportedField struct {
	A uint32
	M map[string]string
}

func TestUnsupportedTypeReportsTheKind(t *testing.T) {
	// This path used to fmt.Printf the kind to stdout and return a bare
	// "unsupported type".
	var out unsupportedField
	err := NewDecoder(bytes.NewReader([]byte{1, 0, 0, 0, 0, 0, 0, 0}), false).Decode(&out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type map")
	assert.Contains(t, err.Error(), "M")

	_, err = NewEncoder(bytes.NewBuffer(nil), false).Encode(&out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type map")
}

// --- A3: count arithmetic must not wrap ---

func TestVaryingOffsetOverflowIsAnError(t *testing.T) {
	// offset 0xFFFFFFFF + actual count 1 wraps to 0 in uint32, which used to
	// yield an empty slice and no error at all.
	b := []byte{0xff, 0xff, 0xff, 0xff, 0x01, 0x00, 0x00, 0x00}
	var out hardVarying
	err := NewDecoder(bytes.NewReader(b), false).Decode(&out)
	require.Error(t, err, "offset+count overflow must be reported")
}

func TestConformantVaryingOverflowIsAnError(t *testing.T) {
	// max count 0, offset 0xFFFFFFFF, actual count 1: the max >= offset+actual
	// check itself used to wrap and pass.
	b := []byte{
		0x00, 0x00, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff,
		0x01, 0x00, 0x00, 0x00,
	}
	var out hardConformantVarying
	err := NewDecoder(bytes.NewReader(b), false).Decode(&out)
	require.Error(t, err)
}

// --- A4: a truncated alignment gap must be reported ---

type alignmentGap struct {
	A uint8
	B uint16
}

func TestTruncatedAlignmentGapIsAnError(t *testing.T) {
	// One byte of input: reading B requires consuming a one-byte alignment gap
	// that is not there. The short discard used to be silently ignored.
	err := NewDecoder(bytes.NewReader([]byte{0x01}), false).Decode(&alignmentGap{})
	require.Error(t, err)
}

func TestDecodeErrorsCarryTheFieldPath(t *testing.T) {
	var out unsupportedField
	err := NewDecoder(bytes.NewReader([]byte{1, 0, 0, 0}), false).Decode(&out)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "unsupportedField"),
		"error should name the struct being decoded, got: %v", err)
}

// --- B1: alignment origin in header (type serialization) mode ---

type hyperWithHeader struct {
	A uint64
	B uint32
	C uint64
}

func TestHeaderModeAlignmentOrigin(t *testing.T) {
	// The encoder writes the top-level referent ID into the buffer it aligns
	// against, so octet 0 of the object buffer is that pointer. The decoder used
	// to restart its counter at 0 *after* skipping the pointer, putting the two
	// sides 4 octets apart and corrupting every 8-octet-aligned field.
	in := hyperWithHeader{A: 0x1122334455667788, B: 0xaabbccdd, C: 0x99aabbccddeeff00}
	b, err := NewEncoder(bytes.NewBuffer(nil), true).Encode(&in)
	require.NoError(t, err)

	var out hyperWithHeader
	require.NoError(t, NewDecoder(bytes.NewReader(b), true).Decode(&out))
	assert.Equal(t, in, out)
}

func TestHeaderModeFourByteFieldsUnaffected(t *testing.T) {
	// Types whose alignment never exceeds 4 are unchanged by the fix: 4 and 0
	// are congruent modulo 4. This is what keeps existing PAC structures decoding
	// exactly as before.
	type fourByte struct {
		A uint32
		B uint16
		C uint8
	}
	in := fourByte{A: 0xdeadbeef, B: 0x1234, C: 0x56}
	b, err := NewEncoder(bytes.NewBuffer(nil), true).Encode(&in)
	require.NoError(t, err)
	var out fourByte
	require.NoError(t, NewDecoder(bytes.NewReader(b), true).Decode(&out))
	assert.Equal(t, in, out)
}

// --- B4: conformant max counts are owned by the scope that hoisted them ---

type scopedUnion struct {
	Tag  uint16     `ndr:"unionTag"`
	ArmA *scopedArm `ndr:"unionField,pointer"`
	ArmB uint32     `ndr:"unionField"`
}

type scopedArm struct {
	Values []uint32 `ndr:"conformant"`
}

func (u scopedUnion) SwitchFunc(t interface{}) string {
	if t.(uint16) == 1 {
		return "ArmA"
	}
	return "ArmB"
}

type scopedInner struct {
	X uint32
}

type scopedOuter struct {
	U scopedUnion
	P *scopedInner `ndr:"pointer"`
}

func TestUnionWithPointerArmDoesNotDisturbNestedScope(t *testing.T) {
	// A pointer arm defers its conformant array into a scope of its own, so
	// nothing is hoisted ahead of the discriminant and the deferred referent
	// that follows the union stays in step.
	in := scopedOuter{U: scopedUnion{Tag: 2, ArmB: 5}, P: &scopedInner{X: 7}}
	b, err := NewEncoder(bytes.NewBuffer(nil), false).Encode(&in)
	require.NoError(t, err)

	var out scopedOuter
	require.NoError(t, NewDecoder(bytes.NewReader(b), false).Decode(&out))
	assert.Equal(t, uint16(2), out.U.Tag)
	assert.Equal(t, uint32(5), out.U.ArmB)
	require.NotNil(t, out.P)
	assert.Equal(t, uint32(7), out.P.X, "deferred referent must not be desynchronised")
}

// --- B3: an arm that would hoist a max count cannot be represented ---

type inlineConformantArm struct {
	Tag  uint16   `ndr:"unionTag"`
	ArmA []uint32 `ndr:"unionField,conformant"`
	ArmB uint32   `ndr:"unionField"`
}

func (u inlineConformantArm) SwitchFunc(t interface{}) string { return "ArmB" }

type nestedConformantArm struct {
	Tag  uint16    `ndr:"unionTag"`
	ArmA scopedArm `ndr:"unionField"`
	ArmB uint32    `ndr:"unionField"`
}

func (u nestedConformantArm) SwitchFunc(t interface{}) string { return "ArmB" }

func TestConformantArrayInUnionArmIsRejected(t *testing.T) {
	// The max count would have to precede the discriminant that selects the
	// arm, so no peer could know how many counts to expect. Both sides used to
	// hoist one per arm, which agreed with itself but with nothing else.
	for _, tc := range []struct {
		name string
		val  interface{}
	}{
		{"conformant slice arm", &inlineConformantArm{Tag: 2, ArmB: 1}},
		{"struct arm containing a conformant slice", &nestedConformantArm{Tag: 2, ArmB: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewEncoder(bytes.NewBuffer(nil), false).Encode(tc.val)
			require.Error(t, err, "encoder must reject the arm")
			assert.Contains(t, err.Error(), "union arm")

			err = NewDecoder(bytes.NewReader(make([]byte, 32)), false).Decode(tc.val)
			require.Error(t, err, "decoder must reject the arm")
			assert.Contains(t, err.Error(), "union arm")
		})
	}
}

func TestUnionWithPointerArmEmitsNoSpuriousMaxCount(t *testing.T) {
	// Selecting ArmB must not put ArmA's max count on the wire.
	in := scopedUnion{Tag: 2, ArmB: 0x99}
	b, err := NewEncoder(bytes.NewBuffer(nil), false).Encode(&in)
	require.NoError(t, err)
	// discriminant twice (non-encapsulated) + pad + ArmB
	assert.Equal(t, []byte{0x02, 0x00, 0x02, 0x00, 0x99, 0x00, 0x00, 0x00}, b)
}

type scopedConformant struct {
	Items []uint32      `ndr:"conformant"`
	P     *scopedNested `ndr:"pointer"`
}

type scopedNested struct {
	Inner []uint32 `ndr:"conformant"`
}

func TestNestedScopesKeepTheirOwnMaxCounts(t *testing.T) {
	in := scopedConformant{Items: []uint32{1, 2, 3}, P: &scopedNested{Inner: []uint32{9, 8}}}
	b, err := NewEncoder(bytes.NewBuffer(nil), false).Encode(&in)
	require.NoError(t, err)

	var out scopedConformant
	require.NoError(t, NewDecoder(bytes.NewReader(b), false).Decode(&out))
	assert.Equal(t, []uint32{1, 2, 3}, out.Items)
	require.NotNil(t, out.P)
	assert.Equal(t, []uint32{9, 8}, out.P.Inner)
}

// --- B2: no trailing padding after string data ---

type stringThenSmallFields struct {
	S string `ndr:"conformant,varying"`
	B uint8
	C uint8
}

func TestStringFollowedBySmallFieldRoundTrips(t *testing.T) {
	// The encoder used to pad to a 4-byte boundary after the string data while
	// the decoder never consumed it, so B and C were read out of the padding.
	in := stringThenSmallFields{S: "ab", B: 0x11, C: 0x22}
	b, err := NewEncoder(bytes.NewBuffer(nil), false).Encode(&in)
	require.NoError(t, err)

	var out stringThenSmallFields
	require.NoError(t, NewDecoder(bytes.NewReader(b), false).Decode(&out))
	assert.Equal(t, in, out)
}

func TestStringFollowedByAlignedFieldStillPads(t *testing.T) {
	// Removing the trailing pad does not remove alignment: a following uint32
	// still aligns itself, so the gap appears exactly where NDR puts it.
	type strThenU32 struct {
		S string `ndr:"conformant,varying"`
		N uint32
	}
	in := strThenU32{S: "ab", N: 0xdeadbeef}
	b, err := NewEncoder(bytes.NewBuffer(nil), false).Encode(&in)
	require.NoError(t, err)
	assert.Equal(t, 0, len(b)%4, "the uint32 must land on a 4-byte boundary")

	var out strThenU32
	require.NoError(t, NewDecoder(bytes.NewReader(b), false).Decode(&out))
	assert.Equal(t, in, out)
}

// --- B6: a conformant-only string array carries no offset/actual count ---

type conformantStringArray struct {
	Names []string `ndr:"conformant"`
}

type conformantVaryingStringArray struct {
	Names []string `ndr:"conformant,varying"`
}

func TestConformantOnlyStringArrayOmitsVaryingMetadata(t *testing.T) {
	in := conformantStringArray{Names: []string{"ab", "cd"}}
	b, err := NewEncoder(bytes.NewBuffer(nil), false).Encode(&in)
	require.NoError(t, err)

	var out conformantStringArray
	require.NoError(t, NewDecoder(bytes.NewReader(b), false).Decode(&out))
	assert.Equal(t, in.Names, out.Names)

	// A conformant+varying array of the same content carries 8 more bytes: the
	// array's own offset and actual count.
	inV := conformantVaryingStringArray{Names: []string{"ab", "cd"}}
	bV, err := NewEncoder(bytes.NewBuffer(nil), false).Encode(&inV)
	require.NoError(t, err)
	assert.Equal(t, 8, len(bV)-len(b), "only the varying form carries offset+actual count")

	var outV conformantVaryingStringArray
	require.NoError(t, NewDecoder(bytes.NewReader(bV), false).Decode(&outV))
	assert.Equal(t, inV.Names, outV.Names)
}

// --- C3: unrecognised struct tags are rejected ---

type typoTag struct {
	S []uint32 `ndr:"conformnat"`
}

type typoKey struct {
	S []uint32 `ndr:"conformant,maxcout:4"`
}

type badMaxCountSibling struct {
	N uint32
	S []uint32 `ndr:"conformant,maxcount:Nope"`
}

type goodMaxCountSibling struct {
	N uint32
	S []uint32 `ndr:"conformant,maxcount:N"`
}

type nestedTypo struct {
	Inner struct {
		S string `ndr:"conformant,varyng"`
	}
}

func TestUnknownTagsAreRejected(t *testing.T) {
	for _, tc := range []struct {
		name string
		val  interface{}
		want string
	}{
		{"misspelled value tag", &typoTag{}, `unknown ndr tag "conformnat"`},
		{"misspelled key tag", &typoKey{}, `unknown ndr tag key "maxcout"`},
		{"maxcount naming a missing sibling", &badMaxCountSibling{}, `does not exist`},
		{"typo nested in a sub-struct", &nestedTypo{}, `unknown ndr tag "varyng"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewEncoder(nil, false).Encode(tc.val)
			require.Error(t, err, "encoder must reject the tag")
			assert.Contains(t, err.Error(), tc.want)

			err = NewDecoder(bytes.NewReader(make([]byte, 32)), false).Decode(tc.val)
			require.Error(t, err, "decoder must reject the tag")
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestValidTagsAreAccepted(t *testing.T) {
	in := goodMaxCountSibling{N: 4, S: []uint32{1, 2}}
	_, err := NewEncoder(nil, false).Encode(&in)
	require.NoError(t, err)
}

func TestStrictTagsCanBeDisabled(t *testing.T) {
	enc := NewEncoder(nil, false)
	enc.SetStrictTags(false)
	_, err := enc.Encode(&typoTag{S: []uint32{1}})
	require.NoError(t, err, "the escape hatch must allow an uncorrectable struct through")
}

// --- C4: an Encoder can be reused, and honours a caller-supplied prefix ---

func TestEncodeIsRepeatable(t *testing.T) {
	enc := NewEncoder(bytes.NewBuffer(nil), false)
	in := scopedInner{X: 1}
	first, err := enc.Encode(&in)
	require.NoError(t, err)
	firstCopy := append([]byte(nil), first...)
	second, err := enc.Encode(&in)
	require.NoError(t, err)
	assert.Equal(t, firstCopy, second, "a second Encode used to append to the first")
}

func TestEncoderPreservesCallerPrefix(t *testing.T) {
	// A prefix already in the buffer is kept, is not counted towards NDR
	// alignment, and is not included in the returned slice.
	prefix := []byte{0xaa, 0xbb, 0xcc}
	buf := bytes.NewBuffer(append([]byte(nil), prefix...))
	enc := NewEncoder(buf, false)

	in := hyperWithHeader{A: 0x1122334455667788, B: 1, C: 2}
	out, err := enc.Encode(&in)
	require.NoError(t, err)
	assert.Equal(t, append(prefix, out...), buf.Bytes(), "prefix must be retained in the buffer")

	// The encoding must be identical to one produced without a prefix.
	clean, err := NewEncoder(nil, false).Encode(&in)
	require.NoError(t, err)
	assert.Equal(t, clean, out, "a caller prefix must not shift NDR alignment")
}

func TestNewEncoderAcceptsNilBuffer(t *testing.T) {
	b, err := NewEncoder(nil, false).Encode(&scopedInner{X: 7})
	require.NoError(t, err)
	assert.Equal(t, []byte{7, 0, 0, 0}, b)
}

// --- C6: errors carry a matchable sentinel and preserve their cause ---

func TestErrorsMatchSentinelAndUnwrap(t *testing.T) {
	err := NewDecoder(bytes.NewReader([]byte{0x01}), false).Decode(&alignmentGap{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrMalformed), "should match ErrMalformed, got %v", err)
	assert.True(t, errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF),
		"the underlying read failure should stay reachable, got %v", err)
}
