// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package sqlnull_test

import (
	"database/sql"
	"fmt"

	"github.com/altessa-s/go-convert"
	"github.com/altessa-s/go-convert/codec/sqlnull"
)

func ExampleNew() {
	type Row struct {
		Name  sql.NullString
		Email sql.Null[string]
		Age   sql.NullInt64
	}

	type User struct {
		Name  string
		Email *string
		Age   *int64
	}

	user, err := convert.ConvertE(Row{
		Name:  sql.NullString{String: "Ann", Valid: true},
		Email: sql.Null[string]{V: "ann@example.com", Valid: true},
	}, &User{}, convert.WithErrorCodecs(sqlnull.New()))
	fmt.Println(user.Name, *user.Email, user.Age == nil, err)

	row, err := convert.ConvertE(User{Name: "Bob"}, &Row{}, convert.WithErrorCodecs(sqlnull.New()))
	fmt.Println(row.Name, row.Email.Valid, row.Age.Valid, err)
	// Output:
	// Ann ann@example.com true <nil>
	// {Bob true} false false <nil>
}

func ExampleWithZeroAsNull() {
	type User struct{ Age int64 }

	type UserRow struct{ Age sql.NullInt64 }

	row, err := convert.ConvertE(User{}, &UserRow{}, convert.WithErrorCodecs(sqlnull.New(sqlnull.WithZeroAsNull())))
	fmt.Println(row.Age.Valid, err)
	// Output: false <nil>
}
