// This demonstrates how to use re2 via FFI based on WebAssembly via wazero.
package main

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"log"
)

// This is the compiled Wasm binary of the patched re2:
// https://github.com/google/re2/commit/78f07ebbf92c164fcb9a5f7e13d0954a6eb01b47
//
//go:embed libre2.wasm
var libre2 []byte

// re2Instance corresponds to a re2 library Wasm instance.
type re2Instance struct {
	// new creates a new regexp instance in re2 library for a given pointer to the regexp string inside Wasm.
	new api.Function
	// del is used to delete a regexp instance in re2 library for a given pointer to the instnace created by new.
	del api.Function
	// match returns true if a given string matches the regexp instance in re2 library.
	match api.Function

	// malloc allocates a heap memory region inside Wasm memory.
	malloc api.Function
	// free releases a data region pointed by the given pointer allocated by malloc.
	free api.Function
	// memory corresponds to a memory instance of Wasm module instance.
	memory api.Memory
}

func newRe2(ctx context.Context, r wazero.Runtime) *re2Instance {
	// Stubbed main function to be used as a library (WASI reactor).
	if _, err := r.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(func(int32, int32) int32 { return 0 }).Export("__main_argc_argv").
		Instantiate(ctx, r); err != nil {
		panic(err)
	}

	// Instantiates a module instance from the re2 Wasm binary.
	mod, err := r.InstantiateModuleFromBinary(ctx, libre2)
	if err != nil {
		log.Fatal(err)
	}

	return &re2Instance{
		new:    mod.ExportedFunction("cre2_new"),
		del:    mod.ExportedFunction("cre2_delete"),
		match:  mod.ExportedFunction("cre2_match"),
		malloc: mod.ExportedFunction("malloc"),
		free:   mod.ExportedFunction("free"),
		memory: mod.Memory(),
	}
}

func (r *re2Instance) mustCompile(ctx context.Context, str string) *re2Regexp {
	// Allocate a memory for the regex str inside the Wasm memory.
	ret, err := r.malloc.Call(ctx, uint64(len(str)))
	if err != nil {
		log.Fatalf("failed to allocate wasm memory for pattern string: %v", err)
	}

	// After the allocation, write the regexp string into memory.
	if !r.memory.Write(ctx, uint32(ret[0]), []byte(str)) {
		log.Fatalf("failed to write to wasm memory at %d", uint32(ret[0]))
	}

	// Create a new regexp instance in re2 library by passing the allocated string.
	ptr, err := r.new.Call(ctx, ret[0], uint64(len(str)), 0)
	if err != nil {
		log.Fatalf("failed to compile pattern: %v", err)
	}

	// Now, we are ready to free the memory region used by the original regexp str.
	_, err = r.free.Call(ctx, ret[0])
	if err != nil {
		panic(err)
	}
	return &re2Regexp{re2Inst: r, ptr: uint32(ptr[0])}
}

// re2Regexp is like regexp.Regexp, but implemented by re2 inside Wasm module instance.
type re2Regexp struct {
	re2Inst *re2Instance
	ptr     uint32
}

func (r *re2Regexp) match(ctx context.Context, s []byte) bool {
	ret, err := r.re2Inst.malloc.Call(ctx, uint64(len(s)))
	if err != nil {
		panic(err)
	}

	if !r.re2Inst.memory.Write(ctx, uint32(ret[0]), s) {
		panic("failed to write string to wasm memory")
	}

	matched, err := r.re2Inst.match.Call(ctx, uint64(r.ptr), ret[0], uint64(len(s)), 0, uint64(len(s)), 0, 0, 0)
	if err != nil {
		panic(err)
	}

	_, err = r.re2Inst.free.Call(ctx, ret[0])
	if err != nil {
		panic(err)
	}

	return matched[0] == 1
}

func (r *re2Regexp) close(ctx context.Context) error {
	_, err := r.re2Inst.del.Call(ctx, uint64(r.ptr))
	if err != nil {
		return fmt.Errorf("failed to delete compiled pattern: %w", err)
	}
	return nil
}
