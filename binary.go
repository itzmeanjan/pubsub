package pubsub

import (
	"encoding/binary"
	"io"
)

type Binary []byte

func (b Binary) Bytes() []byte {
	return b
}

func (b Binary) WriteTo(w io.Writer) (int64, error) {
	if err := binary.Write(w, binary.BigEndian, uint32(len(b))); err != nil {
		return 0, err
	}

	n, err := w.Write(b)
	return int64(n) + 4, err
}

func (b *Binary) ReadFrom(r io.Reader) (int64, error) {
	var size uint32
	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return 0, err
	}

	*b = make([]byte, size)
	n, err := r.Read(*b)
	return int64(n) + 4, err
}