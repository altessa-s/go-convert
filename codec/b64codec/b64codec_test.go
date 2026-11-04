// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package b64codec_test

import (
	"encoding/base64"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-convert/codec/b64codec"

	converter "github.com/altessa-s/go-convert"
	convcodec "github.com/altessa-s/go-convert/codec"
)

var errNext = errors.New("next called")

func failNext(string, reflect.Value, reflect.Value) error { return errNext }

type (
	blob  []byte
	token string
)

func run(t *testing.T, codec convcodec.CodecE, src any, dstPtr any) error {
	t.Helper()
	return codec("field", reflect.ValueOf(src), reflect.ValueOf(dstPtr).Elem(), failNext)
}

// payload encodes differently in every alphabet and padding variant.
var payload = []byte{0xfb, 0xff, 0xbf, 0x01}

func TestNew_Encode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		enc  *base64.Encoding
		src  any
		dst  any
		want any
	}{
		{"std", nil, payload, new(string), "+/+/AQ=="},
		{"url", base64.URLEncoding, payload, new(string), "-_-_AQ=="},
		{"raw std", base64.RawStdEncoding, payload, new(string), "+/+/AQ"},
		{"raw url", base64.RawURLEncoding, payload, new(string), "-_-_AQ"},
		{"pointer to pointer", nil, new([]byte("hi")), new(*string), new("aGk=")},
		{"named types", nil, blob("hi"), new(token), token("aGk=")},
		{"nil slice", nil, []byte(nil), new("x"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, run(t, b64codec.New(b64codec.WithEncoding(tt.enc)), tt.src, tt.dst))
			require.Equal(t, tt.want, reflect.ValueOf(tt.dst).Elem().Interface())
		})
	}
}

func TestNew_Decode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		enc  *base64.Encoding
		src  any
		dst  any
		want any
	}{
		{"std", nil, "+/+/AQ==", new([]byte), payload},
		{"url", base64.URLEncoding, "-_-_AQ==", new([]byte), payload},
		{"raw std", base64.RawStdEncoding, "+/+/AQ", new([]byte), payload},
		{"raw url", base64.RawURLEncoding, "-_-_AQ", new([]byte), payload},
		{"pointer to pointer", nil, new("aGk="), new(*[]byte), new([]byte("hi"))},
		{"named types", nil, token("aGk="), new(blob), blob("hi")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, run(t, b64codec.New(b64codec.WithEncoding(tt.enc)), tt.src, tt.dst))
			require.Equal(t, tt.want, reflect.ValueOf(tt.dst).Elem().Interface())
		})
	}
}

func TestNew_NilSource(t *testing.T) {
	t.Parallel()

	codec := b64codec.New()

	preset := new("keep")
	require.NoError(t, run(t, codec, (*[]byte)(nil), &preset))
	require.Nil(t, preset, "nil source sets a nil pointer")

	text := "keep"
	require.NoError(t, run(t, codec, (*[]byte)(nil), &text))
	require.Empty(t, text, "nil source sets the empty string")

	dst := new([]byte("old"))
	require.NoError(t, run(t, codec, (*string)(nil), &dst))
	require.Nil(t, dst, "nil source sets a nil pointer")

	data := []byte("old")
	require.NoError(t, run(t, codec, (*string)(nil), &data))
	require.Nil(t, data, "nil source sets a nil slice")
}

func TestNew_EmptyStringDecodesToEmptySlice(t *testing.T) {
	t.Parallel()

	codec := b64codec.New()

	data := []byte("old")
	require.NoError(t, run(t, codec, "", &data))
	require.NotNil(t, data)
	require.Empty(t, data)

	p := new([]byte("old"))
	require.NoError(t, run(t, codec, new(""), &p))
	require.NotNil(t, p)
	require.NotNil(t, *p)
	require.Empty(t, *p)

	var encoded string
	require.NoError(t, run(t, codec, []byte{}, &encoded))
	require.NoError(t, run(t, codec, encoded, &data))
	require.NotNil(t, data, "[]byte{} survives a round trip")
}

func TestNew_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		enc  *base64.Encoding
		src  any
		dst  any
	}{
		{"invalid character", nil, "not base64!", new([]byte)},
		{"url alphabet with std", nil, "-_-_AQ==", new([]byte("old"))},
		{"missing padding with std", nil, "aGk", new(*[]byte)},
		{"padding with raw", base64.RawStdEncoding, "aGk=", new(blob)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			before := reflect.ValueOf(tt.dst).Elem().Interface()
			err := run(t, b64codec.New(b64codec.WithEncoding(tt.enc)), tt.src, tt.dst)

			var b64Err *b64codec.Error
			require.ErrorAs(t, err, &b64Err)
			require.Equal(t, "field", b64Err.Path)

			var corrupt base64.CorruptInputError
			require.ErrorAs(t, err, &corrupt)
			require.Contains(t, err.Error(), `b64codec: field "field": illegal base64 data`)
			require.Equal(t, before, reflect.ValueOf(tt.dst).Elem().Interface(), "failed conversion leaves dst unchanged")
		})
	}
}

func TestError_RootPath(t *testing.T) {
	t.Parallel()

	err := &b64codec.Error{Err: base64.CorruptInputError(3)}
	require.Equal(t, "b64codec: illegal base64 data at input byte 3", err.Error())
}

func TestNew_PassThrough(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  any
		dst  any
	}{
		{"bytes to bytes", []byte("a"), new([]byte)},
		{"string to string", "a", new(string)},
		{"int to string", 1, new(string)},
		{"string to int slice", "a", new([]int)},
		{"bytes to interface", []byte("a"), new(any)},
		{"byte array to string", [2]byte{1, 2}, new(string)},
	}
	codec := b64codec.New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, run(t, codec, tt.src, tt.dst), errNext)
		})
	}
}

type file struct {
	Name    string
	Content []byte
	Digest  *[]byte
}

type fileDTO struct {
	Name    string
	Content string
	Digest  *string
}

func TestConvertE_RoundTrip(t *testing.T) {
	t.Parallel()

	opt := converter.WithErrorCodecs(b64codec.New(b64codec.WithEncoding(base64.RawURLEncoding)))
	src := file{Name: "a.bin", Content: payload, Digest: new([]byte{1, 2, 3})}

	dto, err := converter.ConvertE(src, &fileDTO{}, opt)
	require.NoError(t, err)
	require.Equal(t, &fileDTO{Name: "a.bin", Content: "-_-_AQ", Digest: new("AQID")}, dto)

	back, err := converter.ConvertE(dto, &file{}, opt)
	require.NoError(t, err)
	require.Equal(t, &src, back)
}

func TestConvertE_ReturnsCodecErrorUnchanged(t *testing.T) {
	t.Parallel()

	codec := b64codec.New()

	var returned error

	recording := func(path string, src, dst reflect.Value, next convcodec.CodecHandlerE) error {
		returned = codec(path, src, dst, next)
		return returned
	}

	_, err := converter.ConvertE(fileDTO{Content: "%%%"}, &file{}, converter.WithErrorCodecs(recording))
	require.Error(t, err)
	require.Same(t, returned, err)

	var b64Err *b64codec.Error
	require.ErrorAs(t, err, &b64Err)
	require.Equal(t, "content", b64Err.Path)
}
