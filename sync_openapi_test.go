package cqrshtmx_test

import (
	"bytes"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"encoding/json/jsontext"
	"encoding/json/v2"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/openapi"
)

var updateGoldens = flag.Bool("update", false, "rewrite golden files instead of comparing")

// goldenPath is the pinned serialized form of the sync paths added by
// AddSyncOpenAPI. Regenerate deliberately (review the diff — this IS the
// published wire contract documentation) with:
//
//	go test ./ -run TestAddSyncOpenAPI_Golden -update
const goldenPath = "testdata/sync-openapi-paths.golden.json"

// TestAddSyncOpenAPI_Golden pins the emitted spec fragment byte-for-byte:
// any drift in the sync operation metadata (params, schemas, enums, headers,
// response codes) is a consumer-visible documentation change and must be
// reviewed, not silently regenerated.
func TestAddSyncOpenAPI_Golden(t *testing.T) {
	spec := openapi.New("Golden API", "1.0.0")
	cqrshtmx.AddSyncOpenAPI(spec, "/sync/pull", "/sync/push")

	var buf bytes.Buffer
	if err := json.MarshalWrite(&buf, spec.Paths, jsontext.WithIndent("  ")); err != nil {
		t.Fatalf("marshal paths: %v", err)
	}

	got := buf.Bytes()

	if *updateGoldens {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}

		t.Logf("golden rewritten: %s (%d bytes)", goldenPath, len(got))

		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf(
			"sync OpenAPI fragment drifted; diff old/new and re-pin deliberately with -update:\n--- golden ---\n%s\n--- got ---\n%s",
			want,
			got,
		)
	}
}

// TestSyncOpenAPIOps_CarryMethodAndContract asserts the structural essentials
// (fast failure independent of the byte golden): methods, tag, and the
// statuses the handlers actually emit.
func TestSyncOpenAPIOps_CarryMethodAndContract(t *testing.T) {
	pull := cqrshtmx.SyncPullOpenAPIOp()
	if pull.HTTPMethod != "get" {
		t.Errorf("pull HTTPMethod = %q, want get", pull.HTTPMethod)
	}

	for _, status := range []int{http.StatusOK, http.StatusNotModified, http.StatusBadRequest} {
		if _, ok := pull.Responses[status]; !ok {
			t.Errorf("pull operation missing %d response", status)
		}
	}

	if pull.Responses[http.StatusOK].Headers["ETag"] == nil {
		t.Error("pull 200 response does not document the ETag header")
	}

	push := cqrshtmx.SyncPushOpenAPIOp()
	if push.HTTPMethod != "post" {
		t.Errorf("push HTTPMethod = %q, want post", push.HTTPMethod)
	}

	for _, status := range []int{http.StatusOK, http.StatusBadRequest, http.StatusServiceUnavailable} {
		if _, ok := push.Responses[status]; !ok {
			t.Errorf("push operation missing %d response", status)
		}
	}

	if push.RequestBody == nil || push.RequestBody.Required != true {
		t.Error("push operation must declare a required request body")
	}
}
