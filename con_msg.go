package pubsub

import "io"

// PublishedMessage - Subscriber will receive message for consumption in this form
type PublishedMessage struct {
	Topic String
	Data  Binary
}

// WriteTo - Writes byte serialised content into given stream
func (p *PublishedMessage) WriteTo(w io.Writer) (int64, error) {
	var n int64

	_n, err := p.Topic.WriteTo(w)
	if err != nil {
		return n, err
	}
	n += _n

	_n, err = p.Data.WriteTo(w)
	if err != nil {
		return n, err
	}
	n += _n

	return n, nil
}

// ReadFrom - Read from byte stream into structured message
func (p *PublishedMessage) ReadFrom(r io.Reader) (int64, error) {
	var n int64

	t := new(String)
	_n, err := t.ReadFrom(r)
	if err != nil {
		return n, err
	}
	n += _n

	b := new(Binary)
	_n, err = b.ReadFrom(r)
	if err != nil {
		return n, err
	}
	n += _n

	p.Topic = *t
	p.Data = *b

	return n, nil
}
