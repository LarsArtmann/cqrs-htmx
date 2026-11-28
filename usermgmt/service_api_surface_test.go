package usermgmt

import (
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// TestService_ExportedAPISurface pins the exported method set of *Service:
// one signature line per method, sorted, exactly as a consumer's reflect
// sees it. The ADR-0038 decomposition moves methods onto focused
// sub-services that *Service embeds, and promotion (never hand-written
// delegation) must keep every one of them reachable on *Service. A method
// missing here is a consumer-visible break, whatever the code inside the
// package says. Adding or changing API is deliberate: review the diff, then
// regenerate with -update.
func TestService_ExportedAPISurface(t *testing.T) {
	const golden = "testdata/service_api_surface.golden"

	got := exportedAPISurface(reflect.TypeOf(&Service{}))

	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o750); err != nil {
			t.Fatalf("create testdata directory: %v", err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o600); err != nil {
			t.Fatalf("rewrite %s: %v", golden, err)
		}
	}

	pinned, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read %s: %v", golden, err)
	}

	if string(pinned) != got {
		t.Errorf(
			"*Service exported API surface drifted from %s; review the diff, then regenerate with -update\n--- pinned\n%s--- got\n%s",
			golden,
			pinned,
			got,
		)
	}
}

func exportedAPISurface(typ reflect.Type) string {
	lines := make([]string, 0, typ.NumMethod())
	for i := range typ.NumMethod() {
		method := typ.Method(i)
		lines = append(lines, method.Name+strings.TrimPrefix(method.Type.String(), "func"))
	}
	slices.Sort(lines)
	return strings.Join(lines, "\n") + "\n"
}
