// Package cdk turns sequential ids into short, unguessable redemption codes.
//
// The mapping is a bijection: every id has exactly one code and every code has
// exactly one id. A generator therefore never needs a database lookup to check
// for duplicates.
//
// Four codecs are provided:
//
//	Fixed      fixed-length Base32 + Feistel permutation (scheme A)
//	Variable   Feistel permutation + bijective Base32 (scheme B)
//	Bijective  bijective Base32 only (scheme C)
//	Banded     length-banded Feistel; the id range selects the code length
//
// See README.md for the trade-offs between them.
package cdk

import (
	"errors"
	"fmt"
	"sync/atomic"
)

const (
	// DefaultAlphabet is Crockford Base32: 0-9 plus A-Z without I, L, O and U.
	DefaultAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	// DefaultRounds is the default number of Feistel rounds.
	DefaultRounds = 8

	maxUint64 = ^uint64(0)
)

var (
	// ErrConfig reports an invalid codec configuration.
	ErrConfig = errors.New("cdk: invalid configuration")
	// ErrInvalidCode reports a malformed or out-of-domain code.
	ErrInvalidCode = errors.New("cdk: invalid code")
	// ErrNotIssued reports a well-formed code whose id is outside the issued range.
	ErrNotIssued = errors.New("cdk: code is not in the issued range")
	// ErrOutOfRange reports an id or value outside the codec domain.
	ErrOutOfRange = errors.New("cdk: value out of range")
)

// Config configures a codec. The zero value is usable except for Key, which is
// required by Fixed and Variable.
type Config struct {
	// Alphabet is the symbol set used by codes. It must contain 2..64 unique
	// characters from 0-9 and A-Z. The default is DefaultAlphabet.
	Alphabet string

	// Key seeds the permutation. Fixed and Variable require a non-empty key.
	// Bijective uses it to permute the alphabet; an empty key keeps the
	// alphabet unchanged.
	Key []byte

	// Rounds is the number of Feistel rounds used by Fixed and Variable.
	// It defaults to DefaultRounds and must be between 4 and 255.
	Rounds int

	// IssuedMax is the exclusive upper bound of issued ids: a decoded id n is
	// valid when n < IssuedMax. Zero means no code is considered issued.
	IssuedMax uint64
}

// Codec is implemented by Fixed, Variable and Bijective.
type Codec interface {
	// Encode maps an id to a code.
	Encode(n uint64) (string, error)
	// Decode reverses a code without checking the issued range.
	Decode(code string) (uint64, error)
	// DecodeChecked reverses a code and rejects ids outside [0, IssuedMax).
	DecodeChecked(code string) (uint64, error)
	// Valid reports whether n is inside the issued range.
	Valid(n uint64) bool
	// IssuedMax returns the exclusive upper bound of issued ids.
	IssuedMax() uint64
	// SetIssuedMax updates the exclusive upper bound of issued ids.
	SetIssuedMax(m uint64)
	// Capacity returns the exclusive upper bound of encodable ids.
	Capacity() uint64
}

type base struct {
	alphabet  []byte
	index     [256]int16
	normalize [256]byte
	key       []byte
	rounds    int
	issued    atomic.Uint64
}

func newBase(cfg Config, requireKey bool) (*base, error) {
	alpha := cfg.Alphabet
	if alpha == "" {
		alpha = DefaultAlphabet
	}
	if len(alpha) < 2 || len(alpha) > 64 {
		return nil, fmt.Errorf("%w: alphabet must have 2..64 symbols", ErrConfig)
	}

	var seen [256]bool
	for i := 0; i < len(alpha); i++ {
		c := alpha[i]
		if !isSymbol(c) {
			return nil, fmt.Errorf("%w: invalid symbol %q", ErrConfig, c)
		}
		if seen[c] {
			return nil, fmt.Errorf("%w: duplicate symbol %q", ErrConfig, c)
		}
		seen[c] = true
	}

	if requireKey && len(cfg.Key) == 0 {
		return nil, fmt.Errorf("%w: key is required", ErrConfig)
	}

	rounds := cfg.Rounds
	if rounds == 0 {
		rounds = DefaultRounds
	}
	if rounds < 4 || rounds > 255 {
		return nil, fmt.Errorf("%w: rounds must be between 4 and 255", ErrConfig)
	}

	b := &base{
		key:    append([]byte(nil), cfg.Key...),
		rounds: rounds,
	}
	b.setAlphabet([]byte(alpha))

	for i := 0; i < 256; i++ {
		b.normalize[i] = byte(i)
	}
	for c := byte('a'); c <= 'z'; c++ {
		b.normalize[c] = c - ('a' - 'A')
	}
	if alpha == DefaultAlphabet {
		// Crockford confusable normalization: O -> 0, I/L -> 1.
		b.normalize['O'] = '0'
		b.normalize['I'] = '1'
		b.normalize['L'] = '1'
	}

	b.issued.Store(cfg.IssuedMax)
	return b, nil
}

