package cdk

// Bijective is scheme C: plain bijective Base32 over n+1.
//
// Codes contain no redundant symbols and are as short as possible: ids 0..31
// are one character, 32..1055 are two characters, and so on. The mapping is a
// bijection, so no duplicate checks are needed.
//
// The trade-off is that code length reveals the magnitude of n. If the ids are
// sensitive, use Fixed or Variable instead.
type Bijective struct {
	*base
}

// NewBijective creates a variable-length, order-revealing codec. A non-empty
// Key permutes the alphabet but does not hide the length.
func NewBijective(cfg Config) (*Bijective, error) {
	b, err := newBase(cfg, false)
	if err != nil {
		return nil, err
	}
	if len(cfg.Key) > 0 {
		b.setAlphabet(permuteAlphabet(b.alphabet, cfg.Key))
	}
	return &Bijective{base: b}, nil
}

// Encode maps n to its shortest bijective Base32 representation.
func (c *Bijective) Encode(n uint64) (string, error) {
	if n == maxUint64 {
		return "", ErrOutOfRange
	}
	return c.encodeBijective(n + 1)
}

// Decode reverses a code without checking the issued range.
func (c *Bijective) Decode(code string) (uint64, error) {
	s := c.clean(code)
	if len(s) == 0 {
		return 0, ErrInvalidCode
	}
	v, err := c.decodeBijective(s)
	if err != nil {
		return 0, err
	}
	return v - 1, nil
}

// DecodeChecked reverses a code and rejects ids outside [0, IssuedMax).
func (c *Bijective) DecodeChecked(code string) (uint64, error) {
	n, err := c.Decode(code)
	if err != nil {
		return 0, err
	}
	if !c.Valid(n) {
		return 0, ErrNotIssued
	}
	return n, nil
}

// Capacity returns the exclusive upper bound of encodable ids.
func (c *Bijective) Capacity() uint64 { return maxUint64 }
