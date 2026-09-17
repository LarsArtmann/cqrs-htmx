package setup

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	appkit "github.com/larsartmann/go-appkit"
)

// M4 of the 2026-09-17 gap-bundle plan: Config.Metrics and Config.Version
// must reach the appkit ServiceConfig (Basic-Auth-gated /metrics, JSON
// /version), while zero values keep the route surface byte-identical (the
// appkit mux falls through to the bundle's catch-all handler).

func TestRunWithAppkit_MetricsBasicAuthAndVersion(t *testing.T) {
	bundle := MustNew(Config{
		Title:   "metrics-adoption",
		Metrics: &appkit.MetricsConfig{BasicAuthUser: "metrics", BasicAuthPass: "secret"},
		Version: "v1.2.3-test",
	})
	defer func() { _ = bundle.Close() }()

	addr := freeLocalAddr(t)
	stop := serveInBackground(t, func(ctx context.Context, addr string, h http.Handler) error {
		return bundle.runWithAppkit(ctx, addr, h, testDrainDelay, appkit.LogLevelError)
	}, addr, nil)
	defer stop()

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("http://" + addr + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics without auth: %v", err)
	}
	unauthStatus := resp.StatusCode
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	if unauthStatus != http.StatusUnauthorized {
		t.Errorf("/metrics without Basic Auth must 401, got %d", unauthStatus)
	}

	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodGet, "http://"+addr+"/metrics", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.SetBasicAuth("metrics", "secret")

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("GET /metrics with auth: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if err != nil {
		t.Fatalf("read /metrics: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/metrics with Basic Auth must 200, got %d", resp.StatusCode)
	}

	if !strings.Contains(string(body), "appkit_build_info") {
		t.Errorf("/metrics must expose the Prometheus text exposition, got: %.200s", body)
	}

	versionResp, err := client.Get("http://" + addr + "/version")
	if err != nil {
		t.Fatalf("GET /version: %v", err)
	}
	versionBody, err := io.ReadAll(versionResp.Body)
	_ = versionResp.Body.Close()

	if err != nil {
		t.Fatalf("read /version: %v", err)
	}

	if versionResp.StatusCode != http.StatusOK {
		t.Fatalf("/version must 200, got %d", versionResp.StatusCode)
	}

	var payload struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(versionBody, &payload); err != nil {
		t.Fatalf("/version must serve JSON: %v (body %.100s)", err, versionBody)
	}

	if payload.Version != "v1.2.3-test" {
		t.Errorf("/version = %q, want the Config.Version stamp", payload.Version)
	}
}

func TestRunWithAppkit_ZeroValuesMountNoMetricsOrVersionRoutes(t *testing.T) {
	bundle := MustNew(Config{Title: "zero-metrics"})
	defer func() { _ = bundle.Close() }()

	addr := freeLocalAddr(t)
	stop := serveInBackground(t, func(ctx context.Context, addr string, h http.Handler) error {
		return bundle.runWithAppkit(ctx, addr, h, testDrainDelay, appkit.LogLevelError)
	}, addr, nil)
	defer stop()

	client := &http.Client{Timeout: 10 * time.Second}

	metricsResp, err := client.Get("http://" + addr + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	metricsBody, _ := io.ReadAll(metricsResp.Body)
	_ = metricsResp.Body.Close()

	if metricsResp.Header.Get("Www-Authenticate") != "" {
		t.Error("zero Metrics must not mount an auth-challenged /metrics route")
	}

	if strings.Contains(string(metricsBody), "appkit_build_info") {
		t.Error("zero Metrics must not expose the Prometheus exposition")
	}

	versionResp, err := client.Get("http://" + addr + "/version")
	if err != nil {
		t.Fatalf("GET /version: %v", err)
	}
	versionBody, _ := io.ReadAll(versionResp.Body)
	_ = versionResp.Body.Close()

	if strings.HasPrefix(strings.TrimSpace(string(versionBody)), "{") &&
		strings.Contains(string(versionBody), `"version"`) {
		t.Error("zero Version must not mount the /version JSON endpoint")
	}
}
