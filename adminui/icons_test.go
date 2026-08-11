package adminui

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/templ-components/icons"
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
	iconCall := regexp.MustCompile(`@icon\("([a-z-]+)"\)`)
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
			// Unknown icon names fall back to Question in templ-components, but
			// we still want to catch typos. If the path data matches the Question
			// fallback and the name isn't "question", it's likely a typo.
			if name == "question" {
				continue
			}
			paths := icons.IconPathData(icons.Name(name))
			questionPaths := icons.IconPathData(icons.Question)
			if len(paths) == len(questionPaths) {
				match := true
				for i := range paths {
					if paths[i] != questionPaths[i] {
						match = false
						break
					}
				}
				if match {
					missing[name] = true
				}
			}
		}
	}
	for name := range missing {
		t.Errorf("icon %q is used in a .templ file but resolves to the Question fallback — likely a typo", name)
	}
}

// TestIcons_ConstantsAreKnown verifies the icon key constants all resolve to
// real icons (not the Question fallback).
func TestIcons_ConstantsAreKnown(t *testing.T) {
	t.Parallel()
	for _, name := range []string{iconDashboard, iconUsers, iconTenants, iconMembers, iconAudit} {
		paths := icons.IconPathData(icons.Name(name))
		questionPaths := icons.IconPathData(icons.Question)
		if len(paths) == len(questionPaths) && paths[0] == questionPaths[0] {
			t.Errorf("icon constant %q resolves to the Question fallback", name)
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
		got, err := usermgmt.ParseActorID(original.PrefixedString())
		if err != nil {
			t.Fatalf("ParseActorID(%q): %v", original.PrefixedString(), err)
		}
		if got.PrefixedString() != original.PrefixedString() {
			t.Errorf("round-trip %q -> %q", original.PrefixedString(), got.PrefixedString())
		}
		if got.Kind() != original.Kind() {
			t.Errorf("kind %v -> %v", got.Kind(), original.Kind())
		}
	}
}
