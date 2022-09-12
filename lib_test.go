package main

import (
	"context"
	"regexp"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

func TestBasic(t *testing.T) {
	ctx := context.Background()

	// Creates a new runtime.
	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig())
	defer r.Close(ctx)

	_, err := wasi_snapshot_preview1.Instantiate(ctx, r)
	if err != nil {
		t.Fatal(err)
	}

	// Creates a new re2 instance which has the separate sandbox.
	re2 := newRe2(ctx, r)

	// Creates a regexp instance inside Wasm.
	re := re2.mustCompile(ctx, `foo.?`)

	// Test the instance!
	if !re.match(ctx, []byte(`seafood fool`)) {
		t.Fatal("must match")
	}
	if re.match(ctx, []byte(`something else`)) {
		t.Fatal("must not match")
	}
}

// The following benchmark related codes are borrowed from the standard regexp library test:
// https://github.com/golang/go/blob/54182ff54a687272dd7632c3a963e036ce03cb7c/src/regexp/exec_test.go

var benchData = []struct {
	name, re string
	match    bool
}{
	{name: "Hard/not_match", re: "[ -~]*ABCDEFGHIJKLMNOPQRSTUVWXYZ$", match: false},
	{name: "Hard/match", re: "([ -~]*ABCDEFGHIJKLMNOPQRSTUVWXYZ$)|((?s)\\A.*\\z)", match: true},
	{name: "Hard1/not_match", re: "ABCD|CDEF|EFGH|GHIJ|IJKL|KLMN|MNOP|OPQR|QRST|STUV|UVWX|WXYZ", match: false},
	{name: "Hard1/match", re: "ABCD|CDEF|EFGH|GHIJ|IJKL|KLMN|MNOP|OPQR|QRST|STUV|UVWX|WXYZ|((?s)\\A.*\\z)", match: true},
}

var benchSizes = []struct {
	name string
	n    int
}{
	{"16B", 16},
	{"1KB", 1 << 10},
	{"1MB", 1 << 20},
	{"32MB", 32 << 20},
}

var text []byte

func makeText(n int) []byte {
	if len(text) >= n {
		return text[:n]
	}
	text = make([]byte, n)
	x := ^uint32(0)
	for i := range text {
		x += x
		x ^= 1
		if int32(x) < 0 {
			x ^= 0x88888eef
		}
		if x%31 == 0 {
			text[i] = '\n'
		} else {
			text[i] = byte(x%(0x7E+1-0x20) + 0x20)
		}
	}
	return text
}

func BenchmarkRegexpMatch(b *testing.B) {
	ctx := context.Background()

	// Creates a new runtime.
	r := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig())
	defer r.Close(ctx)

	_, err := wasi_snapshot_preview1.Instantiate(ctx, r)
	if err != nil {
		b.Fatal(err)
	}

	// Creates a new re2 instance which has the separate sandbox.
	re2 := newRe2(ctx, r)

	tests := []struct {
		name    string
		compile func(string) interface{}
		match   func(re interface{}, text []byte) bool
		close   func(interface{})
	}{
		{
			name: "wazero+libre2",
			compile: func(re string) interface{} {
				return re2.mustCompile(ctx, re)
			},
			match: func(re interface{}, text []byte) bool {
				return re.(*re2Regexp).match(ctx, text)
			},
			close: func(r interface{}) {
				r.(*re2Regexp).close(ctx)
			},
		},
		{
			name: "stdlib",
			compile: func(re string) interface{} {
				return regexp.MustCompile(re)
			},
			match: func(re interface{}, text []byte) bool {
				return re.(*regexp.Regexp).Match(text)
			},
			close: func(interface{}) {},
		},
	}

	for _, tc := range tests {
		tt := tc
		b.Run(tt.name, func(b *testing.B) {
			for _, data := range benchData {
				r := tt.compile(data.re)
				for _, size := range benchSizes {
					t := makeText(size.n)
					b.Run(data.name+"/"+size.name, func(b *testing.B) {
						b.SetBytes(int64(size.n))
						for i := 0; i < b.N; i++ {
							if actual := tt.match(r, t); actual != data.match {
								b.Fatalf("expected %v, but was %v", data.match, actual)
							}
						}
					})
				}
				tt.close(r)
			}
		})
	}
}
