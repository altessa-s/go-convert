// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package durationcodec_test

import (
	"fmt"
	"time"

	"github.com/altessa-s/go-convert/codec/durationcodec"

	converter "github.com/altessa-s/go-convert"
)

func ExampleNew() {
	type Config struct {
		Timeout time.Duration
		Backoff time.Duration
	}

	type ConfigDTO struct {
		Timeout string
		Backoff int64
	}

	opt := converter.WithErrorCodecs(durationcodec.New(durationcodec.WithUnit(time.Millisecond)))

	dto, err := converter.ConvertE(Config{Timeout: 90 * time.Minute, Backoff: 1500 * time.Millisecond}, &ConfigDTO{}, opt)
	fmt.Println(dto.Timeout, dto.Backoff, err)

	cfg, err := converter.ConvertE(ConfigDTO{Timeout: "2m30s", Backoff: 250}, &Config{}, opt)
	fmt.Println(cfg.Timeout, cfg.Backoff, err)

	_, err = converter.ConvertE(ConfigDTO{Timeout: "soon"}, &Config{}, opt)
	fmt.Println(err)
	// Output:
	// 1h30m0s 1500 <nil>
	// 2m30s 250ms <nil>
	// durationcodec: field "timeout": time: invalid duration "soon"
}
