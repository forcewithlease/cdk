package cdk_test

import (
	"fmt"

	"github.com/forcewithlease/cdk"
)

func ExampleFixed() {
	c, err := cdk.NewFixed(10, cdk.Config{
		Key:       []byte("demo-key"),
		IssuedMax: 1_000_000,
	})
	if err != nil {
		panic(err)
	}

	code, err := c.Encode(12345)
	if err != nil {
		panic(err)
	}
	fmt.Println(code)

	n, err := c.DecodeChecked(code)
	if err != nil {
		panic(err)
	}
	fmt.Println(n)

	// Output:
	// P1KG99TKY4
	// 12345
}

func ExampleVariable() {
	c, err := cdk.NewVariable(10, cdk.Config{
		Key:       []byte("demo-key"),
		IssuedMax: 1_000_000,
	})
	if err != nil {
		panic(err)
	}

	code, err := c.Encode(12345)
	if err != nil {
		panic(err)
	}
	fmt.Println(code)

	n, err := c.DecodeChecked(code)
	if err != nil {
		panic(err)
	}
	fmt.Println(n)

	// Output:
	// N0JF88SJX4
	// 12345
}

func ExampleBijective() {
	c, err := cdk.NewBijective(cdk.Config{Key: []byte("demo-key")})
	if err != nil {
		panic(err)
	}

	for _, n := range []uint64{0, 31, 32, 12345} {
		code, err := c.Encode(n)
		if err != nil {
			panic(err)
		}
		fmt.Println(n, code, len(code))
	}

	// Output:
	// 0 E 1
	// 31 H 1
	// 32 EE 2
	// 12345 QEY 3
}

func ExampleBanded() {
	c, err := cdk.NewBanded(6, 8, cdk.Config{Key: []byte("demo-key")})
	if err != nil {
		panic(err)
	}

	for _, length := range []int{6, 7, 8} {
		lo, hi, err := c.RangeForLength(length)
		if err != nil {
			panic(err)
		}
		code, err := c.Encode(hi)
		if err != nil {
			panic(err)
		}
		n, err := c.Decode(code)
		if err != nil {
			panic(err)
		}
		fmt.Println(length, lo, hi, len(code), n == hi)
	}

	// Output:
	// 6 0 1073741823 6 true
	// 7 1073741824 35433480191 7 true
	// 8 35433480192 1134945107967 8 true
}
