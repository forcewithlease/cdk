package cdk_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/forcewithlease/cdk"
)

const testKey = "test-key-0123456789abcdef"

func codecs(t *testing.T) map[string]cdk.Codec {
	t.Helper()
	fixed, err := cdk.NewFixed(6, cdk.Config{Key: []byte(testKey), IssuedMax: 1 << 20})
	if err != nil {
		t.Fatalf("NewFixed: %v", err)
	}
	variable, err := cdk.NewVariable(6, cdk.Config{Key: []byte(testKey), IssuedMax: 1 << 20})
	if err != nil {
		t.Fatalf("NewVariable: %v", err)
	}
	bijective, err := cdk.NewBijective(cdk.Config{Key: []byte(testKey), IssuedMax: 1 << 20})
	if err != nil {
		t.Fatalf("NewBijective: %v", err)
	}
	return map[string]cdk.Codec{
		"fixed":     fixed,
		"variable":  variable,
		"bijective": bijective,
	}
}

func TestRoundTrip(t *testing.T) {
	values := []uint64{0, 1, 2, 31, 32, 33, 12345, (1 << 20) - 1}
	for name, c := range codecs(t) {
		t.Run(name, func(t *testing.T) {
			for _, n := range values {
				code, err := c.Encode(n)
				if err != nil {
					t.Fatalf("Encode(%d): %v", n, err)
				}
				got, err := c.DecodeChecked(code)
				if err != nil {
					t.Fatalf("DecodeChecked(%q): %v", code, err)
				}
				if got != n {
					t.Fatalf("round trip: got %d, want %d (code %q)", got, n, code)
				}
			}
		})
	}
}

func TestUniqueCodes(t *testing.T) {
	for name, c := range codecs(t) {
		t.Run(name, func(t *testing.T) {
			seen := make(map[string]uint64, 5000)
			for n := uint64(0); n < 5000; n++ {
				code, err := c.Encode(n)
				if err != nil {
					t.Fatalf("Encode(%d): %v", n, err)
				}
				if prev, ok := seen[code]; ok {
					t.Fatalf("duplicate code %q for ids %d and %d", code, prev, n)
				}
				seen[code] = n
			}
		})
	}
}

