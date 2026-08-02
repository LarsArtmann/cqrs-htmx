package datastar_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ds "github.com/larsartmann/cqrs-htmx/datastar/v4"
	"github.com/stretchr/testify/require"
)

func TestReadSignalsFromPOSTBody(t *testing.T) {
	t.Parallel()

	body := strings.NewReader(`{"datastar":{"title":"Buy milk","id":"todo-1"}}`)
	req := httptest.NewRequest(http.MethodPost, "/todos", body)
	req.Header.Set("Content-Type", "application/json")

	var s struct {
		Title string `json:"title"`
		ID    string `json:"id"`
	}

	err := ds.ReadSignals(req, &s)
	require.NoError(t, err)
	require.Equal(t, "Buy milk", s.Title)
	require.Equal(t, "todo-1", s.ID)
}

func TestReadSignalsFromGETQuery(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/todos?datastar=%7B%22title%22%3A%22Buy+milk%22%7D", nil)

	var s struct {
		Title string `json:"title"`
	}

	err := ds.ReadSignals(req, &s)
	require.NoError(t, err)
	require.Equal(t, "Buy milk", s.Title)
}

func TestReadSignalsEmptyBody(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(""))

	var s struct {
		Title string `json:"title"`
	}

	err := ds.ReadSignals(req, &s)
	require.Error(t, err)
}

func TestReadSignalsMalformedJSON(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(`{"datastar": invalid}`))
	req.Header.Set("Content-Type", "application/json")

	var s struct {
		Title string `json:"title"`
	}

	err := ds.ReadSignals(req, &s)
	require.Error(t, err)
}

func TestReadSignalsNestedStruct(t *testing.T) {
	t.Parallel()

	body := strings.NewReader(`{"datastar":{"todo":{"title":"Write tests","done":true}}}`)
	req := httptest.NewRequest(http.MethodPost, "/todos", body)
	req.Header.Set("Content-Type", "application/json")

	var s struct {
		Todo struct {
			Title string `json:"title"`
			Done  bool   `json:"done"`
		} `json:"todo"`
	}

	err := ds.ReadSignals(req, &s)
	require.NoError(t, err)
	require.Equal(t, "Write tests", s.Todo.Title)
	require.True(t, s.Todo.Done)
}

func TestReadSignalsGETEmptyQuery(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/todos", nil)

	var s struct {
		Title string `json:"title"`
	}

	err := ds.ReadSignals(req, &s)
	require.NoError(t, err)
	require.Empty(t, s.Title)
}
