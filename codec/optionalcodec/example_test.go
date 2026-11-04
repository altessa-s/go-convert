// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package optionalcodec_test

import (
	"fmt"

	"github.com/altessa-s/go-atlas-core/types/optional"

	"github.com/altessa-s/go-convert"
	"github.com/altessa-s/go-convert/codec/optionalcodec"
)

func ExampleCodec() {
	type Project struct {
		Name        optional.Optional[string]
		Description optional.Optional[string]
	}

	type ProjectRow struct {
		Name        *string
		Description *string
	}

	opt := convert.WithCodecs(optionalcodec.Codec)

	row := convert.Convert(Project{Name: optional.Some("radar")}, &ProjectRow{}, opt)
	fmt.Println(*row.Name, row.Description == nil)

	project := convert.Convert(ProjectRow{Name: row.Name}, &Project{}, opt)
	fmt.Println(project.Name.IsSome(), project.Description.IsSome())
	// Output:
	// radar true
	// true false
}
