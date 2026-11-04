// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package convert_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	converter "github.com/altessa-s/go-convert"
)

type FuzzStruct struct {
	S string
	I int
	B bool
}

func FuzzConvert(f *testing.F) {
	f.Add("test", 123, true)
	f.Add("", 0, false)
	f.Add("long string", -1, true)

	f.Fuzz(func(t *testing.T, s string, i int, b bool) {
		src := FuzzStruct{S: s, I: i, B: b}
		var dst FuzzStruct

		// Should never panic
		converter.Convert(src, &dst)

		assert.Equal(t, s, dst.S)
		assert.Equal(t, i, dst.I)
		assert.Equal(t, b, dst.B)

	})
}

// Fuzz input limits keep every iteration fast.
const (
	fzMaxNodes = 48
	fzMaxDepth = 6
	fzMaxItems = 3
	fzMaxStr   = 6
)

type fzInner struct {
	Secret string
	Count  uint16
}

// fzNode is the source shape decoded from fuzz bytes.
type fzNode struct {
	Name  string
	Num   int64
	Flag  bool
	Score *float64
	Next  *fzNode
	Kids  []*fzNode
	Attrs map[string]*fzNode
	Tags  []string
	Dyn   any
	Inner fzInner
	Opt   *fzInner
}

type fzInnerMirror struct {
	Secret string
	Count  uint16
}

// fzNodeMirror has the shape of fzNode but distinct named types.
type fzNodeMirror struct {
	Name  string
	Num   int64
	Flag  bool
	Score *float64
	Next  *fzNodeMirror
	Kids  []*fzNodeMirror
	Attrs map[string]*fzNodeMirror
	Tags  []string
	Dyn   any
	Inner fzInnerMirror
	Opt   *fzInnerMirror
}

type fzInnerDTO struct {
	Secret string
	Count  int8
}

// fzNodeDTO is structurally similar to fzNode: narrower numbers, value
// elements in containers and a field the source does not have.
type fzNodeDTO struct {
	Name    string
	Num     int8
	Flag    bool
	Score   *float32
	Next    *fzNodeDTO
	Kids    []fzNodeDTO
	Attrs   map[string]fzNodeDTO
	Tags    []string
	Dyn     any
	Inner   fzInnerDTO
	Opt     *fzInnerDTO
	Missing string
}

// fzDecoder turns fuzz bytes into a bounded fzNode graph. Reads past the end
// of the input yield zero bytes.
type fzDecoder struct {
	data        []byte
	pos         int
	nodes       int
	allowCycles bool
	ancestors   []*fzNode
	done        []*fzNode
}

func newFzDecoder(data []byte, allowCycles bool) *fzDecoder {
	return &fzDecoder{data: data, allowCycles: allowCycles}
}

func (d *fzDecoder) byte() byte {
	if d.pos >= len(d.data) {
		return 0
	}
	b := d.data[d.pos]
	d.pos++
	return b
}

func (d *fzDecoder) string() string {
	n := int(d.byte()) % (fzMaxStr + 1)
	b := make([]byte, n)
	for i := range b {
		b[i] = d.byte()
	}
	return string(b)
}

// node decodes one node. Back edges to ancestors create cycles and are only
// produced when allowCycles is set; edges to finished nodes form a DAG.
func (d *fzDecoder) node(depth int) *fzNode {
	switch b := d.byte(); {
	case b%4 == 0 || depth >= fzMaxDepth || d.nodes >= fzMaxNodes:
		return nil
	case b%4 == 1 && d.allowCycles && len(d.ancestors) > 0:
		return d.ancestors[int(d.byte())%len(d.ancestors)]
	case b%4 == 2 && len(d.done) > 0:
		return d.done[int(d.byte())%len(d.done)]
	}
	d.nodes++
	n := &fzNode{}
	d.ancestors = append(d.ancestors, n)
	d.fill(n, depth)
	d.ancestors = d.ancestors[:len(d.ancestors)-1]
	d.done = append(d.done, n)
	return n
}

