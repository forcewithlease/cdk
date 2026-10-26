package cdk

import (
	"crypto/sha256"
	"encoding/binary"
)

// feistel is a keyed pseudo-random permutation over [0, ml*mr).
//
// Even rounds update the left half and odd rounds update the right half. For an
// even number of digits this is the classic balanced Feistel network; for an
// odd number it is an unbalanced variant that is still a bijection.
//
// The round function is HMAC-SHA256 with the key schedule expanded once at
// construction time, so encoding and decoding do not allocate.
type feistel struct {
	ipad   [sha256.BlockSize]byte
	opad   [sha256.BlockSize]byte
	rounds int
	scope  uint64
}

func newFeistel(key []byte, rounds int, scope uint64) feistel {
	var block [sha256.BlockSize]byte
	if len(key) > sha256.BlockSize {
		sum := sha256.Sum256(key)
		copy(block[:], sum[:])
	} else {
		copy(block[:], key)
	}

	f := feistel{rounds: rounds, scope: scope}
	for i := 0; i < sha256.BlockSize; i++ {
		f.ipad[i] = block[i] ^ 0x36
		f.opad[i] = block[i] ^ 0x5c
	}
	return f
}

func (f *feistel) permute(x, ml, mr uint64) uint64 {
	l, r := x/mr, x%mr
	for round := 0; round < f.rounds; round++ {
		if round%2 == 0 {
			l = addMod(l, f.roundFn(round, r, ml), ml)
		} else {
			r = addMod(r, f.roundFn(round, l, mr), mr)
		}
	}
	return l*mr + r
}

func (f *feistel) invert(x, ml, mr uint64) uint64 {
	l, r := x/mr, x%mr
	for round := f.rounds - 1; round >= 0; round-- {
		if round%2 == 0 {
			l = subMod(l, f.roundFn(round, r, ml), ml)
		} else {
			r = subMod(r, f.roundFn(round, l, mr), mr)
		}
	}
	return l*mr + r
}

func (f *feistel) roundFn(round int, x, mod uint64) uint64 {
	// inner = SHA256(ipad || round || scope || x)
	var inner [sha256.BlockSize + 17]byte
	copy(inner[:sha256.BlockSize], f.ipad[:])
	inner[sha256.BlockSize] = byte(round)
	binary.BigEndian.PutUint64(inner[sha256.BlockSize+1:], f.scope)
	binary.BigEndian.PutUint64(inner[sha256.BlockSize+9:], x)
	isum := sha256.Sum256(inner[:])

	// outer = SHA256(opad || inner)
	var outer [sha256.BlockSize + sha256.Size]byte
	copy(outer[:sha256.BlockSize], f.opad[:])
	copy(outer[sha256.BlockSize:], isum[:])
	osum := sha256.Sum256(outer[:])

	return binary.BigEndian.Uint64(osum[:8]) % mod
}

func addMod(a, b, m uint64) uint64 {
	if a >= m-b {
		return a - (m - b)
	}
	return a + b
}

func subMod(a, b, m uint64) uint64 {
	if a >= b {
		return a - b
	}
	return a + (m - b)
}
