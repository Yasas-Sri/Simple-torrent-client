package bencode

import (
	"errors"
	"fmt"
	"strconv"
	"unicode"
)

type Decoder struct {
	data []byte
	pos  int
}

func NewDecoder(data []byte) *Decoder {
	return &Decoder{data: data}
}

func (d *Decoder) Pos() int {
	return d.pos
}

func (d *Decoder) Decode() (Bvalue, error) {

	if d.pos >= len(d.data) {
		return nil, errors.New("end of data")
	}

	b := d.data[d.pos]

	switch {
	case b == 'i':
		return d.decodeInt()
	case b == 'l':
		return d.decodeList()
	case b == 'd':
		return d.decodeDict()
	case unicode.IsDigit(rune(b)):
		return d.decodeString()
	default:
		return nil, fmt.Errorf("invalid bencode byte: %c", b)
	}

}

func (d *Decoder) decodeInt() (Bvalue, error) {

	d.pos++

	start := d.pos
	for d.data[d.pos] != 'e' {
		d.pos++
		if d.pos >= len(d.data) {
			return nil, errors.New("unterminated integer")
		}
	}

	numStr := string(d.data[start:d.pos])
	d.pos++

	n, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return nil, err
	}

	return BInt(n), nil
}

func (d *Decoder) decodeString() (Bvalue, error) {
	start := d.pos

	for d.data[d.pos] != ':' {
		if !unicode.IsDigit(rune(d.data[d.pos])) {
			return nil, errors.New("invalid string length")
		}
		d.pos++
	}

	lengthStr := string(d.data[start:d.pos])
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, err
	}

	d.pos++

	if d.pos+length > len(d.data) {
		return nil, errors.New("string exceeds data length")
	}

	str := d.data[d.pos : d.pos+length]
	d.pos += length

	return BString(str), nil
}

func (d *Decoder) decodeList() (Bvalue, error) {
	d.pos++

	var list BList

	for d.data[d.pos] != 'e' {
		val, err := d.Decode()
		if err != nil {
			return nil, err
		}
		list = append(list, val)
	}

	d.pos++
	return list, nil
}

func (d *Decoder) decodeDict() (Bvalue, error) {
	d.pos++

	dict := make(BDict)

	for d.data[d.pos] != 'e' {
		keyVal, err := d.decodeString()
		if err != nil {
			return nil, err
		}

		key := string(keyVal.(BString))

		val, err := d.Decode()
		if err != nil {
			return nil, err
		}

		dict[key] = val
	}

	d.pos++
	return dict, nil
}

func (d *Decoder) DecodeDictWithSpan() (BDict, []byte, error) {
	start := d.pos

	val, err := d.decodeDict()
	if err != nil {
		return nil, nil, err
	}

	end := d.pos
	raw := d.data[start:end]

	return val.(BDict), raw, nil
}
