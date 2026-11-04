// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package anypb

import (
	"fmt"
	"reflect"
	"sync"

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
// every pair before passing the Registry to [WithRegistry]: [New] freezes
// it, and a later Register panics. Register is safe to call from several
// goroutines, and a frozen Registry is safe for concurrent use.
type Registry struct {
	byName   map[protoreflect.FullName]registration
	byDomain map[reflect.Type]registration
	// mu serializes Register and freeze; once frozen the maps are read
	// without locking, as nothing writes them any more.
	mu     sync.Mutex
	frozen bool
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
// panics when message or domain is nil, when domain is an interface type,
// when either type is already registered, or when the Registry was already
// passed to [New].
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

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.frozen {
		panic("anypb: Register called after the registry was passed to New; register every pair first")
	}

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

// freeze marks the Registry as in use by a codec, so that later Register
// calls panic instead of racing with conversions. A nil Registry is ignored.
func (r *Registry) freeze() {
	if r == nil {
		return
	}

	r.mu.Lock()
	r.frozen = true
	r.mu.Unlock()
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
