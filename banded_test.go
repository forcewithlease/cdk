package cdk_test

import (
	"errors"
	"testing"

	"github.com/forcewithlease/cdk"
)

func TestBandedLengths(t *testing.T) {
	c, err := cdk.NewBanded(1, 3, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}

	seen := make(map[string]uint64, int(c.Capacity()))
	for n := uint64(0); n < c.Capacity(); n++ {
		code, err := c.Encode(n)
		if err != nil {
			t.Fatalf("Encode(%d): %v", n, err)
		}
		length, err := c.LengthFor(n)
		if err != nil {
			t.Fatalf("LengthFor(%d): %v", n, err)
		}
		if len(code) != length {
			t.Fatalf("Encode(%d) = %q, length %d, want %d", n, code, len(code), length)
		}
		lo, hi, err := c.RangeForLength(length)
		if err != nil {
			t.Fatalf("RangeForLength(%d): %v", length, err)
		}
		if n < lo || n > hi {
			t.Fatalf("n=%d outside [%d, %d] for length %d", n, lo, hi, length)
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
			t.Fatalf("round trip: got %d, want %d", got, n)
		}
	}
}

func TestBandedRanges(t *testing.T) {
	c, err := cdk.NewBanded(6, 8, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}

	want := []struct {
		length int
		lo, hi uint64
	}{
		{6, 0, 1073741823},
		{7, 1073741824, 35433480191},
		{8, 35433480192, 1134945107967},
	}
	for _, w := range want {
		lo, hi, err := c.RangeForLength(w.length)
		if err != nil {
			t.Fatalf("RangeForLength(%d): %v", w.length, err)
		}
		if lo != w.lo || hi != w.hi {
			t.Fatalf("RangeForLength(%d) = [%d, %d], want [%d, %d]", w.length, lo, hi, w.lo, w.hi)
		}
		for _, n := range []uint64{lo, lo + 1, hi - 1, hi} {
			code, err := c.Encode(n)
			if err != nil {
				t.Fatalf("Encode(%d): %v", n, err)
			}
			if len(code) != w.length {
				t.Fatalf("Encode(%d) = %q, length %d, want %d", n, code, len(code), w.length)
			}
			got, err := c.Decode(code)
			if err != nil {
				t.Fatalf("Decode(%q): %v", code, err)
			}
			if got != n {
				t.Fatalf("round trip: got %d, want %d", got, n)
			}
		}
	}

	if c.Capacity() != 1134945107968 {
		t.Fatalf("Capacity = %d, want 1134945107968", c.Capacity())
	}
	if _, err := c.Encode(c.Capacity()); !errors.Is(err, cdk.ErrOutOfRange) {
		t.Fatalf("Encode(capacity) error = %v, want ErrOutOfRange", err)
	}
}

func TestBandedFixedLength(t *testing.T) {
	c, err := cdk.NewBanded(8, 8, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	for n := uint64(0); n < 10000; n++ {
		code, err := c.Encode(n)
		if err != nil {
			t.Fatalf("Encode(%d): %v", n, err)
		}
		if len(code) != 8 {
			t.Fatalf("Encode(%d) = %q, length %d, want 8", n, code, len(code))
		}
		got, err := c.Decode(code)
		if err != nil {
			t.Fatalf("Decode(%q): %v", code, err)
		}
		if got != n {
			t.Fatalf("round trip: got %d, want %d", got, n)
		}
	}
}

func TestBandedIssuedRange(t *testing.T) {
	c, err := cdk.NewBanded(1, 2, cdk.Config{Key: []byte(testKey), IssuedMax: 10})
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

func TestBandedInvalidCodes(t *testing.T) {
	c, err := cdk.NewBanded(6, 8, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"", "00000", "000000000", "!!!!!!"} {
		if _, err := c.Decode(code); !errors.Is(err, cdk.ErrInvalidCode) {
			t.Fatalf("Decode(%q) error = %v, want ErrInvalidCode", code, err)
		}
	}
	if _, err := c.LengthFor(c.Capacity()); !errors.Is(err, cdk.ErrOutOfRange) {
		t.Fatalf("LengthFor(capacity) error = %v, want ErrOutOfRange", err)
	}
}

func TestBandedConfigErrors(t *testing.T) {
	cases := []struct {
		name string
		new  func() error
	}{
		{"zero minLen", func() error {
			_, err := cdk.NewBanded(0, 3, cdk.Config{Key: []byte(testKey)})
			return err
		}},
		{"maxLen below minLen", func() error {
			_, err := cdk.NewBanded(6, 5, cdk.Config{Key: []byte(testKey)})
			return err
		}},
		{"overflow maxLen", func() error {
			_, err := cdk.NewBanded(1, 13, cdk.Config{Key: []byte(testKey)})
			return err
		}},
		{"no key", func() error {
			_, err := cdk.NewBanded(1, 3, cdk.Config{})
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

	c, err := cdk.NewBanded(6, 8, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	for _, length := range []int{5, 9} {
		if _, _, err := c.RangeForLength(length); !errors.Is(err, cdk.ErrConfig) {
			t.Fatalf("RangeForLength(%d) error = %v, want ErrConfig", length, err)
		}
	}
}

func TestBandedMaxDefaultDomain(t *testing.T) {
	c, err := cdk.NewBanded(1, 12, cdk.Config{Key: []byte(testKey)})
	if err != nil {
		t.Fatal(err)
	}
	if c.MaxLen() != 12 {
		t.Fatalf("MaxLen = %d, want 12", c.MaxLen())
	}
	if c.Capacity() == 0 {
		t.Fatal("capacity is zero")
	}
}

func TestBandedCodecInterface(t *testing.T) {
	var _ cdk.Codec = (*cdk.Banded)(nil)
}
