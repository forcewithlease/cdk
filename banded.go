package cdk

import "fmt"

// Banded is a length-banded codec: the id range determines the code length.
//
// The bijective Base32 value space is split into contiguous bands, one per
// code length. Within each band a Feistel permutation scrambles the ids, so
// codes are not sequential. Bands have different lengths, so codes from
// different bands can never collide. The whole mapping is still a bijection.
//
// Use RangeForLength to find the id range that produces a given code length.
type Banded struct {
	*base
	minLen, maxLen int
	offset         uint64
	capacity       uint64
	bands          []bandInfo
}

type bandInfo struct {
	length int
	start  uint64 // first absolute bijective index of this band
	end    uint64 // one past the last absolute bijective index
	size   uint64 // radix^length
	ml, mr uint64
	f      feistel
}

var _ Codec = (*Banded)(nil)

// NewBanded creates a codec that produces codes of length minLen..maxLen.
// The id range selects the length: ids in the first band get minLen
// characters, ids in the next band get minLen+1, and so on.
//
// maxLen must be small enough that radix^maxLen and the total space
// sum(radix^i) fit in a uint64; for the default 32-symbol alphabet that means
// maxLen <= 12.
func NewBanded(minLen, maxLen int, cfg Config) (*Banded, error) {
	if minLen < 1 {
		return nil, fmt.Errorf("%w: minLen must be >= 1", ErrConfig)
	}
	if maxLen < minLen {
		return nil, fmt.Errorf("%w: maxLen must be >= minLen", ErrConfig)
	}
	b, err := newBase(cfg, true)
	if err != nil {
		return nil, err
	}
	radix := b.radix()

	bands := make([]bandInfo, maxLen+1)
	var total uint64
	for length := 1; length <= maxLen; length++ {
		size, err := powRadix(radix, length)
		if err != nil {
			return nil, err
		}
		if total > maxUint64-size {
			return nil, fmt.Errorf("%w: code space overflows uint64", ErrConfig)
		}
		start := total
		total += size
		if length < minLen {
			continue
		}

		ml, mr, capacity, err := splitDomain(radix, length)
		if err != nil {
			return nil, err
		}
		if capacity != size {
			return nil, fmt.Errorf("%w: internal band size mismatch", ErrConfig)
		}
		bands[length] = bandInfo{
			length: length,
			start:  start,
			end:    total,
			size:   size,
			ml:     ml,
			mr:     mr,
			f:      newFeistel(b.key, b.rounds, size),
		}
	}

	offset := bands[minLen].start
	return &Banded{
		base:     b,
		minLen:   minLen,
		maxLen:   maxLen,
		offset:   offset,
		capacity: total - offset,
		bands:    bands,
	}, nil
}

// Encode maps n to a code whose length is selected by n's band.
func (c *Banded) Encode(n uint64) (string, error) {
	band, local, err := c.bandFor(n)
	if err != nil {
		return "", err
	}
	j := band.f.permute(local, band.ml, band.mr)
	return c.encodeBijective(band.start + 1 + j)
}

// Decode reverses a code without checking the issued range.
func (c *Banded) Decode(code string) (uint64, error) {
	s := c.clean(code)
	length := len(s)
	if length < c.minLen || length > c.maxLen {
		return 0, ErrInvalidCode
	}

	v, err := c.decodeBijective(s)
	if err != nil {
		return 0, err
	}
	a := v - 1
	band := &c.bands[length]
	if a < band.start || a >= band.end {
		return 0, ErrInvalidCode
	}

	i := band.f.invert(a-band.start, band.ml, band.mr)
	return band.start + i - c.offset, nil
}

// DecodeChecked reverses a code and rejects ids outside [0, IssuedMax).
func (c *Banded) DecodeChecked(code string) (uint64, error) {
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
func (c *Banded) Capacity() uint64 { return c.capacity }

// MinLen returns the shortest supported code length.
func (c *Banded) MinLen() int { return c.minLen }

// MaxLen returns the longest supported code length.
func (c *Banded) MaxLen() int { return c.maxLen }

// RangeForLength returns the inclusive id range [lo, hi] that produces codes of
// exactly the given length.
func (c *Banded) RangeForLength(length int) (lo, hi uint64, err error) {
	if length < c.minLen || length > c.maxLen {
		return 0, 0, fmt.Errorf("%w: length must be in [%d, %d]", ErrConfig, c.minLen, c.maxLen)
	}
	band := &c.bands[length]
	return band.start - c.offset, band.end - 1 - c.offset, nil
}

// LengthFor returns the code length that n maps to.
func (c *Banded) LengthFor(n uint64) (int, error) {
	band, _, err := c.bandFor(n)
	if err != nil {
		return 0, err
	}
	return band.length, nil
}

func (c *Banded) bandFor(n uint64) (*bandInfo, uint64, error) {
	if n >= c.capacity {
		return nil, 0, ErrOutOfRange
	}
	a := c.offset + n
	for length := c.minLen; length <= c.maxLen; length++ {
		band := &c.bands[length]
		if a < band.end {
			return band, a - band.start, nil
		}
	}
	return nil, 0, ErrOutOfRange
}
