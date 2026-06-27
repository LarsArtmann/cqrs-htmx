package adminui

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v3"
)

// TestIcons_AllReferencedIconsExist scans the committed .templ sources for
// @icon("name") calls and asserts every name resolves to a real SVG. This
// catches typos that would otherwise silently render the fallback dot.
func TestIcons_AllReferencedIconsExist(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	iconCall := regexp.MustCompile(`@icon\("([a-z]+)"\)`)
	missing := map[string]bool{}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".templ") {
			continue
		}
		data, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		for _, m := range iconCall.FindAllStringSubmatch(string(data), -1) {
			name := m[1]
			if _, ok := iconPaths[name]; !ok {
				missing[name] = true
			}
		}
	}
	for name := range missing {
		t.Errorf("icon %q is used in a .templ file but not defined in the icons map", name)
	}
}

// TestIcons_ConstantsAreKnown verifies the exported-by-convention icon key
// constants all map to defined icons.
func TestIcons_ConstantsAreKnown(t *testing.T) {
	t.Parallel()
	for _, name := range []string{iconDashboard, iconUsers, iconTenants, iconMembers, iconAudit} {
		if _, ok := iconPaths[name]; !ok {
			t.Errorf("icon constant %q is not in the iconPaths map", name)
		}
	}
}

// TestParseActorID_RoundTrips verifies ActorID serialization is invertible,
// which the member remove route depends on.
func TestParseActorID_RoundTrips(t *testing.T) {
	t.Parallel()
	cases := []usermgmt.ActorID{
		usermgmt.NewActorID(usermgmt.ActorUser, "01JXEXAMPLE0001"),
		usermgmt.NewActorID(usermgmt.ActorBot, "deploy-bot"),
	}
	for _, original := range cases {
		got := usermgmt.ParseActorID(original.PrefixedString())
		if got.PrefixedString() != original.PrefixedString() {
			t.Errorf("round-trip %q -> %q", original.PrefixedString(), got.PrefixedString())
		}
		if got.Kind() != original.Kind() {
			t.Errorf("kind %v -> %v", original.Kind(), got.Kind())
		}
	}
}
