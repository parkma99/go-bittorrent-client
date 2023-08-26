package bencode

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	val := "abc"
	buf := new(bytes.Buffer)
	wLen := EncodeString(buf, val)
	assert.Equal(t, 5, wLen)
	str, _ := DecodeString(buf)
	assert.Equal(t, val, str)

	val = ""
	for i := 0; i < 20; i++ {
		val += string(byte('a' + i))
	}
	buf.Reset()
	wLen = EncodeString(buf, val)
	assert.Equal(t, 23, wLen)
	str, _ = DecodeString(buf)
	assert.Equal(t, val, str)
}

func TestInt(t *testing.T) {
	val := 999
	buf := new(bytes.Buffer)
	wLen := EncodeInt(buf, val)
	assert.Equal(t, 5, wLen)
	iv, _ := DecodeInt(buf)
	assert.Equal(t, val, iv)

	val = 0
	buf.Reset()
	wLen = EncodeInt(buf, val)
	assert.Equal(t, 3, wLen)
	iv, _ = DecodeInt(buf)
	assert.Equal(t, val, iv)

	val = -99
	buf.Reset()
	wLen = EncodeInt(buf, val)
	assert.Equal(t, 5, wLen)
	iv, _ = DecodeInt(buf)
	assert.Equal(t, val, iv)
}

func TestBencode(t *testing.T) {
	testCases := []struct {
		name      string
		input     *BObject
		wantError error
		wantLen   int
	}{
		{
			name: "empty string",
			input: &BObject{
				type_: BSTR,
				val_:  "",
			},
			wantError: nil,
			wantLen:   2, // "0:"
		},
		{
			name: "string",
			input: &BObject{
				type_: BSTR,
				val_:  "Hello, world!",
			},
			wantError: nil,
			wantLen:   16, // "13:Hello, world!"
		},
		{
			name: "empty list",
			input: &BObject{
				type_: BLIST,
				val_:  []*BObject{},
			},
			wantError: nil,
			wantLen:   2, // "le"
		},
		{
			name: "list",
			input: &BObject{
				type_: BLIST,
				val_: []*BObject{
					{type_: BSTR, val_: "hello"},
					{type_: BINT, val_: 123},
				},
			},
			wantError: nil,
			wantLen:   14, // "l5:helloi123ee"
		},
		{
			name: "empty dict",
			input: &BObject{
				type_: BDICT,
				val_:  map[string]*BObject{},
			},
			wantError: nil,
			wantLen:   2, // "de"
		},
		{
			name: "dict",
			input: &BObject{
				type_: BDICT,
				val_: map[string]*BObject{
					"hello": {type_: BSTR, val_: "world"},
					"num":   {type_: BINT, val_: 123},
				},
			},
			wantError: nil,
			wantLen:   26, // "d5:hello5:world3:numi123ee"
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bb := &bytes.Buffer{}
			gotLen := tc.input.Bencode(bb)
			if gotLen != tc.wantLen {
				t.Errorf("Bencode() got len = %d, want %d", bb.Len(), tc.wantLen)
			}
		})
	}
}

func objAssertStr(t *testing.T, expect string, o *BObject) {
	assert.Equal(t, BSTR, o.type_)
	str, err := o.Str()
	assert.Equal(t, nil, err)
	assert.Equal(t, expect, str)
}

func objAssertInt(t *testing.T, expect int, o *BObject) {
	assert.Equal(t, BINT, o.type_)
	val, err := o.Int()
	assert.Equal(t, nil, err)
	assert.Equal(t, expect, val)
}

func TestDecodeString(t *testing.T) {
	var o *BObject
	in := "3:abc"
	buf := bytes.NewBufferString(in)
	o, _ = Bdecode(buf)
	objAssertStr(t, "abc", o)

	out := bytes.NewBufferString("")
	assert.Equal(t, len(in), o.Bencode(out))
	assert.Equal(t, in, out.String())
}

func TestDecodeInt(t *testing.T) {
	var o *BObject
	in := "i123e"
	buf := bytes.NewBufferString(in)
	o, _ = Bdecode(buf)
	objAssertInt(t, 123, o)

	out := bytes.NewBufferString("")
	assert.Equal(t, len(in), o.Bencode(out))
	assert.Equal(t, in, out.String())
}

func TestDecodeList(t *testing.T) {
	var o *BObject
	var list []*BObject
	in := "li123e6:archeri789ee"
	buf := bytes.NewBufferString(in)
	o, _ = Bdecode(buf)
	assert.Equal(t, BLIST, o.type_)
	list, err := o.List()
	assert.Equal(t, nil, err)
	assert.Equal(t, 3, len(list))
	objAssertInt(t, 123, list[0])
	objAssertStr(t, "archer", list[1])
	objAssertInt(t, 789, list[2])

	out := bytes.NewBufferString("")
	assert.Equal(t, len(in), o.Bencode(out))
	assert.Equal(t, in, out.String())
}

func TestDecodeMap(t *testing.T) {
	var o *BObject
	var dict map[string]*BObject
	in := "d4:name6:archer3:agei29ee"
	buf := bytes.NewBufferString(in)
	o, _ = Bdecode(buf)
	assert.Equal(t, BDICT, o.type_)
	dict, err := o.Dict()
	assert.Equal(t, nil, err)
	objAssertStr(t, "archer", dict["name"])
	objAssertInt(t, 29, dict["age"])

	out := bytes.NewBufferString("")
	assert.Equal(t, len(in), o.Bencode(out))
}

func TestDecodeComMap(t *testing.T) {
	var o *BObject
	var dict map[string]*BObject
	in := "d4:userd4:name6:archer3:agei29ee5:valueli80ei85ei90eee"
	buf := bytes.NewBufferString(in)
	o, _ = Bdecode(buf)
	assert.Equal(t, BDICT, o.type_)
	dict, err := o.Dict()
	assert.Equal(t, nil, err)
	assert.Equal(t, BDICT, dict["user"].type_)
	assert.Equal(t, BLIST, dict["value"].type_)
}
