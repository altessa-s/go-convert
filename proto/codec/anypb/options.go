// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package anypb

//go:generate optgen generate

import "google.golang.org/protobuf/reflect/protoregistry"

// The default resolver must satisfy Resolver.
var _ Resolver = protoregistry.GlobalTypes

// options holds configuration options for Any conversion behavior.
type options struct {
	// ignoreNil ([WithIgnoreNil]) leaves the destination untouched when the
	// source is nil. Without it a nil source sets the destination to its
	// zero value.
	ignoreNil bool
	// registry ([WithRegistry]) maps protobuf message types to Go domain
	// types, so that an Any converts to and from a registered domain type.
	// Without a registry only protobuf messages are packed and unpacked.
	registry *Registry `opt:"Registry"`
	// resolver ([WithResolver]) resolves type URLs when an Any is unpacked
	// into an interface destination, and is passed to proto.Unmarshal for
	// nested Any and extension fields. Defaults to protoregistry.GlobalTypes.
	resolver Resolver `opt:"Resolver" optgen:"default=protoregistry.GlobalTypes"`
}
