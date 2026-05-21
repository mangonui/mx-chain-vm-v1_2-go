package contexts

import (
	"testing"

	"github.com/multiversx/mx-chain-vm-v1_2-go/vmhost"
)

type hostRegistryVMHostStub struct {
	vmhost.VMHost
}

func TestRuntimeContextReleaseHostRegistryHandle(t *testing.T) {
	t.Parallel()

	context := &runtimeContext{host: &hostRegistryVMHostStub{}}
	firstHandle := context.hostRegistryHandle()
	if firstHandle == 0 {
		t.Fatal("expected non-zero host registry handle")
	}

	context.ReleaseHostRegistryHandle()
	if context.hostHandle != 0 {
		t.Fatalf("expected hostHandle to be cleared after release, got %d", context.hostHandle)
	}

	// Must be idempotent.
	context.ReleaseHostRegistryHandle()

	secondHandle := context.hostRegistryHandle()
	if secondHandle == 0 {
		t.Fatal("expected non-zero host registry handle after re-register")
	}
	if secondHandle == firstHandle {
		t.Fatalf("expected a fresh monotonic handle after release; first=%d second=%d", firstHandle, secondHandle)
	}
}