func (d *fzDecoder) fill(n *fzNode, depth int) {
	n.Name = d.string()
	n.Num = int64(int8(d.byte())) << (d.byte() % 64)
	flags := d.byte()
	n.Flag = flags&1 != 0
	if flags&2 != 0 {
		score := float64(d.byte()) / 4
		n.Score = &score
	}
	n.Inner = fzInner{Secret: d.string(), Count: uint16(d.byte()) << 4}
	if flags&4 != 0 {
		n.Opt = &fzInner{Secret: d.string(), Count: uint16(d.byte())}
	}
	if flags&8 != 0 {
		n.Tags = make([]string, int(d.byte())%(fzMaxItems+1))
		for i := range n.Tags {
			n.Tags[i] = d.string()
		}
	}
	if flags&16 != 0 {
		n.Kids = make([]*fzNode, int(d.byte())%(fzMaxItems+1))
		for i := range n.Kids {
			n.Kids[i] = d.node(depth + 1)
		}
	}
	if flags&32 != 0 {
		n.Attrs = make(map[string]*fzNode)
		for range int(d.byte()) % (fzMaxItems + 1) {
			n.Attrs[d.string()] = d.node(depth + 1)
		}
	}
	if flags&64 != 0 {
		n.Dyn = d.dynamic(depth + 1)
	}
	n.Next = d.node(depth + 1)
}

// dynamic decodes a value stored in an interface field.
func (d *fzDecoder) dynamic(depth int) any {
	if depth >= fzMaxDepth {
		return nil
	}
	switch d.byte() % 10 {
	case 1:
		return d.string()
	case 2:
		return int(int8(d.byte()))
	case 3:
		return (*fzNode)(nil)
	case 4:
		return d.node(depth)
	case 5:
		return fzInner{Secret: d.string(), Count: uint16(d.byte())}
	case 6:
		m := make(map[string]any)
		for range int(d.byte()) % (fzMaxItems + 1) {
			m[d.string()] = d.dynamic(depth + 1)
		}
		return m
	case 7:
		s := make([]any, int(d.byte())%(fzMaxItems+1))
		for i := range s {
			s[i] = d.dynamic(depth + 1)
		}
		return s
	case 8:
		m := map[string]any{"secret": d.string()}
		if d.allowCycles {
			m["self"] = m
		}
		return m
	case 9:
		return map[string]any{"name": d.string(), "inner": map[string]any{"secret": d.string()}}
	default:
		return nil
	}
}

var (
	fzIgnorePaths  = []string{"name", "next.name", "inner.secret", "kids.inner.secret", "attrs", "dyn.secret", "dyn.inner.secret", "next.next", "opt"}
	fzIgnoreFields = []string{"secret", "num", "tags", "next", "dyn", "count"}
)

// fzOptions derives an option set from the decoder's next bytes.
func fzOptions(d *fzDecoder) []converter.Option {
	flags := d.byte()
	var opts []converter.Option
	if flags&1 != 0 {
		opts = append(opts, converter.WithIgnorePaths(fzIgnorePaths[int(d.byte())%len(fzIgnorePaths)], fzIgnorePaths[int(d.byte())%len(fzIgnorePaths)]))
	}
	if flags&2 != 0 {
		opts = append(opts, converter.WithIgnoreFields(fzIgnoreFields[int(d.byte())%len(fzIgnoreFields)]))
	}
	if flags&4 != 0 {
		opts = append(opts, converter.WithSparseMerge())
	}
	if flags&8 != 0 {
		opts = append(opts, converter.WithIgnoreZeroValues())
	}
	if flags&16 != 0 {
		opts = append(opts, converter.WithCopySlices())
	}
	if flags&32 != 0 {
		opts = append(opts, converter.WithOverflowCheck())
	}
	if flags&64 != 0 {
		opts = append(opts, converter.WithMaxDepth(1+int(d.byte())%16))
	}
	if flags&128 != 0 {
		opts = append(opts, converter.WithIgnoreNilValues())
	}
	return opts
}

// requireDocumentedError fails unless err is nil or one of the exported
// error types ConvertE documents for inputs without codecs.
func requireDocumentedError(t *testing.T, err error) {
	t.Helper()
	switch err.(type) {
	case nil, converter.UnsupportedTypeError, converter.OverflowError, converter.DepthError:
	default:
		t.Fatalf("unexpected error type %T: %v", err, err)
	}
}

