package dashboardui

import (
	"fmt"
	"testing"
	"time"
)

func TestZZDebugWorkerStates(t *testing.T) {
	host := newTestProjectionHost(t)
	ctx := t.Context()
	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = host.Stop() }()
	time.Sleep(200 * time.Millisecond)
	for _, ws := range host.Status() {
		fmt.Printf("ZZDEBUG name=%s status=%s processed=%d errors=%d\n", ws.Name, ws.Status, ws.Processed, ws.Errors)
	}
}
