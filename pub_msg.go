package pubsub

import (
	"encoding/binary"
	"io"
)

// Message - Publisher showing intent of publishing arbitrary byte slice to topics
type Message struct {
	Topics []String
	Data   Binary
}

// WriteTo - Writes byte serialised content into given stream
func (m *Message) WriteTo(w io.Writer) (int64, error) {
	var n int64

	if err := binary.Write(w, binary.BigEndian, uint16(len(m.Topics))); err != nil {
		return 0, err
	}

	n += 2

	for i := 0; i < len(m.Topics); i++ {

		if _n, err := m.Topics[i].WriteTo(w); err != nil {
			return n, err
		} else {
			n += _n
		}

	}

	if _n, err := m.Data.WriteTo(w); err != nil {
		return n, err
	} else {
		n += _n
	}

	return n, nil
}

// ReadFrom - Read from byte stream into structured message
func (m *Message) ReadFrom(r io.Reader) (int64, error) {
	var n int64

	var size uint16
	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return 0, err
	}

	n += 2

	buf := make([]String, 0, size)
	for i := 0; i < int(size); i++ {

		t := new(String)
		if _n, err := t.ReadFrom(r); err != nil {
			return n, err
		} else {
			n += _n
			buf = append(buf, *t)
		}

	}

	b := new(Binary)
	if _n, err := b.ReadFrom(r); err != nil {
		return n, err
	} else {
		n += _n
	}

	m.Topics = buf
	m.Data = *b

	return n, nil
}