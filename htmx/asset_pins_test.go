package htmx_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	"github.com/larsartmann/cqrs-htmx/v4/htmx"
)

// The embedded bundles are vendored PRISTINE from the upstream npm dist
// artifacts. These pins make silent asset drift fail CI instead of shipping:
// a re-vendored bundle of the wrong version, a partial copy, or a formatter
// re-pretty-print (the pre-2.0.11 file had been prettified from the dist's
// 51,238 bytes to 72,513 before this pin existed — the exclusion that was
// supposed to prevent it arrived later than the file). When bumping a bundle,
// recompute and update its pin in the SAME change and record the source URL
// beside it.
func TestEmbeddedBundles_MatchPinnedSHA256(t *testing.T) {
	t.Parallel()

	pins := []struct {
		name string
		file string
		url  string
		sha  string
	}{
		{
			name: "default bundle",
			file: "htmx.min.js",
			url:  "https://unpkg.com/htmx.org@" + htmx.Version() + "/dist/htmx.min.js",
			sha:  "d6fdc75f204e6bdefa99b69bf1e6d4ac69b8a364f77929f45c13476b4000f717", // htmx.org@2.0.11 dist, 52182 B
		},
		{
			name: "htmx 4 bundle",
			file: "htmx4.min.js",
			url:  "https://unpkg.com/htmx.org@" + htmx.HTMX4Version + "/dist/htmx.min.js",
			sha:  "a0abb04f2f45a1f89d1d4844023245537e5e6014c87b144553e0607b4cc32130", // htmx.org@4.0.1 dist, 37448 B
		},
		{
			name: "sse extension",
			file: "extensions/sse.min.js",
			url:  "https://unpkg.com/htmx-ext-sse@2.2.4/dist/sse.min.js",
			sha:  "98a46496de0c3605fbffdce9167ba427bdd9553184f83f149c261891a92c0136", // htmx-ext-sse@2.2.4 dist, 2853 B
		},
		{
			name: "idiomorph extension",
			file: "extensions/idiomorph-ext.min.js",
			url:  "https://unpkg.com/idiomorph@0.8.0/dist/idiomorph-ext.min.js",
			sha:  "5811b9a7eda14878b7dab4378bf60432e1bb6f11fcfb970cc26540642650181e", // idiomorph@0.8.0 dist, 11037 B
		},
	}

	for _, pin := range pins {
		t.Run(pin.name, func(t *testing.T) {
			t.Parallel()

			raw, err := os.ReadFile(pin.file)
			if err != nil {
				t.Fatalf("read %s: %v", pin.file, err)
			}

			sum := sha256.Sum256(raw)
			got := hex.EncodeToString(sum[:])

			if got != pin.sha {
				t.Fatalf("embedded %s drifted:\n  got sha256 %s (%d bytes)\n  pin sha256 %s\n"+
					"  expected source: %s\n"+
					"Re-vendor the intended bytes (pristine, never formatted) and update this pin in the same change.",
					pin.file, got, len(raw), pin.sha, pin.url)
			}
		})
	}
}