func TestFixedLength(t *testing.T) {
	c, err := cdk.NewFixed(6, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	for n := uint64(0); n < 1000; n++ {
		code, err := c.Encode(n)
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != 6 {
			t.Fatalf("Encode(%d) = %q, length %d, want 6", n, code, len(code))
		}
	}
}

func TestVariableLength(t *testing.T) {
	c, err := cdk.NewVariable(6, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	for n := uint64(0); n < 1000; n++ {
		code, err := c.Encode(n)
		if err != nil {
			t.Fatal(err)
		}
		if len(code) < 1 || len(code) > 6 {
			t.Fatalf("Encode(%d) = %q, length %d, want 1..6", n, code, len(code))
		}
	}
}

func TestBijectiveLength(t *testing.T) {
	c, err := cdk.NewBijective(cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	for n := uint64(0); n < 32; n++ {
		code, err := c.Encode(n)
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != 1 {
			t.Fatalf("Encode(%d) = %q, length %d, want 1", n, code, len(code))
		}
	}
	code, err := c.Encode(32)
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 2 {
		t.Fatalf("Encode(32) = %q, length %d, want 2", code, len(code))
	}

	// Length must grow monotonically with n.
	prev := 0
	for n := uint64(0); n < 100000; n++ {
		code, err := c.Encode(n)
		if err != nil {
			t.Fatal(err)
		}
		if len(code) < prev {
			t.Fatalf("length decreased at n=%d: %d < %d", n, len(code), prev)
		}
		prev = len(code)
	}
}

func TestNormalization(t *testing.T) {
	c, err := cdk.NewFixed(6, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	code, err := c.Encode(123)
	if err != nil {
		t.Fatal(err)
	}

	variants := []string{
		strings.ToLower(code),
		code[:3] + "-" + code[3:],
		" " + code[:2] + " " + code[2:4] + " " + code[4:] + "\n",
	}
	for _, v := range variants {
		got, err := c.Decode(v)
		if err != nil {
			t.Fatalf("Decode(%q): %v", v, err)
		}
		if got != 123 {
			t.Fatalf("Decode(%q) = %d, want 123", v, got)
		}
	}
}

func TestConfusableNormalization(t *testing.T) {
	c, err := cdk.NewFixed(6, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	var code string
	for n := uint64(0); n < 10000; n++ {
		code, err = c.Encode(n)
		if err != nil {
			t.Fatal(err)
		}
		if strings.ContainsAny(code, "01") {
			break
		}
	}
	alt := strings.NewReplacer("0", "O", "1", "I").Replace(code)
	got, err := c.Decode(alt)
	if err != nil {
		t.Fatalf("Decode(%q): %v", alt, err)
	}
	want, err := c.Decode(code)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("confusable decode = %d, want %d", got, want)
	}
}

func TestIssuedRange(t *testing.T) {
	c, err := cdk.NewFixed(4, cdk.Config{Key: []byte(testKey), IssuedMax: 10})
	if err != nil {
		t.Fatal(err)
	}
	code, err := c.Encode(10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DecodeChecked(code); !errors.Is(err, cdk.ErrNotIssued) {
		t.Fatalf("DecodeChecked(unissued) error = %v, want ErrNotIssued", err)
	}
	ok, err := c.Encode(9)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := c.DecodeChecked(ok); err != nil || n != 9 {
		t.Fatalf("DecodeChecked(issued) = %d, %v; want 9, nil", n, err)
	}
	c.SetIssuedMax(11)
	if n, err := c.DecodeChecked(code); err != nil || n != 10 {
		t.Fatalf("after SetIssuedMax: %d, %v; want 10, nil", n, err)
	}
}

func TestNoIssuedCodesByDefault(t *testing.T) {
	c, err := cdk.NewFixed(4, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	code, err := c.Encode(1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DecodeChecked(code); !errors.Is(err, cdk.ErrNotIssued) {
		t.Fatalf("DecodeChecked error = %v, want ErrNotIssued", err)
	}
}

func TestInvalidCodes(t *testing.T) {
	fixed, err := cdk.NewFixed(6, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	variable, err := cdk.NewVariable(6, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	bijective, err := cdk.NewBijective(cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		c    cdk.Codec
		code string
	}{
		{"fixed empty", fixed, ""},
		{"fixed short", fixed, "ABC"},
		{"fixed long", fixed, "ABCDEFG"},
		{"fixed bad char", fixed, "ABC!EF"},
		{"variable empty", variable, ""},
		{"variable long", variable, "0000000"},
		{"variable bad char", variable, "000!"},
		{"bijective empty", bijective, ""},
		{"bijective bad char", bijective, "!"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.c.Decode(tc.code); !errors.Is(err, cdk.ErrInvalidCode) {
				t.Fatalf("Decode(%q) error = %v, want ErrInvalidCode", tc.code, err)
			}
		})
	}
}

func TestFullDomainPermutation(t *testing.T) {
	c, err := cdk.NewFixed(2, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]uint64, int(c.Capacity()))
	for n := uint64(0); n < c.Capacity(); n++ {
		code, err := c.Encode(n)
		if err != nil {
			t.Fatalf("Encode(%d): %v", n, err)
		}
		if prev, ok := seen[code]; ok {
			t.Fatalf("duplicate code %q for ids %d and %d", code, prev, n)
		}
		seen[code] = n
		got, err := c.Decode(code)
		if err != nil {
			t.Fatalf("Decode(%q): %v", code, err)
		}
		if got != n {
			t.Fatalf("Decode(Encode(%d)) = %d", n, got)
		}
	}
	if uint64(len(seen)) != c.Capacity() {
		t.Fatalf("saw %d codes, want %d", len(seen), c.Capacity())
	}
}

func TestOddDigitDomains(t *testing.T) {
	fixed, err := cdk.NewFixed(3, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	variable, err := cdk.NewVariable(3, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	for n := uint64(0); n < 10000; n++ {
		for name, c := range map[string]cdk.Codec{"fixed": fixed, "variable": variable} {
			code, err := c.Encode(n)
			if err != nil {
				t.Fatalf("%s Encode(%d): %v", name, n, err)
			}
			got, err := c.Decode(code)
			if err != nil {
				t.Fatalf("%s Decode(%q): %v", name, code, err)
			}
			if got != n {
				t.Fatalf("%s round trip: got %d, want %d", name, got, n)
			}
		}
	}
}

func TestSingleDigitDomain(t *testing.T) {
	fixed, err := cdk.NewFixed(1, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	variable, err := cdk.NewVariable(1, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	for n := uint64(0); n < 32; n++ {
		for name, c := range map[string]cdk.Codec{"fixed": fixed, "variable": variable} {
			code, err := c.Encode(n)
			if err != nil {
				t.Fatalf("%s Encode(%d): %v", name, n, err)
			}
			if len(code) != 1 {
				t.Fatalf("%s Encode(%d) = %q, want length 1", name, n, code)
			}
			got, err := c.Decode(code)
			if err != nil {
				t.Fatalf("%s Decode(%q): %v", name, code, err)
			}
			if got != n {
				t.Fatalf("%s round trip: got %d, want %d", name, got, n)
			}
		}
	}
}

func TestCapacityBoundary(t *testing.T) {
	fixed, err := cdk.NewFixed(2, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixed.Encode(fixed.Capacity()); !errors.Is(err, cdk.ErrOutOfRange) {
		t.Fatalf("Fixed.Encode(capacity) error = %v, want ErrOutOfRange", err)
	}
	code, err := fixed.Encode(fixed.Capacity() - 1)
	if err != nil {
		t.Fatalf("Fixed.Encode(capacity-1): %v", err)
	}
	if n, err := fixed.Decode(code); err != nil || n != fixed.Capacity()-1 {
		t.Fatalf("Fixed.Decode = %d, %v; want %d", n, err, fixed.Capacity()-1)
	}

	variable, err := cdk.NewVariable(2, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := variable.Encode(variable.Capacity()); !errors.Is(err, cdk.ErrOutOfRange) {
		t.Fatalf("Variable.Encode(capacity) error = %v, want ErrOutOfRange", err)
	}
	code, err = variable.Encode(variable.Capacity() - 1)
	if err != nil {
		t.Fatalf("Variable.Encode(capacity-1): %v", err)
	}
	if n, err := variable.Decode(code); err != nil || n != variable.Capacity()-1 {
		t.Fatalf("Variable.Decode = %d, %v; want %d", n, err, variable.Capacity()-1)
	}
}

func TestBijectiveExtremes(t *testing.T) {
	c, err := cdk.NewBijective(cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Encode(^uint64(0)); !errors.Is(err, cdk.ErrOutOfRange) {
		t.Fatalf("Encode(max) error = %v, want ErrOutOfRange", err)
	}
	n := ^uint64(0) - 1
	code, err := c.Encode(n)
	if err != nil {
		t.Fatalf("Encode(max-1): %v", err)
	}
	got, err := c.Decode(code)
	if err != nil {
		t.Fatalf("Decode(%q): %v", code, err)
	}
	if got != n {
		t.Fatalf("round trip: got %d, want %d", got, n)
	}
}

func TestDeterministicAndKeyed(t *testing.T) {
	a, err := cdk.NewFixed(6, cdk.Config{Key: []byte("key-a")})
	if err != nil {
		t.Fatal(err)
	}
	b, err := cdk.NewFixed(6, cdk.Config{Key: []byte("key-b")})
	if err != nil {
		t.Fatal(err)
	}
	ca1, _ := a.Encode(1)
	ca2, _ := a.Encode(1)
	if ca1 != ca2 {
		t.Fatalf("same key produced different codes: %q and %q", ca1, ca2)
	}
	cb, _ := b.Encode(1)
	if ca1 == cb {
		t.Fatalf("different keys produced the same code %q", ca1)
	}
}

func TestCustomAlphabet(t *testing.T) {
	c, err := cdk.NewFixed(4, cdk.Config{
		Alphabet: "0123456789ABCDEF",
		Key:      []byte(testKey),
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.Capacity() != 1<<16 {
		t.Fatalf("capacity = %d, want %d", c.Capacity(), 1<<16)
	}
	for n := uint64(0); n < 10000; n++ {
		code, err := c.Encode(n)
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != 4 {
			t.Fatalf("code %q has length %d, want 4", code, len(code))
		}
		got, err := c.Decode(code)
		if err != nil {
			t.Fatal(err)
		}
		if got != n {
			t.Fatalf("round trip: got %d, want %d", got, n)
		}
	}
}

func TestConfigErrors(t *testing.T) {
	cases := []struct {
		name string
		new  func() error
	}{
		{"fixed no key", func() error {
			_, err := cdk.NewFixed(6, cdk.Config{})
			return err
		}},
		{"variable no key", func() error {
			_, err := cdk.NewVariable(6, cdk.Config{})
			return err
		}},
		{"fixed zero digits", func() error {
			_, err := cdk.NewFixed(0, cdk.Config{Key: []byte(testKey)})
			return err
		}},
		{"fixed overflow digits", func() error {
			_, err := cdk.NewFixed(13, cdk.Config{Key: []byte(testKey)})
			return err
		}},
		{"variable zero maxLen", func() error {
			_, err := cdk.NewVariable(0, cdk.Config{Key: []byte(testKey)})
			return err
		}},
		{"variable overflow maxLen", func() error {
			_, err := cdk.NewVariable(13, cdk.Config{Key: []byte(testKey)})
			return err
		}},
		{"lowercase alphabet", func() error {
			_, err := cdk.NewBijective(cdk.Config{Alphabet: "abc"})
			return err
		}},
		{"duplicate alphabet", func() error {
			_, err := cdk.NewBijective(cdk.Config{Alphabet: "AAB"})
			return err
		}},
		{"short alphabet", func() error {
			_, err := cdk.NewBijective(cdk.Config{Alphabet: "A"})
			return err
		}},
		{"too few rounds", func() error {
			_, err := cdk.NewFixed(6, cdk.Config{Key: []byte(testKey), Rounds: 3})
			return err
		}},
		{"too many rounds", func() error {
			_, err := cdk.NewFixed(6, cdk.Config{Key: []byte(testKey), Rounds: 256})
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.new(); !errors.Is(err, cdk.ErrConfig) {
				t.Fatalf("error = %v, want ErrConfig", err)
			}
		})
	}
}

func TestCodecInterface(t *testing.T) {
	var (
		_ cdk.Codec = (*cdk.Fixed)(nil)
		_ cdk.Codec = (*cdk.Variable)(nil)
		_ cdk.Codec = (*cdk.Bijective)(nil)
	)
}
