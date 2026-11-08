package huma

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/httperr"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// The validation failure code is spelled independently in this module
// (validationCode) and in root's options_validate.go (an inline literal on
// the WrapRejection call). Neither side can import the other's spelling
// without exporting a new API symbol, so this test drives BOTH pipelines
// through their public surfaces and requires the emitted codes to be
// IDENTICAL: a rename on either side reddens here with a diff that names
// both spellings, instead of drifting silently until a client notices.
func TestValidationCode_NeverDriftsFromRootPipeline(t *testing.T) {
	adapterErr := validateCommandValue(mustDriftCommand(t), func(command.Command) error {
		return errDriftValidation
	})

	adapterCode := httperr.ErrorCode(adapterErr)

	if adapterCode != validationCode {
		t.Fatalf(
			"adapter leg: validateCommandValue emitted code %q, want the validationCode const %q",
			adapterCode,
			validationCode,
		)
	}

	rootCode := rootPipelineValidationCode(t)

	if rootCode != adapterCode {
		t.Fatalf(
			"VALIDATION CODE DRIFT: root's pipeline emits %q while the huma adapter emits %q — one side renamed the code without the other",
			rootCode,
			adapterCode,
		)
	}

	t.Logf("both transports emit the same validation code %q", adapterCode)
}

// errDriftValidation is the deliberately-unclassified validator error both
// legs fail with — unclassified on purpose, so the test proves the WRAPPING
// (not the error's own code) is what names the failure.
var errDriftValidation error = driftError("drift probe: invalid input")

type driftError string

func (e driftError) Error() string { return string(e) }

func mustDriftCommand(t *testing.T) command.Command {
	t.Helper()

	core, err := command.New("Drift", id.NewStreamID())
	if err != nil {
		t.Fatalf("command.New: %v", err)
	}

	return core
}

// rootPipelineValidationCode drives root's own App pipeline through a
// failing cqrshtmx.ValidateQuery and extracts the code the JSON error
// handler puts on the wire.
func rootPipelineValidationCode(t *testing.T) string {
	t.Helper()

	qryDisp := query.NewDispatcher()

	if err := qryDisp.Register("Drift", func(_ context.Context, _ query.Query) (any, error) {
		return 0, nil
	}); err != nil {
		t.Fatalf("query register: %v", err)
	}

	app := cqrshtmx.MustNew(cqrshtmx.Config{
		Queries:      qryDisp,
		ErrorHandler: httperr.JSONErrorHandler,
	})

	handler := app.Query("Drift",
		cqrshtmx.DecodeJSONQuery(func(_ struct{}) (query.Query, error) {
			return &rootDriftQuery{}, nil
		}),
		cqrshtmx.ValidateQuery(func(query.Query) error {
			return errDriftValidation
		}),
		cqrshtmx.RenderJSON[int](),
	)

	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodPost, "/drift", strings.NewReader("{}")))

	var body struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("root leg: decode JSON error body %q: %v", recorder.Body.String(), err)
	}

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("root leg: status = %d, want 400 (body %s)", recorder.Code, recorder.Body.String())
	}

	if body.Code == "" {
		t.Fatalf("root leg: no code on the wire (body %s)", recorder.Body.String())
	}

	return body.Code
}

type rootDriftQuery struct{}

func (*rootDriftQuery) Type() query.Type { return "Drift" }
