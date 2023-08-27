package bencode

import (
	"bufio"
	"errors"
	"io"
)

type BType uint8

const (
	BINT  BType = 0x01
	BSTR  BType = 0x02
	BLIST BType = 0x03
	BDICT BType = 0x04
)

type BValue interface{}

type BObject struct {
	type_ BType
	val_  BValue
	raw_  []byte
}

func (o *BObject) Int() (int, error) {
	if o.type_ != BINT {
		return 0, errors.New("expect Int")
	}
	return o.val_.(int), nil
}

func (o *BObject) Str() (string, error) {
	if o.type_ != BSTR {
		return "", errors.New("expect String")
	}
	return o.val_.(string), nil
}

func (o *BObject) List() ([]*BObject, error) {
	if o.type_ != BLIST {
		return nil, errors.New("expect List")
	}
	return o.val_.([]*BObject), nil
}

func (o *BObject) Dict() (map[string]*BObject, error) {
	if o.type_ != BDICT {
		return nil, errors.New("expect Dict")
	}
	return o.val_.(map[string]*BObject), nil
}

func (o *BObject) Raw() []byte {
	return o.raw_
}

func (o *BObject) Bencode(w io.Writer) int {
	bw, ok := w.(*bufio.Writer)
	if !ok {
		bw = bufio.NewWriter(w)
	}
	wLen := 0
	switch o.type_ {
	case BINT:
		val, _ := o.Int()
		wLen += EncodeInt(bw, val)
	case BSTR:
		str, _ := o.Str()
		wLen += EncodeString(bw, str)
	case BLIST:
		bw.WriteByte('l')
		list, _ := o.List()
		for _, elem := range list {
			wLen += elem.Bencode(bw)
		}
		bw.WriteByte('e')
		wLen += 2
	case BDICT:
		bw.WriteByte('d')
		dict, _ := o.Dict()
		for k, v := range dict {
			wLen += EncodeString(bw, k)
			wLen += v.Bencode(bw)
		}
		bw.WriteByte('e')
		wLen += 2
	}
	bw.Flush()
	return wLen
}

func Bdecode(r io.Reader) (*BObject, []byte, error) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}

	b, err := br.Peek(1)
	if err != nil {
		return nil, nil, err
	}
	var ret BObject
	raw_ := make([]byte, 0)
	switch {
	case b[0] == 'i':
		// parse int
		val, raw, err := DecodeInt(br)
		if err != nil {
			return nil, nil, err
		}
		ret.type_ = BINT
		ret.val_ = val
		ret.raw_ = append(raw_, raw...)

	case b[0] >= '0' && b[0] <= '9':
		// parse string
		val, raw, err := DecodeString(br)
		if err != nil {
			return nil, nil, err
		}
		ret.type_ = BSTR
		ret.val_ = val
		ret.raw_ = append(raw_, raw...)
	case b[0] == 'l':
		// parse list
		b, _ := br.ReadByte()
		raw_ = append(raw_, b)
		var list []*BObject
		for {
			if p, _ := br.Peek(1); p[0] == 'e' {
				b, _ = br.ReadByte()
				raw_ = append(raw_, b)
				break
			}
			elem, raw, err := Bdecode(br)
			if err != nil {
				return nil, nil, err
			}
			raw_ = append(raw_, raw...)
			list = append(list, elem)
		}
		ret.type_ = BLIST
		ret.val_ = list
		ret.raw_ = raw_
	case b[0] == 'd':
		// parse map
		b, _ := br.ReadByte()
		raw_ = append(raw_, b)
		dict := make(map[string]*BObject)
		for {
			if p, _ := br.Peek(1); p[0] == 'e' {
				b, _ = br.ReadByte()
				raw_ = append(raw_, b)
				break
			}
			key, raw, err := DecodeString(br)
			raw_ = append(raw_, raw...)
			if err != nil {
				return nil, nil, err
			}
			val, raw, err := Bdecode(br)
			raw_ = append(raw_, raw...)
			if err != nil {
				return nil, nil, err
			}
			dict[key] = val
		}
		ret.type_ = BDICT
		ret.val_ = dict
		ret.raw_ = raw_
	default:
		return nil, nil, errors.New("expect num")
	}
	return &ret, ret.raw_, nil
}

func readDecimal(r *bufio.Reader) (val int, raw []byte, len int) {
	sign := 1
	b, _ := r.ReadByte()
	raw = append(raw, b)
	len++
	if b == '-' {
		sign = -1
		b, _ = r.ReadByte()
		raw = append(raw, b)
		len++
	}
	for {
		if !(b >= '0' && b <= '9') {
			r.UnreadByte()
			len--
			raw = raw[:len]
			return sign * val, raw, len
		}
		val = val*10 + int(b-'0')
		b, _ = r.ReadByte()
		raw = append(raw, b)
		len++
	}
}

func writeDecimal(w *bufio.Writer, val int) (len int) {
	if val == 0 {
		w.WriteByte('0')
		len++
		return
	}
	if val < 0 {
		w.WriteByte('-')
		len++
		val *= -1
	}

	dividend := 1
	for {
		if dividend > val {
			dividend /= 10
			break
		}
		dividend *= 10
	}
	for {
		num := byte(val / dividend)
		w.WriteByte('0' + num)
		len++
		if dividend == 1 {
			return
		}
		val %= dividend
		dividend /= 10
	}
}

func EncodeInt(w io.Writer, val int) int {
	bw := bufio.NewWriter(w)
	wLen := 0
	bw.WriteByte('i')
	wLen++
	nLen := writeDecimal(bw, val)
	wLen += nLen
	bw.WriteByte('e')
	wLen++

	err := bw.Flush()
	if err != nil {
		return 0
	}
	return wLen
}

func DecodeInt(r io.Reader) (val int, raw []byte, err error) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	b, _ := br.ReadByte()
	raw = append(raw, b)
	if b != 'i' {
		return val, raw, errors.New("expect num")
	}
	var raw_ []byte
	val, raw_, _ = readDecimal(br)
	raw = append(raw, raw_...)
	b, err = br.ReadByte()
	raw = append(raw, b)
	if b != 'e' {
		return val, raw, errors.New("expect num")
	}
	return
}

func EncodeString(w io.Writer, val string) int {
	strLen := len(val)
	bw := bufio.NewWriter(w)
	wLen := writeDecimal(bw, strLen)
	bw.WriteByte(':')
	wLen++
	bw.WriteString(val)
	wLen += strLen

	err := bw.Flush()
	if err != nil {
		return 0
	}
	return wLen
}

func DecodeString(r io.Reader) (val string, raw []byte, e error) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	num, raw_, len := readDecimal(br)
	raw = append(raw, raw_...)
	if len == 0 {
		return val, raw_, errors.New("expect num")
	}
	b, _ := br.ReadByte()
	raw = append(raw, b)
	if b != ':' {
		return val, raw, errors.New("expect num")
	}
	buf := make([]byte, num)
	_, e = io.ReadAtLeast(br, buf, num)
	raw = append(raw, buf...)
	val = string(buf)
	return
}
