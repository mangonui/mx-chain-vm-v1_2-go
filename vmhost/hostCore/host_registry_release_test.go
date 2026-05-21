package hostCore

import "testing"

type releaseRecorder struct {
	calls int
}

func (rr *releaseRecorder) ReleaseHostRegistryHandle() {
	rr.calls++
}

func TestReleaseRuntimeHostRegistryHandle(t *testing.T) {
	t.Parallel()

	recorder := &releaseRecorder{}
	releaseRuntimeHostRegistryHandle(recorder)
	if recorder.calls != 1 {
		t.Fatalf("expected one release call, got %d", recorder.calls)
	}

	releaseRuntimeHostRegistryHandle(struct{}{})
	releaseRuntimeHostRegistryHandle(nil)
}
