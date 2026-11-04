// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package codec_test

import (
	"fmt"
	"reflect"
	"strings"

	convcodec "github.com/altessa-s/go-convert/codec"
)

func ExampleSet_Run() {
	upper := func(field string, src, dst reflect.Value, next convcodec.CodecHandler) {
		if src.Kind() == reflect.String && dst.Kind() == reflect.String {
			dst.SetString(strings.ToUpper(src.String()))
			return
		}

		next(field, src, dst)
	}

	assign := func(_ string, src, dst reflect.Value) { dst.Set(src) }
	set := convcodec.NewCodecsSet(upper)

	var name string

	set.Run("name", reflect.ValueOf("ada"), reflect.ValueOf(&name).Elem(), assign)

	var age int

	set.Run("age", reflect.ValueOf(36), reflect.ValueOf(&age).Elem(), assign)

	fmt.Println(name, age)
	// Output: ADA 36
}

func ExampleSet_RunError() {
	required := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		if src.Kind() == reflect.String && src.String() == "" {
			return fmt.Errorf("%s is required", path)
		}

		return next(path, src, dst)
	}

	set := convcodec.NewCodecsSet().AddError(required)
	assign := func(_ string, src, dst reflect.Value) error {
		dst.Set(src)
		return nil
	}

	var city string

	fmt.Println(set.RunError("address.city", "city", reflect.ValueOf(""), reflect.ValueOf(&city).Elem(), assign))
	fmt.Println(set.RunError("address.city", "city", reflect.ValueOf("Oslo"), reflect.ValueOf(&city).Elem(), assign), city)
	// Output:
	// address.city is required
	// <nil> Oslo
}
