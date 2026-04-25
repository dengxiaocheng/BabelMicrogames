//go:build linux && cgo

package corehost

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdint.h>
#include <stdlib.h>

typedef int (*babel_sim_step_fn)(const uint8_t* request_bytes, size_t request_len, uint8_t** response_bytes, size_t* response_len);
typedef void (*babel_sim_free_fn)(void* ptr);

static void* babel_open(const char* path) {
	dlerror();
	return dlopen(path, RTLD_NOW | RTLD_LOCAL);
}

static void* babel_lookup(void* handle, const char* symbol) {
	dlerror();
	return dlsym(handle, symbol);
}

static const char* babel_error(void) {
	return dlerror();
}

static int babel_call_step(void* fn, const uint8_t* request_bytes, size_t request_len, uint8_t** response_bytes, size_t* response_len) {
	return ((babel_sim_step_fn)fn)(request_bytes, request_len, response_bytes, response_len);
}

static void babel_call_free(void* fn, void* ptr) {
	((babel_sim_free_fn)fn)(ptr);
}

static void babel_close(void* handle) {
	if (handle != NULL) {
		dlclose(handle);
	}
}
*/
import "C"

import (
	"context"
	"fmt"
	"unsafe"
)

type dlopenTransport struct {
	handle unsafe.Pointer
	stepFn unsafe.Pointer
	freeFn unsafe.Pointer
}

func NewDlopenTransport(path string) (ByteTransport, error) {
	if path == "" {
		return nil, fmt.Errorf("missing shared library path")
	}

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	handle := C.babel_open(cpath)
	if handle == nil {
		return nil, fmt.Errorf("dlopen %s: %s", path, goDLError())
	}

	stepFn, err := lookupSymbol(handle, "babel_sim_step")
	if err != nil {
		C.babel_close(handle)
		return nil, err
	}
	freeFn, err := lookupSymbol(handle, "babel_sim_free")
	if err != nil {
		C.babel_close(handle)
		return nil, err
	}

	return &dlopenTransport{
		handle: handle,
		stepFn: stepFn,
		freeFn: freeFn,
	}, nil
}

func (t *dlopenTransport) Call(ctx context.Context, request []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var requestPtr *C.uint8_t
	if len(request) > 0 {
		requestPtr = (*C.uint8_t)(unsafe.Pointer(&request[0]))
	}

	var responsePtr *C.uint8_t
	var responseLen C.size_t
	status := C.babel_call_step(
		t.stepFn,
		requestPtr,
		C.size_t(len(request)),
		&responsePtr,
		&responseLen,
	)
	if responsePtr != nil {
		defer C.babel_call_free(t.freeFn, unsafe.Pointer(responsePtr))
	}
	if status != 0 {
		if responsePtr != nil && responseLen > 0 {
			return nil, fmt.Errorf("babel_sim_step status=%d: %s", int(status), C.GoStringN((*C.char)(unsafe.Pointer(responsePtr)), C.int(responseLen)))
		}
		return nil, fmt.Errorf("babel_sim_step status=%d", int(status))
	}
	if responsePtr == nil || responseLen == 0 {
		return nil, fmt.Errorf("babel_sim_step returned empty response")
	}
	return C.GoBytes(unsafe.Pointer(responsePtr), C.int(responseLen)), nil
}

func (t *dlopenTransport) Close() {
	if t == nil || t.handle == nil {
		return
	}
	C.babel_close(t.handle)
	t.handle = nil
}

func lookupSymbol(handle unsafe.Pointer, name string) (unsafe.Pointer, error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))

	symbol := C.babel_lookup(handle, cname)
	if symbol == nil {
		return nil, fmt.Errorf("dlsym %s: %s", name, goDLError())
	}
	return symbol, nil
}

func goDLError() string {
	if err := C.babel_error(); err != nil {
		return C.GoString(err)
	}
	return "unknown error"
}
