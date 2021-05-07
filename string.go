package pubsub

import (
	"encoding/binary"
	"io"
)

// String - Message topic to be written into stream in this form
type String string

func (s String) String() string {
	return string(s)
}

func (s String) Bytes() []byte {
	return []byte(s)
}

// WriteTo - Length prefixed byte content ( topic ) is written into stream
func (s String) WriteTo(w io.Writer) (int64, error) {
	if err := binary.Write(w, binary.BigEndian, uint32(len(s))); err != nil {
		return 0, err
	}

	n, err := w.Write(s.Bytes())
	return int64(n) + 4, err
}

// ReadFrom - Length prefixed byte content ( topic ) is read from stream
func (s *String) ReadFrom(r io.Reader) (int64, error) {
	var size uint32
	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return 0, err
	}

	buf := make([]byte, size)
	n, err := r.Read(buf)
	if err == nil {
		*s = String(buf)
		return int64(n) + 4, err
	}
	return 4, err
}
