package dashboardui

import (
	"reflect"
	"testing"

	"github.com/larsartmann/cqrs-htmx/dashboardui/v4/core"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func TestValidate_ReturnsNormalizedConfig(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	normalized, err := Config{EventSource: store}.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v; want nil", err)
	}

	if normalized.Title != defaultTitle {
		t.Errorf("Title = %q; want default %q", normalized.Title, defaultTitle)
	}

	if normalized.BasePath != defaultBasePath {
		t.Errorf("BasePath = %q; want default %q", normalized.BasePath, defaultBasePath)
	}

	if normalized.PageSize != core.DefaultPageSize() {
		t.Errorf("PageSize = %d; want %d", normalized.PageSize, core.DefaultPageSize())
	}

	if normalized.PayloadRenderer == nil {
		t.Error("PayloadRenderer should default to the DefaultPayloadRenderer")
	}
}

func TestValidate_ClampsPageSizeToCoreMaximum(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	normalized, err := Config{EventSource: store, PageSize: 100_000}.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v; want nil", err)
	}

	if normalized.PageSize != core.MaxPageSize() {
		t.Errorf("PageSize = %d; want clamped %d", normalized.PageSize, core.MaxPageSize())
	}
}

func TestValidate_RejectsMissingDataSources(t *testing.T) {
	_, err := Config{}.Validate()
	if err == nil {
		t.Fatal("Validate() with no event source/journal should fail")
	}

	if errorfamily.Classify(err) != errorfamily.Rejection {
		t.Errorf("Validate error family = %v; want Rejection; got %v", errorfamily.Classify(err), err)
	}
}

func TestValidate_RejectsInvalidAccentColor(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	_, err := Config{EventSource: store, AccentColor: "url(javascript:1)"}.Validate()
	if err == nil {
		t.Fatal("Validate() with a non-literal AccentColor should fail")
	}
}

func TestNewMatchesValidate(t *testing.T) {
	store := memorystorage.NewMemoryStore()
	config := Config{EventSource: store, PageSize: 999}

	normalized, err := config.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	d, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer d.Close()

	if !reflect.DeepEqual(d.Config(), normalized) {
		t.Errorf("New-resolved config %+v differs from Validate result %+v", d.Config(), normalized)
	}
}
