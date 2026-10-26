package cdk

import "fmt"

// Fixed is scheme A: a fixed-length code produced by applying a Feistel
// permutation to the whole code space and then encoding it as fixed-width
// Base32.
//
// Properties: constant length, no duplicates, no magnitude leak and no
// uniqueness checks.
type Fixed struct {
	*base
	f        feistel
	digits   int
	ml, mr   uint64
	capacity uint64
}

// NewFixed creates a fixed-length codec. digits is both the code length and the
// number of Base32 digits. It must be small enough that radix^digits fits in a
// uint64; for the default 32-symbol alphabet that means at most 12.
func NewFixed(digits int, cfg Config) (*Fixed, error) {
	if digits < 1 {
		return nil, fmt.Errorf("%w: digits must be >= 1", ErrConfig)
	}
	b, err := newBase(cfg, true)
	if err != nil {
		return nil, err
	}
	ml, mr, capacity, err := splitDomain(b.radix(), digits)
	if err != nil {
		return nil, err
	}
	return &Fixed{
		base:     b,
		f:        newFeistel(b.key, b.rounds, capacity),
		digits:   digits,
		ml:       ml,
		mr:       mr,
		capacity: capacity,
	}, nil
}

// Encode maps n to a fixed-length code.
func (c *Fixed) Encode(n uint64) (string, error) {
	if n >= c.capacity {
		return "", ErrOutOfRange
	}
	return c.encodeDigits(c.f.permute(n, c.ml, c.mr), c.digits), nil
}

// Decode reverses a code without checking the issued range.
func (c *Fixed) Decode(code string) (uint64, error) {
	s := c.clean(code)
	if len(s) != c.digits {
		return 0, ErrInvalidCode
	}
	v, err := c.decodeDigits(s)
	if err != nil {
		return 0, err
	}
	return c.f.invert(v, c.ml, c.mr), nil
}

// DecodeChecked reverses a code and rejects ids outside [0, IssuedMax).
func (c *Fixed) DecodeChecked(code string) (uint64, error) {
	n, err := c.Decode(code)
	if err != nil {
		return 0, err
	}
	if !c.Valid(n) {
		return 0, ErrNotIssued
	}
	return n, nil
}

// Capacity returns the number of distinct codes.
func (c *Fixed) Capacity() uint64 { return c.capacity }

// Digits returns the code length.
func (c *Fixed) Digits() int { return c.digits }
