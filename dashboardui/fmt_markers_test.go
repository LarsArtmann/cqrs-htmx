package dashboardui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// fmtStubJournal types for the command/query audit panels; empty journals
// render the panels' empty states.
type fmtStubCommandJournal struct{}

func (fmtStubCommandJournal) ReadAll(context.Context) ([]*commandPersistedCommandAlias, error) {
	return nil, nil
}