func (b *base) setAlphabet(alpha []byte) {
	b.alphabet = append([]byte(nil), alpha...)
	for i := range b.index {
		b.index[i] = -1
	}
	for i, c := range b.alphabet {
		b.index[c] = int16(i)
	}
}

func (b *base) radix() uint64 { return uint64(len(b.alphabet)) }

// clean normalizes a human-entered code: it removes grouping separators and
// applies the alphabet's normalization table.
func (b *base) clean(code string) []byte {
	out := make([]byte, 0, len(code))
	for i := 0; i < len(code); i++ {
		c := code[i]
		switch c {
		case '-', ' ', '\t', '\r', '\n':
			continue
		}
		out = append(out, b.normalize[c])
	}
	return out
}

func (b *base) encodeDigits(x uint64, digits int) string {
	radix := b.radix()
	out := make([]byte, digits)
	for i := digits - 1; i >= 0; i-- {
		out[i] = b.alphabet[x%radix]
		x /= radix
	}
	return string(out)
}

func (b *base) decodeDigits(s []byte) (uint64, error) {
	radix := b.radix()
	var x uint64
	for _, c := range s {
		d := b.index[c]
		if d < 0 {
			return 0, ErrInvalidCode
		}
		if x > (maxUint64-uint64(d))/radix {
			return 0, ErrOutOfRange
		}
		x = x*radix + uint64(d)
	}
	return x, nil
}

// encodeBijective writes v (v >= 1) in bijective base B: digits are 1..B and
// there is no zero digit, so every non-empty string is a valid representation.
func (b *base) encodeBijective(v uint64) (string, error) {
	if v == 0 {
		return "", ErrOutOfRange
	}
	var buf [64]byte
	pos := len(buf)
	radix := b.radix()
	for v > 0 {
		v--
		if pos == 0 {
			return "", ErrOutOfRange
		}
		pos--
		buf[pos] = b.alphabet[v%radix]
		v /= radix
	}
	return string(buf[pos:]), nil
}

func (b *base) decodeBijective(s []byte) (uint64, error) {
	if len(s) == 0 {
		return 0, ErrInvalidCode
	}
	radix := b.radix()
	var v uint64
	for _, c := range s {
		d := b.index[c]
		if d < 0 {
			return 0, ErrInvalidCode
		}
		add := uint64(d) + 1
		if v > (maxUint64-add)/radix {
			return 0, ErrOutOfRange
		}
		v = v*radix + add
	}
	return v, nil
}

func (b *base) Valid(n uint64) bool   { return n < b.issued.Load() }
func (b *base) IssuedMax() uint64     { return b.issued.Load() }
func (b *base) SetIssuedMax(m uint64) { b.issued.Store(m) }

func isSymbol(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z')
}

func powRadix(radix uint64, exp int) (uint64, error) {
	if exp < 0 {
		return 0, fmt.Errorf("%w: negative exponent", ErrConfig)
	}
	out := uint64(1)
	for i := 0; i < exp; i++ {
		if out > maxUint64/radix {
			return 0, fmt.Errorf("%w: %d^%d overflows uint64", ErrConfig, radix, exp)
		}
		out *= radix
	}
	return out, nil
}

// splitDomain splits the digit space into a left and a right half. For an odd
// number of digits the right half is one digit larger.
func splitDomain(radix uint64, digits int) (ml, mr, capacity uint64, err error) {
	capacity, err = powRadix(radix, digits)
	if err != nil {
		return 0, 0, 0, err
	}
	ml, err = powRadix(radix, digits/2)
	if err != nil {
		return 0, 0, 0, err
	}
	mr = capacity / ml
	return ml, mr, capacity, nil
}