var fzSeeds = [][]byte{
	{},
	{0xff},
	{0x01, 0xff, 0x00},
	{3, 'a', 'b', 'c', 0x80, 7, 0xff, 9, 2, 'x', 'y', 5, 3, 1, 1, 0xff},
	{1, 3, 'n', 'a', 'm', 0x7f, 63, 0x7f, 4, 1, 0xff, 2, 's', 's', 1, 2, 't', 'u'},
	{0x05, 1, 'a', 1, 0, 0x50, 0, 0, 2, 1, 2, 0x40, 0, 0, 0, 6, 2, 1, 'k', 9, 1, 'v', 1, 'w', 1, 0},
	{0xff, 0xfe, 0xfd, 0xfc, 0xfb, 0xfa, 0xf9, 0xf8, 0xf7, 0xf6, 0xf5, 0xf4, 0xf3, 0xf2, 0xf1, 0xf0},
	{0x41, 0x05, 0x0c, 0x13, 0x1a, 0x21, 0x28, 0x2f, 0x36, 0x3d, 0x44, 0x4b, 0x52, 0x59, 0x60, 0x67, 0x6e, 0x75},
}

// FuzzConvertEGraph converts arbitrary, possibly cyclic graphs into a
// structurally similar DTO, optionally on top of a populated destination,
// under option combinations taken from the input. ConvertE must never panic
// and must only fail with the documented error types.
func FuzzConvertEGraph(f *testing.F) {
	for _, seed := range fzSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		d := newFzDecoder(data, true)
		opts := fzOptions(d)
		src := d.node(0)
		base := d.node(0)

		var dst fzNodeDTO
		if base != nil {
			// Pre-populate the destination; this may itself fail on cycles.
			_, err := converter.ConvertE(base, &dst, converter.WithMaxDepth(64))
			requireDocumentedError(t, err)
		}

		_, err := converter.ConvertE(src, &dst, opts...)
		requireDocumentedError(t, err)

		err = converter.New[*fzNode, *fzNodeMirror](opts...).ConvertE(src, &fzNodeMirror{})
		requireDocumentedError(t, err)
	})
}

// FuzzConvertERoundTrip converts acyclic graphs into an identical-shape type
// and back without options; the result must equal the source.
func FuzzConvertERoundTrip(f *testing.F) {
	for _, seed := range fzSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		src := newFzDecoder(data, false).node(0)
		if src == nil {
			return
		}

		var mirror fzNodeMirror
		_, err := converter.ConvertE(src, &mirror)
		require.NoError(t, err)

		var back fzNode
		_, err = converter.ConvertE(&mirror, &back)
		require.NoError(t, err)

		require.Equal(t, src, &back)
	})
}

// FuzzConvertEMaxDepth checks that WithMaxDepth(n) rejects a chain of more
// than n nodes, and a ring of any length, with a DepthError carrying the
// limit, and accepts shorter chains. Each node adds exactly one level.
func FuzzConvertEMaxDepth(f *testing.F) {
	f.Add(uint8(1), uint8(0), uint8(0))
	f.Add(uint8(1), uint8(2), uint8(0))
	f.Add(uint8(8), uint8(8), uint8(1))
	f.Add(uint8(8), uint8(9), uint8(2))
	f.Add(uint8(31), uint8(63), uint8(3))
	f.Add(uint8(5), uint8(3), uint8(4))
	f.Fuzz(func(t *testing.T, limit, length, mode uint8) {
		n := 1 + int(limit)%32
		size := int(length) % 64
		src := chain(size)

		opts := []converter.Option{converter.WithMaxDepth(n)}
		if mode&1 != 0 {
			// Ignore paths enable path tracking without matching any field.
			opts = append(opts, converter.WithIgnorePaths("next.missing"))
		}
		ring := mode&2 != 0 && src != nil
		if ring {
			last := src
			for last.Next != nil {
				last = last.Next
			}
			last.Next = src
		}

		_, err := converter.ConvertE(src, &listNodeDTO{}, opts...)
		requireDocumentedError(t, err)

		if ring || size > n {
			var depthErr converter.DepthError
			require.ErrorAs(t, err, &depthErr)
			require.Equal(t, n, depthErr.Limit)
			return
		}
		require.NoError(t, err)
	})
}
