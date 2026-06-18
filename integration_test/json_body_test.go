package integration_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func unmarshalJSONBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	return unmarshalJSONBodyMsg(t, w, "")
}

func unmarshalJSONBodyMsg(t *testing.T, w *httptest.ResponseRecorder, prefix string) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("%sinvalid JSON: %v", prefix, err)
	}
	return doc
}
