// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package oneof

//go:generate optgen generate

import (
	"reflect"

	"github.com/altessa-s/go-atlas-core/text/strings/intern"
)

// options holds configuration options for oneof conversion behavior.
// These options control validation, error handling, and custom transformations.
// The map- and name-valued fields keep hand-written setters below because
// they normalize field names and accumulate entries.
type options struct {
	// ignoreNilFields ([WithIgnoreNilFields]) skips nil fields during oneof
	// conversion: when every field is nil, the conversion passes to the next
	// handler. When disabled, nil fields still count during mapping detection.
	ignoreNilFields bool
	// strictValidation ([WithStrictValidation]) makes conversion fail when
	// more than one oneof field is non-nil, enforcing exactly-one semantics.
	strictValidation bool
	fieldConverters  map[string]FieldConverter `opt:"-" optgen:"manual"` // Custom converters for specific fields
	fallbackField    string                    `opt:"-" optgen:"manual"` // Field to use if all fields are nil
	wrapperRegistry  map[string]reflect.Type   `opt:"-" optgen:"manual"` // Registry of wrapper types for struct -> oneof conversion
}

// WithFieldConverter returns an option that registers a custom converter for a specific field.
// The converter function receives the source field value and returns the converted value.
// This allows complex transformations beyond simple type conversions.
//
// Example:
//
//	codec := oneof.New(
//	    oneof.WithFieldConverter("contractor", func(src reflect.Value) (reflect.Value, error) {
//	        // Custom transformation logic
//	        return transformedValue, nil
//	    }),
//	)
func WithFieldConverter(fieldName string, converter FieldConverter) Option {
	return func(o *options) {
		if o.fieldConverters == nil {
			o.fieldConverters = make(map[string]FieldConverter)
		}

		o.fieldConverters[intern.LowerString(fieldName)] = converter
	}
}

// WithFallbackField returns an option that specifies a fallback field to use when all fields are nil.
// This is useful when you want a default oneof variant to be set even when no data is present.
//
// Example:
//
//	codec := oneof.New(oneof.WithFallbackField("contractor"))
//	// Will create empty Contractor wrapper if all fields are nil
func WithFallbackField(fieldName string) Option {
	return func(o *options) {
		o.fallbackField = intern.LowerString(fieldName)
	}
}

// WithWrapperRegistry returns an option that registers wrapper type prototypes for oneof conversion.
// The registry maps field names to their corresponding wrapper type prototypes.
// This is required for struct -> oneof conversion to work properly.
//
// Example:
//
//	codec := oneof.New(
//	    oneof.WithWrapperRegistry(map[string]any{
//	        "Contractor": &profilespb.Invitation_Contractor{},
//	        "Agent":      &profilespb.Invitation_Agent{},
//	    }),
//	)
func WithWrapperRegistry(registry map[string]any) Option {
	return func(o *options) {
		if o.wrapperRegistry == nil {
			o.wrapperRegistry = make(map[string]reflect.Type)
		}

		for fieldName, prototype := range registry {
			protoType := reflect.TypeOf(prototype)
			if protoType.Kind() == reflect.Pointer {
				protoType = protoType.Elem()
			}

			o.wrapperRegistry[intern.LowerString(fieldName)] = protoType
		}
	}
}
