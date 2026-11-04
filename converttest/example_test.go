// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package converttest_test

import (
	"fmt"
	"testing"

	"github.com/altessa-s/go-convert/converttest"
)

// printTB stands in for the *testing.T of a test.
type printTB struct{ testing.TB }

func (printTB) Helper() {}

func (printTB) Fatalf(format string, args ...any) { fmt.Printf(format+"\n", args...) }

func ExampleRequireConvert() {
	type User struct {
		Name string
		Age  int
	}

	type UserDTO struct {
		Name string
		Age  int64
	}

	// In a test, pass its *testing.T.
	var t testing.TB = printTB{}

	dto := converttest.RequireConvert(t, User{Name: "Ada", Age: 36}, &UserDTO{})
	fmt.Println(dto.Name, dto.Age)
	// Output: Ada 36
}
