// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package durpb_test

import (
	"fmt"
	"time"

	"github.com/altessa-s/go-convert"
	"github.com/altessa-s/go-convert/proto/codec/durpb"

	"google.golang.org/protobuf/types/known/durationpb"
)

func ExampleNew() {
	type Policy struct{ TTL time.Duration }

	type PolicyPB struct{ TTL *durationpb.Duration }

	pb := convert.Convert(Policy{TTL: 90 * time.Second}, &PolicyPB{}, convert.WithCodecs(durpb.New()))
	fmt.Println(pb.TTL.GetSeconds())

	policy := convert.Convert(pb, &Policy{}, convert.WithCodecs(durpb.New()))
	fmt.Println(policy.TTL)
	// Output:
	// 90
	// 1m30s
}
