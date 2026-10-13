package errorhandling

import (
	"errors"
	"fmt"
	"strconv"
)

func ExampleBank_Transfer() {
	bank := NewBank(&Account{ID: "alice", Balance: 100}, &Account{ID: "bob", Balance: 0})

	err := bank.Transfer("alice", "carol", 10)
	fmt.Println(err)
	fmt.Println(errors.Is(err, ErrNotFound)) // looks through every %w layer

	err = bank.Transfer("alice", "bob", 500)
	fmt.Println(errors.Is(err, ErrInsufficient))

	err = bank.Transfer("alice", "bob", -5)
	var ve *ValidationError
	if errors.As(err, &ve) { // extract the custom type and read its fields
		fmt.Println("field:", ve.Field)
	}

	fmt.Println(bank.Transfer("alice", "bob", 40), bank.accounts["bob"].Balance)
	// Output:
	// transfer failed: account "carol": not found
	// true
	// true
	// field: amount
	// <nil> 40
}

func ExampleParsePort() {
	_, err := ParsePort("http")
	var numErr *strconv.NumError
	fmt.Println(err, errors.As(err, &numErr))

	_, err = ParsePort("70000")
	fmt.Println(err)

	port, err := ParsePort("8080")
	fmt.Println(port, err)
	// Output:
	// parse port: strconv.Atoi: parsing "http": invalid syntax true
	// invalid port: must be 1-65535
	// 8080 <nil>
}

func ExampleSafeDivide() {
	fmt.Println(SafeDivide(10, 2))
	fmt.Println(SafeDivide(1, 0))
	// Output:
	// 5 <nil>
	// 0 recovered: runtime error: integer divide by zero
}

func ExampleCloseAll() {
	err := CloseAll(
		func() error { return nil },
		func() error { return errors.New("db: close failed") },
		func() error { return errors.New("cache: close failed") },
	)
	fmt.Println(err)
	fmt.Println(CloseAll(func() error { return nil }) == nil)
	// Output:
	// db: close failed
	// cache: close failed
	// true
}
