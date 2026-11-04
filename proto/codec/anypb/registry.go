// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package anypb

import (
	"fmt"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Resolver resolves Any type URLs and extensions to their types. It is
// satisfied by *protoregistry.Types, such as protoregistry.GlobalTypes.
type Resolver interface {
	protoregistry.MessageTypeResolver
	protoregistry.ExtensionTypeResolver
}

// Registry maps protobuf message types to Go domain types. A registered pair
// lets the codec pack a domain value into an Any and unpack an Any into the
// domain type, converting between the message and the domain value with the
// rest of the codec chain.
//
// Each message type and each domain type may be registered once. Register
// every pair before the registry is used; a Registry is safe for concurrent
// use once registration is complete.
type Registry struct {
	byName   map[protoreflect.FullName]registration
	byDomain map[reflect.Type]registration
}

// registration is one message type and the domain type it maps to.
type registration struct {
	message protoreflect.MessageType
	domain  reflect.Type
}

// NewRegistry returns an empty Registry.
//
// Example:
//
//	registry := anypb.NewRegistry().
//	    Register(&wrapperspb.StringValue{}, reflect.TypeFor[Greeting]())
func NewRegistry() *Registry {
	return &Registry{
		byName:   make(map[protoreflect.FullName]registration),
		byDomain: make(map[reflect.Type]registration),
	}
}

// Register maps the type of message to the domain type and returns the
// Registry. A pointer domain type is registered as its element type. Register
// panics when message or domain is nil, when domain is an interface type, or
// when either type is already registered.
func (r *Registry) Register(message proto.Message, domain reflect.Type) *Registry {
	if message == nil || domain == nil {
		panic("anypb: Register called with a nil message or domain type")
	}
	for domain.Kind() == reflect.Pointer {
		domain = domain.Elem()
	}
	if domain.Kind() == reflect.Interface {
		panic(fmt.Sprintf("anypb: domain type %s is an interface", domain))
	}
	messageType := message.ProtoReflect().Type()
	name := messageType.Descriptor().FullName()
	if _, ok := r.byName[name]; ok {
		panic(fmt.Sprintf("anypb: message %s is already registered", name))
	}
	if _, ok := r.byDomain[domain]; ok {
		panic(fmt.Sprintf("anypb: domain type %s is already registered", domain))
	}
	entry := registration{message: messageType, domain: domain}
	r.byName[name] = entry
	r.byDomain[domain] = entry
	return r
}

// byMessageName returns the registration for a message full name.
func (r *Registry) byMessageName(name protoreflect.FullName) (registration, bool) {
	if r == nil {
		return registration{}, false
	}
	entry, ok := r.byName[name]
	return entry, ok
}

// byDomainType returns the registration for a domain type.
func (r *Registry) byDomainType(t reflect.Type) (registration, bool) {
	if r == nil {
		return registration{}, false
	}
	entry, ok := r.byDomain[t]
	return entry, ok
}
