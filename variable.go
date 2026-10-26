package cdk

import "fmt"

// Variable is scheme B: a Feistel permutation over a maxLen-digit domain,
// followed by bijective Base32 encoding.
//
// Codes have between 1 and maxLen characters. Because the permutation output is
// uniform, the vast majority of codes still use maxLen characters; the shorter
// ones are spread uniformly over the id space and do not reveal the magnitude
// of n.
type Variable struct {
	*base
	f        feistel
	maxLen   int
	ml, mr   uint64
	capacity uint64
}

// NewVariable creates a variable-length codec. maxLen is the maximum code
// length and must be small enough that radix^maxLen fits in a uint64; for the
// default 32-symbol alphabet that means at most 12.
func NewVariable(maxLen int, cfg Config) (*Variable, error) {
	if maxLen < 1 {
		return nil, fmt.Errorf("%w: maxLen must be >= 1", ErrConfig)
	}
	b, err := newBase(cfg, true)
	if err != nil {
		return nil, err
	}
	ml, mr, capacity, err := splitDomain(b.radix(), maxLen)
	if err != nil {
		return nil, err
	}
	return &Variable{
		base:     b,
		f:        newFeistel(b.key, b.rounds, capacity),
		maxLen:   maxLen,
		ml:       ml,
		mr:       mr,
		capacity: capacity,
	}, nil
}

// Encode maps n to a code of 1..maxLen characters.
func (c *Variable) Encode(n uint64) (string, error) {
	if n >= c.capacity {
		return "", ErrOutOfRange
	}
	v := c.f.permute(n, c.ml, c.mr)
	return c.encodeBijective(v + 1)
}

// Decode reverses a code without checking the issued range.
func (c *Variable) Decode(code string) (uint64, error) {
	s := c.clean(code)
	if len(s) == 0 || len(s) > c.maxLen {
		return 0, ErrInvalidCode
	}
	v, err := c.decodeBijective(s)
	if err != nil {
		return 0, err
	}
	if v > c.capacity {
		return 0, ErrInvalidCode
	}
	return c.f.invert(v-1, c.ml, c.mr), nil
}

// DecodeChecked reverses a code and rejects ids outside [0, IssuedMax).
func (c *Variable) DecodeChecked(code string) (uint64, error) {
	n, err := c.Decode(code)
	if err != nil {
		return 0, err
	}
	if !c.Valid(n) {
		return 0, ErrNotIssued
	}
	return n, nil
}

// Capacity returns the number of distinct ids the codec can encode.
func (c *Variable) Capacity() uint64 { return c.capacity }

// MaxLen returns the maximum code length.
func (c *Variable) MaxLen() int { return c.maxLen }
