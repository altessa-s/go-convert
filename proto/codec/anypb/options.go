// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package anypb

//go:generate optgen generate

import (
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// The default resolver must satisfy Resolver.
var _ Resolver = protoregistry.GlobalTypes

// options holds configuration options for Any conversion behavior.
// defaultRecursionLimit is the nesting limit used to unmarshal payloads
// unless WithRecursionLimit sets one.
const defaultRecursionLimit = protowire.DefaultRecursionLimit

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
	// maxBytes ([WithMaxBytes]) rejects, with an [Error], Any payloads
	// larger than this many bytes before they are unmarshaled. Zero, the
	// default, sets no limit.
	maxBytes int
	// recursionLimit ([WithRecursionLimit]) limits how deeply messages may
	// nest in an unmarshaled payload; protobuf's default limit, 10000, unless
	// set. Zero also selects the default.
	recursionLimit int `optgen:"default=defaultRecursionLimit"`
}
