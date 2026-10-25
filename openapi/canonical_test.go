package openapi

import (
	"encoding/json/v2"
	"math/rand"
	"strings"
	"testing"
)

// buildMapHeavySpec returns a spec whose serialization touches every canonical
// map kind (paths, schemas, responses, content, headers, properties).
func buildMapHeavySpec() *Spec {
	item := Object(
		Prop("id", String()),
		Prop("zeta", String().WithDescription("sorted after id")),
		Prop("alpha", Integer()),
	)

	return New("Demo", "1.0.0").
		Schema("Item", item).
		Path("/zeta", Get("ListZeta").Response(200, "OK", JSON(Ref("Item")))).
		Path("/alpha", Post("CreateAlpha").
			JSONBody(Ref("Item")).
			Response(201, "Created").
			Response(409, "Conflict").
			RespHeader("X-Trace", String(), "trace id")).
		Path("/mid", Get("GetMid").
			ResponseWithContent(200, "feed", map[string]*MediaType{
				HTMLContentType:        HTML(String()),
				EventStreamContentType: EventStream(),
			}))
}

// TestSpecJSON_DeterministicAcrossInsertionOrders: encoding/json/v2 does not
// sort map keys, so before the canonical MarshalJSONTo the same spec serialized
// to different bytes per process — churning OpenAPISpecHandler's ETag on every
// restart. Now every marshal must be byte-identical.
func TestSpecJSON_DeterministicAcrossInsertionOrders(t *testing.T) {
	t.Parallel()

	first, err := buildMapHeavySpec().JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	for i := range 100 {
		// Reshuffle every map's insertion order; iteration order still varies
		// independently, so any key-order dependence shows up within a few runs.
		spec := buildMapHeavySpec()
		shuffleSpecMaps(spec, rand.New(rand.NewSource(int64(i))))

		again, err := spec.JSON()
		if err != nil {
			t.Fatalf("JSON (iter %d): %v", i, err)
		}

		if string(first) != string(again) {
			t.Fatalf("iteration %d produced different bytes:\nfirst:\n%s\nagain:\n%s", i, first, again)
		}
	}
}

func shuffleSpecMaps(s *Spec, rng *rand.Rand) {
	paths := make([]string, 0, len(s.Paths))

	for p := range s.Paths {
		paths = append(paths, p)
	}

	shuffled := map[string]*PathItem{}

	rng.Shuffle(len(paths), func(i, j int) { paths[i], paths[j] = paths[j], paths[i] })

	for _, p := range paths {
		shuffled[p] = s.Paths[p]
	}

	s.Paths = shuffled

	names := make([]string, 0, len(s.Components.Schemas))

	for n := range s.Components.Schemas {
		names = append(names, n)
	}

	schemas := map[string]*Schema{}

	rng.Shuffle(len(names), func(i, j int) { names[i], names[j] = names[j], names[i] })

	for _, n := range names {
		schemas[n] = s.Components.Schemas[n]
	}

	s.Components.Schemas = schemas
}

// TestSpecJSON_SortedOrderGolden pins the exact member order: paths sorted,
// responses ascending, content sorted, properties sorted.
func TestSpecJSON_SortedOrderGolden(t *testing.T) {
	t.Parallel()

	data, err := New("T", "1.0.0").
		Schema("Zed", String()).
		Schema("Alpha", Integer()).
		Path("/b", Get("B").Response(200, "OK")).
		Path("/a", Post("A").Response(409, "Conflict").Response(201, "Created")).
		JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	out := string(data)

	// paths: /a before /b
	if strings.Index(out, `"/a"`) > strings.Index(out, `"/b"`) {
		t.Errorf("paths not sorted:\n%s", out)
	}

	// responses: 201 before 409
	if strings.Index(out, `"201"`) > strings.Index(out, `"409"`) {
		t.Errorf("responses not ascending:\n%s", out)
	}

	// schemas: Alpha before Zed
	if strings.Index(out, `"Alpha"`) > strings.Index(out, `"Zed"`) {
		t.Errorf("schemas not sorted:\n%s", out)
	}

	// round-trips as valid JSON
	var roundTrip any
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatalf("canonical output is not valid JSON: %v", err)
	}
}

// TestSpecJSON_NestedPropertiesSorted pins the recursive case: schema
// properties sort by name regardless of construction order.
func TestSpecJSON_NestedPropertiesSorted(t *testing.T) {
	t.Parallel()

	data, err := New("T", "1.0.0").
		Schema("S", Object(
			Prop("zz", String()),
			Prop("mm", String()),
			Prop("aa", String()),
		)).
		Path("/x", Get("X").Response(200, "OK")).
		JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	out := string(data)

	idxAA, idxMM, idxZZ := strings.Index(out, `"aa"`), strings.Index(out, `"mm"`), strings.Index(out, `"zz"`)

	if !(idxAA < idxMM && idxMM < idxZZ) {
		t.Errorf("properties not sorted:\n%s", out)
	}
}
