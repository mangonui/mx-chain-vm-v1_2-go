package wasmer

import (
	"fmt"
	"unsafe"
)

// MemoryError represents any kind of errors related to a WebAssembly memory. It
// is returned by `Memory` functions only.
type MemoryError struct {
	// Error message.
	message string
}

// NewMemoryError constructs a new `MemoryError`.
func NewMemoryError(message string) *MemoryError {
	return &MemoryError{message}
}

// `MemoryError` is an actual error. The `Error` function returns
// the error message.
func (error *MemoryError) Error() string {
	return error.message
}

// Memory represents an exported memory of a WebAssembly instance. To read
// and write data, please see the `Data` function.
type Memory struct {
	memory *cWasmerMemoryT
}

// Instantiates a new WebAssembly exported memory.
func newMemory(memory *cWasmerMemoryT) Memory {
	return Memory{memory}
}

// Length calculates the memory length (in bytes).
func (memory *Memory) Length() uint32 {
	if nil == memory.memory {
		return 0
	}

	return uint32(cWasmerMemoryDataLength(memory.memory))
}

// Data returns a slice of bytes over the WebAssembly memory.
//
// DEPRECATED — see issues/ISSUE-012.
//
// The returned slice is an alias over wasmer's linear memory, NOT a
// copy. After any subsequent `memory.Grow()`, wasmer may reallocate the
// backing buffer; the slice returned here then becomes a dangling
// pointer that crashes or returns wrong-but-plausible bytes when the
// caller next reads it. This is silent UAF.
//
// Existing in-tree callers in vmhost/contexts/runtime.go all
// `make + copy` the desired range BEFORE returning, so they are
// safe-by-construction in their current form (the dangle window can't
// open between Data() and the copy because no Grow runs in between).
// New callers MUST instead use [`Memory.ReadMemory`] which encapsulates
// the bounds-check + defensive-copy pattern.
//
// Also: `reflect.SliceHeader` is deprecated since Go 1.20 (use
// `unsafe.Slice`); the body is preserved here only for ABI / behaviour
// compatibility with the legacy VM. Production code paths should
// migrate to ReadMemory.
//
// nolint
func (memory *Memory) Data() []byte {
	if nil == memory.memory {
		return make([]byte, 0)
	}

	length := memory.Length()
	data := (*uint8)(cWasmerMemoryData(memory.memory))
	if data == nil || length == 0 {
		return []byte{}
	}

	// ISSUE-012 cleanup: previously this used `reflect.SliceHeader` (deprecated
	// since Go 1.20) plus a dead self-assignment that pushed lint warnings.
	// `unsafe.Slice` is the modern, lint-clean equivalent and produces the
	// same aliasing slice over wasmer linear memory. The aliasing-after-Grow
	// hazard is unchanged — that's why this method is still deprecated; only
	// the implementation got cleaner. New callers must use ReadMemory.
	return unsafe.Slice(data, length)
}

// ReadMemory returns a stable copy of the requested memory range.
// Unlike Data(), the returned slice is owned by Go and remains valid
// across subsequent memory.Grow() calls. ISSUE-012 / ISSUE-003.
func (memory *Memory) ReadMemory(offset uint32, length uint32) ([]byte, error) {
	if nil == memory.memory {
		return nil, NewMemoryError("memory not initialised")
	}
	end := uint64(offset) + uint64(length)
	totalLen := memory.Length()
	if end > uint64(totalLen) {
		return nil, NewMemoryError("memory range out of bounds")
	}
	if length == 0 {
		return []byte{}, nil
	}
	data := (*uint8)(cWasmerMemoryData(memory.memory))
	if data == nil {
		return nil, NewMemoryError("memory data pointer is nil")
	}
	copied := make([]byte, length)
	copy(copied, unsafe.Slice(data, totalLen)[offset:end])
	return copied, nil
}

// WriteMemory copies `data` into wasm linear memory starting at `offset`.
// Returns an error if the range doesn't fit in the current memory size;
// callers that need growth must invoke Grow themselves before calling
// WriteMemory (this matches the explicit per-host growth-policy pattern
// in vmhost/contexts/runtime.go::MemStore).
//
// ISSUE-012. Encapsulates the slice-fetch + bounds + copy so callers
// never hold the wasm-linear-memory alias across a possible Grow,
// closing the dangling-after-Grow UAF window that Data() exposes.
func (memory *Memory) WriteMemory(offset uint32, data []byte) error {
	if nil == memory.memory {
		return NewMemoryError("memory not initialised")
	}
	if len(data) == 0 {
		return nil
	}
	dataLen := uint32(len(data))
	end := uint64(offset) + uint64(dataLen)
	totalLen := memory.Length()
	if end > uint64(totalLen) {
		return NewMemoryError("memory range out of bounds")
	}
	dataPtr := (*uint8)(cWasmerMemoryData(memory.memory))
	if dataPtr == nil {
		return NewMemoryError("memory data pointer is nil")
	}
	copy(unsafe.Slice(dataPtr, totalLen)[offset:end], data)
	return nil
}

// Grow the memory by a number of pages (65kb each).
func (memory *Memory) Grow(numberOfPages uint32) error {
	if nil == memory.memory {
		return nil
	}

	var growResult = cWasmerMemoryGrow(memory.memory, cUint32T(numberOfPages))

	if growResult != cWasmerOk {
		var lastError, err = GetLastError()
		var errorMessage = "Failed to grow the memory:\n    %s"

		if err != nil {
			errorMessage = fmt.Sprintf(errorMessage, "(unknown details)")
		} else {
			errorMessage = fmt.Sprintf(errorMessage, lastError)
		}

		return NewMemoryError(errorMessage)
	}

	return nil
}

func (memory *Memory) Destroy() {
	if memory.memory != nil {
		cWasmerMemoryDestroy(memory.memory)
	}
}
