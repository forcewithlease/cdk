package cdk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
)

// permuteAlphabet returns a keyed permutation of alpha. It is used by
// Bijective to make the symbol order secret without changing the code length.
func permuteAlphabet(alpha, key []byte) []byte {
	out := append([]byte(nil), alpha...)

	var block []byte
	var counter uint64
	next := func() uint64 {
		if len(block) < 8 {
			mac := hmac.New(sha256.New, key)
			mac.Write([]byte("cdk/alphabet"))
			var ctr [8]byte
			binary.BigEndian.PutUint64(ctr[:], counter)
			counter++
			mac.Write(ctr[:])
			block = mac.Sum(nil)
		}
		v := binary.BigEndian.Uint64(block[:8])
		block = block[8:]
		return v
	}

	for i := len(out) - 1; i > 0; i-- {
		j := int(next() % uint64(i+1))
		out[i], out[j] = out[j], out[i]
	}
	return out
}
