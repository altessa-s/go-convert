// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

//go:build integration

package convert_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
)

type integrationAddress struct {
	City       string
	PostalCode int
}
type integrationCustomer struct {
	ID      string
	Address *integrationAddress
	Tags    []string
}
type integrationCustomerDTO struct {
	ID      string
	Address *integrationAddress
	Tags    []string
}

func TestIntegration_NestedDomainConversion(t *testing.T) {
	t.Parallel()

	source := integrationCustomer{ID: "customer-1", Address: &integrationAddress{City: "Novosibirsk", PostalCode: 630000}, Tags: []string{"active", "verified"}}

	var destination integrationCustomerDTO

	_, err := converter.ConvertE(source, &destination, converter.WithCopySlices())
	require.NoError(t, err)
	require.Equal(t, source.ID, destination.ID)
	require.Equal(t, source.Address, destination.Address)
	require.Equal(t, source.Tags, destination.Tags)
	require.NotSame(t, &source.Tags[0], &destination.Tags[0])
}
