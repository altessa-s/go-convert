// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package oneof

import (
	"fmt"
	"reflect"
	"testing"
)

func TestGeneratedStructCachesRemainBounded(t *testing.T) {
	const count = maxCachedFieldMappings + 100
	intType := reflect.TypeFor[int]()
	types := make([]reflect.Type, count+1)
	for i := range types {
		types[i] = reflect.StructOf([]reflect.StructField{{
			Name: fmt.Sprintf("Field%d", i),
			Type: intType,
		}})
	}

	for i := range count {
		getFieldMetadata(types[i])
		getFieldMapping(types[i], types[i+1])
	}

	structFieldCache.mu.Lock()
	metadataCount := len(structFieldCache.entries)
	structFieldCache.mu.Unlock()
	structPairCache.mu.Lock()
	mappingCount := len(structPairCache.entries)
	structPairCache.mu.Unlock()

	if metadataCount > maxCachedFieldMetadata {
		t.Fatalf("field metadata cache contains %d entries, limit is %d", metadataCount, maxCachedFieldMetadata)
	}
	if mappingCount > maxCachedFieldMappings {
		t.Fatalf("field mapping cache contains %d entries, limit is %d", mappingCount, maxCachedFieldMappings)
	}
}
