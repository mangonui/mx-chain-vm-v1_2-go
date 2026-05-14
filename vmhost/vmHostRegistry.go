package vmhost

import (
	"sync"
	"sync/atomic"
)

// ISSUE-013: typed handle registry for VMHost references in v1_2.
// See mx-chain-vm-v1_4-go/vmhost/vmHostRegistry.go for the full
// rationale and lifecycle note. This file is an intentional duplicate
// per design Option A — legacy VMs are intentionally frozen-by-version
// and we do not couple them to a shared cross-VM registry.

type vmHostRegistry struct {
	mu      sync.RWMutex
	nextID  uint64
	entries map[uint64]VMHost
}

var globalVMHostRegistry = &vmHostRegistry{
	entries: make(map[uint64]VMHost),
}

func (r *vmHostRegistry) Register(host VMHost) uint64 {
	id := atomic.AddUint64(&r.nextID, 1)
	r.mu.Lock()
	r.entries[id] = host
	r.mu.Unlock()
	return id
}

func (r *vmHostRegistry) Lookup(id uint64) VMHost {
	r.mu.RLock()
	h := r.entries[id]
	r.mu.RUnlock()
	return h
}

func (r *vmHostRegistry) Release(id uint64) {
	r.mu.Lock()
	delete(r.entries, id)
	r.mu.Unlock()
}

// RegisterVMHostHandle is the package-public registration helper used
// by vmhost/contexts on first SetContextData call.
func RegisterVMHostHandle(host VMHost) uint64 {
	return globalVMHostRegistry.Register(host)
}

func lookupVMHostOrPanic(handle uint64) VMHost {
	host := globalVMHostRegistry.Lookup(handle)
	if host == nil {
		panic("vmhost (v1_2): VMHost handle not found in registry; runtimeContext lifecycle bug or stale wasmer context (handle=" +
			uint64ToDecimalString(handle) + ")")
	}
	return host
}

func uint64ToDecimalString(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
