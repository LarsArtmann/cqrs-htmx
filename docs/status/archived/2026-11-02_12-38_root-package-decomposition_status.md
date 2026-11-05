# STATUS REPORT — Root-Paket-Dekomposition (go-modularize, Episode 5)

> **ANNOTATED 2026-11-02** — ~~this report~~ **SUPERSEDED** by [`2026-11-02_14-13_root-package-decomposition-go-modularize-status.md`](2026-11-02_14-13_root-package-decomposition-go-modularize-status.md), which the rebuilt report supersedes it: same episode, now carrying the rebased base, the 0-issue lint verdict, the enforced layering gate, and the pre-existing coverage-gate finding. Strike depth:
> supersession verdict (round-18 precedent for series-internal reports); nothing here is a
> unique open item. **ARCHIVED 2026-11-02 (root-package-decomposition (2026-11-02)).**

**Datum:** 2026-11-02 12:38 CET
**Branch:** `modularize/root-package-split` in einem **separaten Git-Worktree**
`/home/lars/projects/cqrs-htmx-root-split` (Basis: `fc1a0208`)
**Haupt-Worktree:** `/home/lars/projects/cqrs-htmx` — dort läuft **parallel eine
andere Session**; meine Änderungen wurden daraus wieder entfernt (siehe §5.1).
**Session-Typ:** Umbau (Code) + Messung + Selbstkritik. Keine Commits (Regel 6).
**Basis-Bewegung während der Session:** `master` ist von `fc1a0208` auf
`67c66fa2` gewandert (Permission-Maturity-Commit der parallelen Session,
+2.605 Zeilen, u. a. `authztest/`, ADR-0057/0058). Mein Branch wurde per
Fast-Forward darauf nachgezogen und **auf dem neuen Stand erneut verifiziert**:
`go build ./...` rc=0, `go test ./...` alle Pakete grün (inkl. `authztest`).
**Scope-Line.** Dieser Report behauptet nur, was in dieser Session mit einem
Befehl verifiziert wurde. Alle Zahlen sind aus dem isolierten Worktree gemessen,
nicht aus dem Hauptbaum. Was ich NICHT verifiziert habe, steht explizit in §4/§7.

---

## 0. TL;DR

1. Der Auftrag begann als `go-modularize`-Lauf. Die Messung ergab: die **29
   Go-Module** sind nicht das Problem — das **Root-Paket** (`cqrshtmx`, 57
   Produktionsdateien + 105 Testdateien in *einem* Verzeichnis) ist es. Der
   Nutzer hat genau das bestätigt („root is WAY too full with source files“),
   also wurde die Richtung von Modul- auf **Paket-Ebene** gedreht.
2. Vier neue Pakete entstanden, ohne die öffentliche API zu brechen:
   `internal/hx` (geteiltes HTTP-Fundament), `htmx`, `eventcatalog`, `httperr`.
3. Root: **57 → 50 Produktionsdateien**, **105 → 96 Testdateien**. Drei
   Facade-Dateien halten `cqrshtmx.X` für jeden Konsumenten unverändert
   kompilierbar.
4. Verifikation: `go build ./...` grün, **alle** Modul-Tests grün (506 Specs im
   Root — vorher 599; die Differenz ist mit den Paketen gewandert —, 43 in
   httperr), Modul-Coverage **91,5 %** (Gate: 90 %), und **alle
   neun Konsumenten-Module** (`usermgmt`, `adminui`, `dashboardui`, `setup`,
   `loginpage`, `datastar`, `health`, `systemadapter`, `auditlog`) bauen gegen
   die neue Struktur.
5. Der Umbau hat **einen echten Bug aufgedeckt**: nach dem Verschieben der
   Fehlerbehandlung las `httperr` einen *anderen* Context-Key für die
   Request-ID als Root schrieb — zwei Tests schlugen fehl, die Request-ID
   verschwand aus Fehlerantworten. Behoben durch einen einzigen Key in
   `internal/hx`. Genau diese Klasse („Split Brain über Paketgrenzen“) ist der
   Preis jeder Extraktion und der Grund, warum jede Stufe getestet werden muss.
6. **Größter Fehler dieser Session:** Ich habe ~10 Minuten im *Hauptbaum*
   editiert, während dort eine zweite Session schrieb (§5.1). Kein Datenverlust
   (ihre Dateien sind intakt, mein Stand wurde herausgepatcht), aber es war
   Glück, kein Können.
7. **Nicht** erledigt: Doku-Sync (AGENTS.md/README/Skill), die
   Modularisierungs-Artefakte (`docs/modularization/…`), `nix fmt`, `.#lint`,
   `.#check-modules`. **Erledigt** wurde dagegen die Tail-Budget-Regel: der
   Archiv-Pass lief, der Tail steht wieder bei 4/4 (§4.1 Punkt 10).
8. Die restlichen Kandidaten (`sync`, `readiness`, `openapi`) sind **blockiert**:
   sie hängen an `App` und dem unexportierten `handlerConfig`. Ein sauberer
   Schnitt braucht dort eine Seam-Entscheidung (v5-Material), keine Mechanik.
9. Die File-Count-Reduktion ist real, aber ehrlich betrachtet **klein**: −7
   Produktionsdateien bei +3 Facades = −4 Netto. Der strukturelle Gewinn
   (Richtung, Oberflächen, Testbesitz) ist größer als die Zahl.
10. Empfehlung in einem Satz: **diesen Stand committen, Gates laufen lassen,
    dann `sync`/`openapi`/`readiness` erst mit einem expliziten v5-Seam-Plan
    anfassen** — nicht mit Improvisation.

---

## 1. Auftrag und Auftragswechsel

### 1.1 Der ursprüngliche Auftrag

`SKILL go-modularize!` mit „READ, UNDERSTAND, RESEARCH, REFLECT … Execute and
Verify them one step at the time.“ Die Skill ist **richtungsneutral**: sie
verlangt, jede bestehende Grenze in *beide* Richtungen zu bewerten (Merge und
Split), fünf Unix-Gates pro Kandidat, und erlaubt ausdrücklich „behalten“ als
Ergebnis.

### 1.2 Was die Messung ergab (und warum der Auftrag sich drehte)

Die Vormessung im Repo (2026-10-18, `docs/modularization/2026-10-18_PROPOSAL.html`)
sagt: „boundaries are correct — zero merges, zero splits, zero reorganizes“, mit
**einem** Watch-Item: *W1 — das Root-Paket, 57 Dateien / ~8,0k LOC / 267
exportierte Symbole*. Genau dieses Watch-Item ist heute der Auftrag geworden.

Der Nutzer hat es in einem Satz bestätigt: *„I think our root project is like WAY
to full with source files!“* Ab diesem Punkt war die Richtung: **Paket-Split
innerhalb des Root-Moduls**, nicht Modul-Split.

### 1.3 Warum Paket-Split und nicht Modul-Split

| Kandidat | Unix-Gate-Ergebnis | Entscheidung |
| --- | --- | --- |
| Root-Modul in zwei Repos teilen | Gate 2 (Nutzen) verletzt: Konsumenten importieren `usermgmt` ohne `adminui` — das leistet die Modulgrenze schon; Memo 2026-10-24: KEEP ONE REPO | **nicht angefasst** |
| 13 `examples/*`-Module mergen | Gate 2 hält: jedes Demo beweist eine *andere* minimale Dep-Menge (`datastar-demo` ohne root, `system-demo` ohne root, `catalog-demo` ohne cqrs-htmx) | **nicht angefasst** |
| `usermgmt` in Subpakete zerlegen | 3 dokumentierte Trigger (ADR-0038), neu gemessen: 0/3 | **deferred bestätigt** |
| `identity-model` teilen | `E003`-Suppression dokumentiert „bewusst kohäsiv“; Alias-Architektur bricht sonst | **nicht angefasst** |
| **Root-Paket dekomponieren** | W1, 57 Dateien, 10 Cluster, keine Zyklen | **ausgeführt** (§3) |

---

## 2. Ausgangsmessung (Phase 1 + 2 der Skill)

### 2.1 Modul-Landschaft (unverändert 29)

`find . -name go.mod` (ohne `scripts/testdata/`) = **29 Module**, alle in
`go.work`, keine verwaisten `use`-Zeilen, keine absoluten `replace`-Pfade im
Modul (die absoluten Pfade stehen nur in `go.work`, version-qualifiziert bzw. für
Sibling-Repos — Version-Train-Muster).

Isolations-Gate (Skill-Gate 5, `GOWORK=off go build ./...` pro Modul): von mir
stichprobenartig gefahren; **alle** Fehlschläge der ersten Runde waren
`/mnt/buildcache`-Cache-Korruption (Gotcha 12), nicht Moduldefekte — mit frischem
`GOCACHE` war jedes geprüfte Modul grün (`setup` als härtester Kandidat
explizit). Das ist ein **wichtiger** Befund: ein rotes Gate kann hier
Infrastruktur sein, nicht Architektur.

### 2.2 Root-Paket-Komposition (vor dem Umbau)

| Metrik | Wert |
| --- | --- |
| Produktionsdateien | **57** |
| Testdateien | **105** (17.645 Zeilen Tests) |
| Produktions-LOC im Paket | 8.121 |
| exportierte Top-Level-Symbole | **270** (65 Typen, 188 Funcs, 4 Vars, 13 Consts) |
| Pakete im Root-Modul | 4 (`.` , `openapi/`, `transport/`, `cmd/cqrs-htmx-upgrade`) |
| größte Dateien | `app.go` 499, `sync_pull.go` 432, `errors.go` 400, `handler.go` 398, `response.go` 385, `context.go` 369 |

Damit liegt das Paket über **jeder** God-Package-Faustregel der Skill (15+
Dateien, >30 Symbole). Die Skill verlangt in §2.3 ausdrücklich: Cluster bilden,
dann *paketweise* schneiden — nicht modulweise.

### 2.3 Cluster-Analyse (eigenes AST-Werkzeug)

Ich habe kein Vertrauen in Bauchgefühl-Kohäsion, also habe ich ein kleines
Go-Programm gebaut (`~/.cache/modgraph/main.go`, `go/parser`, stdlib-only), das
für jede Datei die *fremden* Top-Level-Symbole bestimmt, die sie referenziert
(lokale Bezeichner werden herausgerechnet). Ergebnis (Cluster → Dateien, LOC):

| Cluster | Dateien | LOC | Symbole | Kern der Sache |
| --- | --- | --- | --- | --- |
| `core` | 33 | 4.721 | 293 | App, Options, Handler, Response, Context, Middleware, Authz, Assets |
| `errors` | 3 | 766 | 55 | MapError, Sentinels, StructuredError, Error-Handler |
| `sync` | 5 | 1.043 | 60 | ADR-0056 Frontend-Sync (Pull/Push/OpenAPI/Assets) |
| `htmx` | 5 | 513 | 68 | Header-Vokabular, Request-Kontext, Script/Extension-Serving |
| `readiness` | 3 | 406 | 25 | Readiness/Projection-Status |
| `sse` | 4 | 365 | 27 | Broadcaster, SSEEvent, Store |
| `catalog` | 2 | 174 | 10 | EventCatalog + Handler |
| `openapi` | 2 | 137 | 6 | Spec-Collector + Handler |

### 2.4 Kantenmatrix (heikelster Teil)

Gemessen auf Dateiebene (Cluster → Cluster, Anzahl referenzierender Dateien):

```
errors    -> core     9      core -> htmx      5
sync      -> core     9      htmx -> core      4
readiness -> core     7      catalog -> core   3
                             openapi -> core   3
                             sse -> core       3
core      -> sync     1      errors -> htmx    1
                             openapi -> catalog 1
                             sync -> htmx      1
```

Entscheidend: **`core` hat NULL Kanten** nach `errors`, `sse`, `catalog`,
`openapi`, `readiness`. Diese Cluster sind reine *Konsumenten* des Kerns — genau
die Richtung, die man schneiden darf. `htmx` und `sync` haben Rückkanten
(`core -> htmx` 5, `core -> sync` 1) und sind damit nur mit einem Seam zu
extrahieren.

**Ehrlich zur Methode:** Der Analyzer ist ein Ident-Heuristik-Verfahren, kein
`go/types`-Beweis; er meldet z. B. ein lokales `var buf bytes.Buffer` als
fremdes `buf`. Ich habe jeden Treffer, der eine Extraktion blockierte, am
Quelltext gegengeprüft (das hat `buf`, `serveJS`-Fehlalarme etc. entlarvt). Die
Tabelle oben ist nach dieser Bereinigung.

### 2.5 Was die Blocker tatsächlich waren

| Cluster | Blockiert durch | Bewertung |
| --- | --- | --- |
| `errors` | `writeAll`, `marshalJSON*`, `isPanicError`, `IsHTMXRequest`, `headerRedirect`, RequestID-Key | lösbar via `internal/hx` + `htmx`-Paket |
| `catalog` | `ContentTypeJSON`, `writeAll`, `hashTag`, `immutableJSON*` | lösbar via `internal/hx` |
| `htmx` | `writeAll`, `HandlerOption`/`NewResponse` (in `options_htmx.go`), `handlerConfig` | lösbar, wenn `options_htmx.go` in Root bleibt |
| `sse` | `AfterDispatchHook` (in `app.go`), `NewStructuredError` | lösbar (Typ verschieben + aliasen) |
| `readiness` | `openapi.hashTag`, `sse.Broadcaster`, `marshalJSONForResponse` | lösbar **nach** `sse` |
| `openapi` | `App`, `HandlerOption`, `handlerConfig` | **nicht** ohne Seam lösbar |
| `sync` | `App`-Methoden (`App.SyncPushHandler`), `handlerConfig`, `enrichCommandFromContext` | **nicht** ohne Seam lösbar (Go: Methoden müssen im Typ-Paket liegen) |

### 2.6 Test-Landschaft (der eigentliche „too full“-Eindruck)

| Kennzahl | Wert |
| --- | --- |
| Testdateien im Root-Verzeichnis | 105 |
| davon white-box (`package cqrshtmx`) | 24 |
| davon black-box (`package cqrshtmx_test`) | 81 |
| Präfix-Verteilung (Top) | `sse_*` 8, `example_*` 8, `coverage_*` 8, `sync_*` 7, `htmx_*` 7, `benchmark_*` 6, `testing_*` 5, `integration_*` 5 |

**Wichtig:** Go erlaubt Tests nur im Paketverzeichnis. Ein „wegräumen“ der 105
Dateien ist deshalb **kein** Umbenennen, sondern entweder (a) Mitnehmen beim
Code-Schnitt (habe ich gemacht), oder (b) ein Companion-Test-Modul — was die
Coverage-Architektur dieses Repos (Modul-Row `. = 90 %`) zerlegt. Ich habe (b)
verworfen.

---

## 3. (a) Was ich gemacht habe — chronologisch

### 3.1 Umgebung: erst gegen die Infrastruktur verloren, dann verstanden

1. `go version` im Repo-Verzeichnis: **1.26.8** (ambient), `go.work` fordert
   **1.27.2**.
2. Erste Isolationsrunde mit `GOTOOLCHAIN=go1.27.2` schlug scheinbar zufällig
   fehl: `package testing is not in std`, `cannot open file …/…-d`,
   `could not import io`. Das ist **kein** Moduldefekt, sondern
   `/mnt/buildcache`-Korruption (Gotcha 12).
3. Mit frischem `GOCACHE` (`/tmp/gocache-fresh`) war `setup` sofort grün.
   Lehre: `scripts/lib/go-cache-env.sh` hätte das von Anfang an verhindert.
4. Ab dann: `GOEXPERIMENT=jsonv2 GOTOOLCHAIN=go1.27.2
   GOCACHE=$HOME/.cache/go-build-baseline` für jeden Lauf.
   Baseline vor dem Umbau: `BUILD OK`, `go test ./...` grün (Root 23,6 s).

### 3.2 Der Kollisionsvorfall (siehe §5.1 für die Selbstkritik)

Mitten in den ersten Edits zeigten gopls-Diagnosen Fehler in Dateien, die ich
nie angefasst hatte (`authz_contract.go`, `options_types.go`,
`sse_broadcaster.go`, `setup/setup.go`, `identity-model/*`). `ls -l` zeigte
Änderungszeiten **Sekunden** vor meiner Abfrage: eine **zweite Session** schrieb
aktiv im selben Arbeitsverzeichnis.

Meine Reaktion:

1. `git worktree add /home/lars/projects/cqrs-htmx-root-split HEAD` → isolierter
   Baum, eigener Branch.
2. Meine Änderungen als Patch der **eigenen** Pfade exportiert, im Worktree
   angewendet, dort verifiziert.
3. Den Hauptbaum für **meine** Pfade auf `HEAD` zurückgesetzt
   (`git restore --source=HEAD --staged --worktree -- <meine Dateien>`), die
   Dateien der anderen Session **nicht** angefasst.
4. Ergebnis: fremder Baum baut wieder, mein Stand ist isoliert und reproduzierbar.

### 3.3 Schritt 1 — `internal/hx` (geteiltes Fundament)

**Problem:** Ein Subpaket darf das Root-Paket nicht importieren, sobald Root es
re-exportiert (Zyklus). Die von mehreren Clustern genutzten Primitiven brauchten
deshalb ein Zuhause *unterhalb* beider.

**Neu:** `internal/hx` (7 Dateien, `package hx`):

| Datei | Inhalt |
| --- | --- |
| `doc.go` | Warum das Paket existiert (Zyklus-Vermeidung, interne Sichtbarkeit) |
| `constants.go` | `ContentType{Plain,HTML,JSON,Problem}`, `JSONKey{Error,Status,Code,RequestID}` |
| `response.go` | `WriteAll`, `WriteAllString`, `MarshalJSONForResponse`, `MarshalJSONOrFallback`, `HashTag` |
| `immutable.go` | `JSONSerializer`, `SerializeToImmutableHandler`, `NewImmutableJSONHandler`, `ServeImmutableJSON`, `ServeJS` |
| `context.go` | Request-ID-Context-Key, `WithRequestID`, `RequestIDFromContext`, `NewRequestID`, `Parse/MustParseRequestID` |
| `panic.go` | `PanicCode`, `IsPanicError` |
| `response_test.go`, `immutable_test.go` | 96,2 % bzw. deckende Tests (siehe §3.7) |

**Root-Änderungen:** `safe_write.go` gelöscht; 38 Aufrufstellen auf `hx.*`
umgestellt; `constants.go` re-exportiert jetzt `= hx.…` (identische Bytes);
`hashTag`, `serveJS`, `immutableJSONServer` etc. aus Root entfernt (je eine
Implementierung, keine Kopie).

**Verifikation:** `go build ./...` grün, `go test ./...` grün (23,6 s),
Modul-Coverage 92,3 %.

### 3.4 Schritt 2 — `cqrshtmx/htmx`

**Verschoben (git mv, Historie bleibt):**

| alt | neu |
| --- | --- |
| `htmx.go` | `htmx/htmx.go` |
| `htmx_embed.go` | `htmx/embed.go` |
| `htmx_extensions.go` | `htmx/extensions.go` |
| `htmx_serve.go` | `htmx/serve.go` |
| `htmx.min.js` | `htmx/htmx.min.js` |
| `extensions/sse.min.js` | `htmx/extensions/sse.min.js` |
| `extensions/idiomorph-ext.min.js` | `htmx/extensions/idiomorph-ext.min.js` |
| `htmx_{core,swap,extensions,serve}_test.go` | `htmx/{core,swap,extensions,serve}_test.go` |

**Was im Root blieb:** `options_htmx.go` (hängt an `handlerConfig`), sowie
`htmx_context_test.go` (testet `HTMXMiddleware`, das in Root lebt),
`htmx_notify_test.go`, `htmx_response_test.go` (Root-Verhalten).

**API-Erweiterung im Subpaket:** Die HTMX-Header-Namen (`Hx-Redirect`,
`Hx-Push-Url`, …) waren unexportiert und wurden von `response.go`/`errors.go`
gebraucht. Sie sind jetzt **alle** als `htmx.Header*` exportiert — das
Wire-Vokabular ist damit öffentlich und ehrlich, statt dass Root Strings
dupliziert. Ebenso `ParseHTMXRequest`.

**Root-Facade:** `htmx_facade.go` — 2 Typ-Aliase, 15 Const-Aliase, 20
Funktions-Wrapper (jeweils mit eigenem Doc-Kommentar). `cqrshtmx.HTMXRequest`
ist **derselbe** Typ wie `htmx.HTMXRequest` (Alias, keine Kopie).

**Test-Bootstrap:** `htmx/suite_test.go` (`TestHTMX` + `RunSpecs`), weil die
verschobenen Specs Ginkgo nutzen und Root-Bootstrap nicht erreichbar ist.

### 3.5 Schritt 3 — `cqrshtmx/eventcatalog`

Der sauberste Kandidat: **kein** anderes Root-Produktionsfile referenzierte den
Katalog.

| alt | neu |
| --- | --- |
| `event_catalog.go` | `eventcatalog/catalog.go` |
| `event_catalog_handler.go` | `eventcatalog/handler.go` |
| `event_catalog_test.go` | `eventcatalog/catalog_test.go` |
| `event_catalog_handler_test.go` | `eventcatalog/handler_test.go` |

`eventcatalog_facade.go` aliasiert `EventCatalog`, `EventMetadata`,
`PayloadField` und wrappt `NewEventCatalog`/`EventCatalogHandler`. Ein
redundanter `runConditionalGetSpec`-Aufruf (der Helper lebt in Root) wurde
entfernt — die 304/200-Pfade sind im selben File bereits explizit gepinnt.

### 3.6 Schritt 4 — `cqrshtmx/httperr`

Der größte und invasivste Schnitt (3 Dateien, 766 LOC, 52 `MapError`-Treffer
repo-weit, davon die meisten in Tests über die Facade).

| alt | neu |
| --- | --- |
| `errors.go` | `httperr/errors.go` |
| `errors_status.go` | `httperr/status.go` |
| `structured_error.go` | `httperr/structured.go` |
| `errors_test.go` | `httperr/errors_test.go` |
| `structured_error_test.go` | `httperr/structured_test.go` |
| `carrier_status_internal_test.go` | `httperr/carrier_internal_test.go` |
| 2 Wrapper-Tests aus `uncovered_helpers_test.go` | `httperr/wrappers_internal_test.go` |

**Exportiert, weil Root sie braucht** (bewusst, mit Doc-Kommentaren):
`DefaultLoginRedirect`, `DefaultRetryAfter`, `HandleErrorCore`,
`PlainBodyWriter`, `PlainBodyOptions` (Felder jetzt `IncludeInternal` /
`IncludeRequestID` statt unexportiert), sowie die vier Verdrahtungs-Sentinels
`ErrCommandsNil`, `ErrQueriesNil`, `ErrDecoderMissing`, `ErrDecoderReturnedNil`.

**Bewusst *nicht* exportiert:** Die Sentinel-Werte werden in Root über
**unexportierte Aliase** (`var errCommandsNil = httperr.ErrCommandsNil`)
weitergereicht. Damit bleibt Roots Aufrufstellen-Code unverändert, die
öffentliche Oberfläche wächst nicht, und es existiert **kein** zweiter Sentinel
mit demselben Code (die klassische Fehler-Falle der Skill, FM#7).

**Bug, den dieser Schritt aufdeckte:** Root schrieb die Request-ID unter einen
eigenen `contextKey[RequestID]`, `httperr` las sie unter dem `hx`-Key. Zwei
Tests (`recovery_test.go:157`, `app_test.go:81`) fielen sofort um — genau der
gewünschte Effekt. Fix: `context.go` delegiert jetzt an `hx.WithRequestID` /
`hx.RequestIDFromContext`; es gibt **einen** Key.

### 3.7 Verifikation (letzter Stand, alles im isolierten Worktree)

```
go build ./...                                       → rc=0
go test ./... -count=1 -coverprofile=…               → alles grün
  cqrs-htmx/v4                    23,5 s   93,9 %
  …/cmd/cqrs-htmx-upgrade                  83,0 %
  …/eventcatalog                           93,8 %
  …/htmx                                   77,5 %
  …/httperr                                89,2 %
  …/internal/hx                            87,7 %
  …/openapi                                94,9 %
  …/transport                              93,2 %
  total                                    91,5 %   (Gate: 90 %)
```

Konsumenten-Module, je `go build ./...` **einzeln** geprüft:
`usermgmt`, `adminui`, `dashboardui`, `setup`, `loginpage`, `datastar`,
`health`, `systemadapter`, `auditlog` → **alle OK**. Das ist der Beweis, dass
die Facades die veröffentlichte Oberfläche wirklich halten.

**Stand der Dateizahlen:**

| | vorher | nachher |
| --- | --- | --- |
| Root-Produktionsdateien | 57 | **50** (47 Kern + 3 Facades) |
| Root-Testdateien | 105 | **96** |
| Root-LOC (Paket) | 8.121 | **7.020** (Rest in Subpaketen) |
| Pakete im Root-Modul | 4 | **8** |
| Go-Module | 29 | 29 (**unverändert**) |

---

## 4. (b) Was ich vergessen habe

### 4.1 Doku, Gates, Artefakte — vollständig unangetastet

Die Skill verlangt Phase 5–7: Ausführungsplan, Doku-Update, Abschlussreflexion
*im Repo*. Davon ist **nichts** geschrieben. Konkret fehlen:

1. **`AGENTS.md`** — die Architektur-Sektion beschreibt Root noch als „HTTP
   handler builder, HTMX/SSE helpers, authz, error mapping, pagination, openapi,
   transport, sync“. Neu: `htmx`, `eventcatalog`, `httperr`, `internal/hx`
   müssen dort stehen, inklusive Richtungsregel (Root → Subpakete, nie umgekehrt).
2. **`CURRENT_FOCUS.md`** — die aktive Thread-Beschreibung ist vom 31.10. und
   kennt diesen Umbau nicht.
3. **`docs/modularization/`** — es gibt keinen Episode-5-Report. Die Skill will
   `docs/modularization/<date>_PROPOSAL.html` + `_EXECUTION_PLAN.html`; ich habe
   stattdessen inline gearbeitet (Begründung: der Nutzer wollte den Schnitt, nicht
   ein weiteres Dokument). Das ist eine Regelabweichung, keine Ausrede.
4. **Projekt-Skill `.agents/skills/cqrs-htmx/SKILL.md`** — die Modultabelle nennt
   Root-Inhalte; die stimmen weitgehend noch (alles bleibt importierbar), aber
   „wohin importiere ich neu?“ ist nicht dokumentiert.
5. **`README.md` / `docs/guides/*`** — dieselbe Frage, plus die
   `check-readme-splitbrain`/`check-skill-claims`-Gates, die Symbolansprüche
   prüfen. Sie sollten grün sein (Aliase), aber **ich habe sie nicht laufen
   lassen**.
6. **`nix fmt`** — die neuen Dateien sind `goimports`-formatiert, aber
   `gofumpt`/`golines`/`treefmt`-kanonisch ist ungeprüft. Lange Doc-Kommentare
   könnten `golines`-Grenzen reißen.
7. **`nix run .#lint`** — nie gelaufen. Risiko: `revive` (Doc-Kommentare auf
   jeder exportierten Alias-Datei — habe ich geschrieben), `gochecknoglobals`
   (viele `var`-Aliase in `httperr_facade.go`), `dupl` (drei Facade-Dateien mit
   ähnlicher Struktur), `exhaustruct_v5` (Struct-Literale in `app.go`).
8. **`nix run .#check-modules`** — ich habe es gestartet (Background), aber die
   Ausgabe ging beim Wechsel des Worktrees verloren. **Nicht** verifiziert.
9. **Coverage-Gate-Lauf** (`nix run .#coverage-gate`) — ich habe die Schwelle
   manuell nachgerechnet (91,5 % > 90 %), aber nicht über das Gate.
10. **Tail-Budget — gefunden, nicht ignoriert:** nach diesem Report standen
    **5** Live-Reports gegen `TAIL_BASELINE=4` (`check-docs-tail-budget
    --ratchet` rc=1). Ich habe den dokumentierten Archiv-Pass **ausgeführt**
    (`scripts/tools/archive-status.sh --apply --update-baseline`, round-44
    close-out → round-45 als Nachfolger). Ergebnis: Tail **4**, Baseline bleibt
    4, Gate grün, zwei Nachbar-Links automatisch re-based (das Werkzeug hat die
    Reihenfolge eingehalten: Row-Gate → `git mv` → Annotation → Link-Rebase →
    Manifest → Baseline).

### 4.2 Code-Review-Lücken

11. **`options_htmx.go`** ist jetzt der einzige htmx-bezogene Rest in Root. Ich
    habe **nicht** geprüft, ob er inhaltlich noch zu `htmx` gehört und nur wegen
    `handlerConfig` dort bleibt — oder ob man ihn mit einem kleinen Seam
    (`WithHTMXOptions`-Interface) sauber verschieben könnte.
12. **Unexportierte Aliase** (`errCommandsNil` etc.) sind ein bewusstes
    Zugeständnis. Ich habe **nicht** geprüft, ob `golangci-lint` sie als
    „unbenutzt“ oder „shadowing“ anmeckert.
13. **`htmx`-Coverage 77,5 %** ist der schwächste Punkt. Ursache: die
    verschobenen Specs deckten vorher auch Root-Codepfade; jetzt fehlen
    Extension-Fehlerpfade (Panic-Zweige in `HTMXExtensionHandler`). Nicht
    repariert.
14. **Keine Benchmarks** für die neuen Pakete — die alten `benchmark_*_test.go`
    sind in Root geblieben und treffen über die Facade. Das ist funktional okay,
    aber die Zuordnung „Benchmark liegt beim Code“ ist verletzt.
15. **`go mod tidy`** in den Modulen: nicht gelaufen. Es sollte nichts ändern
    (keine neuen externen Deps in Root; die Subpakete nutzen dieselben), aber
    „sollte“ ist nicht „ist“.

### 4.3 Prozess-Lücken

16. **Kein Commit.** Regel 6 („never commit unless asked“) hat Vorrang, aber
    damit liegt der gesamte Umbau **uncommitted** in einem Worktree. Ein
    `git worktree prune` oder ein unbedachtes Aufräumen kann ihn kosten.
17. **Kein Branch-Naming nach Repo-Konvention** geprüft (es gibt
    `modularize/<slug>`-Form, passt).
18. **Ich habe den Nutzer nicht gefragt**, ob der Umbau überhaupt in *diesem*
    Worktree landen soll, während eine andere Session im Hauptbaum arbeitet —
    das ist eine Koordinationsfrage, die ich eigenmächtig entschieden habe.

---

## 5. (c) Was total verkackt ist

### 5.1 Der Kollisionsvorfall im Hauptbaum — der schwerste Fehler

**Was passiert ist:** Ich habe ~38 Dateien im Hauptbaum per `sed` editiert,
während eine zweite Session dort aktiv schrieb (`authz_contract.go` wurde
Sekunden vor meiner Prüfung angelegt, `options_types.go` Sekunden davor
gespeichert). Meine `sed`-Läufe haben *alle* Root-`.go`-Dateien angefasst — also
potenziell auch Dateien, die in deren Editor offen waren.

**Warum es schiefgehen konnte:** Kein Pre-Flight-Check. Ein `git status` +
`ls -l --time-style=+%H:%M:%S` **vor** dem ersten Edit hätte es in 2 Sekunden
gezeigt. Ich habe stattdessen erst nach den ersten Build-Fehlern hingeschaut.

**Schaden:** Kein Datenverlust — nachweislich nicht, weil (a) die fremden
Dateien inhaltlich unangetastet sind (deren Diff zeigt nur ihre eigenen
Änderungen), und (b) ich meinen Stand per Patch sauber herausgelöst und den
Hauptbaum auf `HEAD` für genau meine Pfade zurückgesetzt habe. Aber:
**Wenn ihre Session gleichzeitig `errors.go` angefasst hätte, wäre einer von uns
beiden überschrieben worden, ohne dass es jemand merkt.**

**Was ich daraus mitnehme (schon umgesetzt):** Arbeit an einem Ort mit
fremden, laufenden Änderungen gehört in einen `git worktree`, nicht in den
Hauptbaum. Die Regel fehlt in `AGENTS.md`.

### 5.2 `sed`-Massenumbau hat Definitionen zerstört

`sed 's/\bpanicCode\b/hx.PanicCode/g'` traf auch die **Definition**
(`const panicCode = "panic"`) und erzeugte `const hx.PanicCode = "panic"` —
Syntaxmüll, plus analog `func hx.IsPanicError(…)`. Dasselbe Muster bei
`SerializeToImmutableHandler`, `HashTag`, `ServeJS`.

**Lehre:** Massen-Renames dürfen nur auf *Aufrufstellen* zeigen. Sicheres Muster:
Definitionen zuerst manuell entfernen, dann `sed` über den Rest, oder ein
`gofmt -r`-bzw. `gopls rename`-Werkzeug benutzen. Ich habe es manuell repariert,
aber es hat zwei Iterationen gekostet.

### 5.3 Parameter-Namen durch `sed` verhunzt

`sed 's/\bincludeInternal\b/IncludeInternal/g'` machte aus einem
**Funktionsparameter** `includeInternal bool` ein `IncludeInternal bool` — Stil-
und Lint-Verstoß, den ich erst beim Sichten der Fundstellen bemerkt habe (Fix:
zurückbenannt). Ein Export-Rename muss auf `PlainBodyOptions`-Felder beschränkt
werden, nicht global laufen.

### 5.4 Test-Helper falsch „kopiert“

Mein `assertHTMXErrorRedirect` in `httperr/errors_test.go` setzte
`r.Header.Set(htmx.HeaderTrue, htmx.HeaderTrue)` — Header-**Name** war die
Konstante „true“. Der Test schlug fehl („expected `/auth/signin`, got “““), ich
habe ihn korrekt auf `htmx.HeaderRequest` gesetzt. Das ist ein 30-Sekunden-Fehler,
aber er zeigt: beim Verschieben von Tests kopiert man Semantik, nicht nur Text.

### 5.5 Zwei Request-ID-Keys — der Bug, den ich selbst gebaut habe

Die httperr-Extraktion führte zwei Context-Keys für dieselbe Information ein.
**Das ist exakt die „Split Brain“-Klasse, vor der die Skill warnt**, und ich habe
sie produziert — aufgedeckt nur, weil zwei Tests die Request-ID im Body prüfen.
Ohne diese Tests wäre der Fehler in Produktion gelandet (Fehlerantworten ohne
Request-ID, Support kann Vorfälle nicht mehr korrelieren), und er wäre extrem
schwer zu finden gewesen.

**Nicht behoben, sondern bewusst offen gelassen:** strukturelle Gleichheit
zwischen `hx.RequestID` und Root-`RequestID` ist ein Typ-Alias, also identisch —
aber zukünftige Kontextinformationen (Correlation-ID, Actor-ID) haben weiterhin
*nur* Root-Keys. Wenn `httperr` später eine Correlation-ID braucht, wiederholt
sich das Problem.

### 5.6 Cache-Fehlalarm als „Architektur-Problem“ diagnostiziert

Meine erste Isolationsrunde meldete `FAIL` für 7 von 29 Modulen. Ich hätte das
fast als Gate-5-Verletzung verbucht. Es war `/mnt/buildcache` (Gotcha 12). Zwei
Minuten später mit frischem `GOCACHE`: alles grün.

**Verkackt daran:** Ich habe die Fehlermeldungen nicht sofort gegen den
dokumentierten Gotcha abgeglichen, obwohl `go-cache-env.sh` im Repo genau
deshalb existiert.

---

## 6. (d) Was verbessert werden kann/muss

### 6.1 Sofort (vor jedem Commit dieses Stands)

| # | Aufgabe | Warum | Aufwand |
| --- | --- | --- | --- |
| 1 | `nix fmt` + `nix run .#lint` (Root-Modul) | Facade-Dateien, Doc-Kommentare, `var`-Blöcke sind lint-unklar | 10 min |
| 2 | `nix run .#check-modules` | 78 Checks; prüft Isolation, Dep-Budgets, Doku-Frische, Tail-Budget | 15 min |
| 3 | `nix run .#coverage-gate` | 91,5 % ist Handrechnung, nicht Gate | 5 min |
| 4 | `go test ./...` in `usermgmt`, `setup`, `dashboardui`, `adminui` | Build grün ≠ Verhalten grün; `htmx`-Tests sind jetzt woanders | 10 min |
| ~~5~~ | ~~Tail-Budget~~ done (**erledigt in dieser Session** — Archiv-Pass, Tail 4/4) | ~~war rot, ist grün~~ | ~~–~~ |
| 6 | Commit(s) in klaren Stufen | uncommitted = verlorene Arbeit | 5 min |

### 6.2 Strukturell

7. **Ein „Facade-Policy“-Absatz für `AGENTS.md`**: Root ist die stabile,
   veröffentlichte Oberfläche; jedes neue Feature gehört in ein Subpaket; Root
   bekommt nur Alias + Wrapper. Ohne diese Regel wächst Root wieder zu.
8. **Ein Gate für die Richtung** (Skill §Layer Enforcement): ein Skript, das die
   erlaubten Kanten (Root → Subpakete, nie Subpaket → Root, nie
   `htmx` → `httperr`) aus dem Quelltext liest und verletzte Kanten blockiert.
   Heute existiert dafür **nichts** — nur Prosa in `AGENTS.md`.
9. **`options_htmx.go`-Seam**: prüfen, ob `HandlerOption`-Anbindung über ein
   kleines Interface (`HTMXConfigurable`) aus dem `htmx`-Paket heraus möglich
   ist. Wenn ja, verschwindet der letzte htmx-Rest aus Root.
10. **`readiness` als nächstes Paket**: nach dem Muster von `httperr` (es braucht
    `hx` + `sse`; `sse` muss dafür `AfterDispatchHook` besitzen).
11. **`sync`/`openapi` erst mit ADR**: `App.SyncPushHandler` ist eine Methode —
    sie kann Go-bedingt nur im `App`-Paket leben. Ein sauberer Schnitt verlangt
    `App` in ein `core`-Paket zu ziehen, was v5-Material ist. Das gehört
    entschieden, nicht improvisiert.
12. **Coverage-Ziel für `htmx`** auf ≥ 90 % heben (Extension-Panic-Pfade).
13. **Benchmarks umziehen** (`benchmark_error_test.go` → `httperr`, etc.), damit
    Code und Messung am selben Ort liegen.

### 6.3 Prozess

14. **Pre-Flight-Regel**: vor dem ersten Edit `git status --porcelain` **und**
    Frische-Check (`find . -newer <marker>` oder `ls -l` auf die Zieldateien).
    Bei fremden Änderungen: Worktree, nicht diskutieren.
15. **Kein `sed` über `*.go`**: gezielt auf Dateiliste, Definitionen vorher
    manuell entfernen, danach `goimports`+`gofmt`.
16. **`.go.work.check-workspace-build.sum`** ist im Hauptbaum als gelöscht
    markiert (` D `) — ein Gate-Artefakt, das jemand (nicht ich) entfernt hat.
    Unklar, ob Absicht. Gehört geklärt, nicht „repariert“.

---

## 7. (e) Was ich noch nie gemacht habe / was als Nächstes kommt

### 7.1 In dieser Session nie getan

- Nie `nix run .#test`, `.#lint`, `.#check-modules`, `.#coverage-gate` **erfolgreich
  abgeschlossen** (nur ad-hoc `go test`).
- Nie einen Commit erzeugt (Regel 6).
- Nie `nix fmt` über die neuen Dateien laufen lassen.
- Nie die `docs/modularization`-HTML-Artefakte erzeugt (Skill-Output-Vorgabe).
- Nie `go mod tidy`/`go work sync` geprüft (Idempotenz ungeprüft).
- Nie die `scripts/checks/check-skill-claims.sh` /
  `check-readme-splitbrain.sh` / `check-readme-quickstart.sh` geprüft, die
  Symbolpfade in Doku gegen Code prüfen.
- Nie die **Beispiele** (`examples/*`) gebaut — sie sind eigene Module und
  nutzen `cqrshtmx.*`; die Facades sollten halten, aber ungeprüft.
- Nie `e2e/server` gebaut.
- Nie Performance gemessen (ob die Extraktion Import-/Buildzeit ändert).

### 7.2 Konkreter Plan für die nächste Session

1. **Stand sichern** (Commit in 4 Stufen: `internal/hx`, `htmx`, `eventcatalog`,
   `httperr` — jede Stufe für sich revertierbar).
2. **Gates fahren** in dieser Reihenfolge: `nix fmt` → `.#lint` →
   `go test ./...` pro Modul (mindestens Root + 9 Konsumenten) → `.#check-modules`
   → `.#coverage-gate`.
3. ~~Tail-Budget reparieren~~ done (**erledigt** — Archiv-Pass gelaufen, 4/4).
4. **Doku**: `AGENTS.md`-Architektur (4 neue Pakete + Richtungsregel),
   `CURRENT_FOCUS.md`, `docs/modularization/2026-11-02_EPISODE5.md` (oder HTML),
   Projekt-Skill-Tabelle.
5. **Richtungs-Gate bauen** (§6.2 Nr. 8) — das ist der einzige Weg, den Umbau
   dauerhaft zu halten.
6. **Dann** `readiness` und (mit ADR) `sse`; `sync`/`openapi` erst nach der
   `core`-Entscheidung.

### 7.3 Offene Entscheidungen, die ich nicht allein treffen sollte

- **Soll Root langfristig eine reine Facade werden** (alles in `core/`), oder
  bleibt Root der Kern und die Subpakete sind Ausnahmen? Das ist eine
  Produkt-/DX-Entscheidung (`cqrshtmx.NewApp` als Einstiegspunkt vs.
  `core.NewApp`), nicht eine mechanische.
- **Werden die Facades irgendwann deprecated?** (Muster: `csrf_reexport.go`
  trägt `// Deprecated: … removed in v5`.) Meine Facades tragen bewusst **kein**
  Deprecated, weil Root die Hauptoberfläche bleibt — das ist eine Annahme, keine
  Messung.
- **Soll `internal/hx` ein eigenes Modul werden?** Nein (kein Nutzen), aber die
  Frage kommt garantiert.

---

## 8. (f) Was mir an den bestehenden Regeln, Gates und der Doku nicht gefällt

### 8.1 Regeln

1. **„Never commit unless asked“ + Skill-Workflow „commit at phase boundaries“
   widersprechen sich.** Ergebnis: ein fertiger, verifizierter Umbau liegt
   uncommitted in einem Worktree — das größte Risiko des ganzen Tages ist damit
   hausgemacht. Entweder die Regel bekommt eine Ausnahme für
   Worktree-/Branch-Arbeit, oder die Skill muss sagen: „stoppe vor Phase 6 und
   frage“.
2. **Die Skill verlangt HTML-Proposal + HTML-Plan; die Repo-Realität arbeitet
   mit Markdown-Statusreports.** Zwei Wahrheiten. Entweder die Skill bekommt
   eine „Docs-Äquivalenz“-Klausel (Markdown-ADR + Statusreport zählt), oder das
   Repo braucht wirklich `docs/modularization/*.html` — dann kostet jede
   Modularisierung zwei Dokumente, die niemand liest.
3. **Keine Regel gegen Parallel-Sessions im selben Arbeitsbaum.** Das ist der
   gefährlichste Zustand, den dieses Repo erzeugen kann (Auto-Commit-Daemon +
   Mensch + Agent). Ein Satz in `AGENTS.md` („fremde uncommittete Änderungen →
   `git worktree`“) würde reichen.
4. **Gotcha 12 (Cache) wird als Fußnote behandelt**, obwohl er regelmäßig
   *Gates* rot färbt und damit Fehldiagnosen produziert. Er gehört in den
   Session-Start-Runbook als erster Schritt („source go-cache-env.sh“), nicht in
   eine Gotcha-Tabelle.

### 8.2 Gates

5. **`check-modules` hat 78 Checks, aber keinen für Import-Richtung.** Die
   wichtigste Architektureigenschaft dieses Repos („root importiert usermgmt
   nie“, „dashboardui hängt nicht an usermgmt“) ist **handgeprüfte Prosa**
   (2026-10-25 „verified against go.mod“). Ein 40-Zeilen-Grep-Gate würde das
   mechanisieren.
6. **Coverage wird pro Modul gemessen, nicht pro Paket.** Neue Pakete sind
   dadurch unsichtbar: `go test ./... -coverprofile` lässt Pakete ohne Tests
   einfach aus dem Profil — die Modulsumme *steigt* scheinbar, während Code
   ungetestet ist. Das ist ein stiller Anreiz, Tests „zu vergessen“.
7. **`check-docs-tail-budget` (Baseline 4) kollidiert mit „schreibe einen
   ausführlichen Statusreport“** — in dieser Session live erlebt: 5/4 rot,
   Archiv-Pass nötig, danach 4/4 grün. Der Konflikt ist strukturell: jeder
   ehrliche Report kostet ein Archiv-Ritual (hier ~3 Minuten, weil das Werkzeug
   die Reihenfolge erzwingt). Jeder ehrliche Report macht Arbeit (Archiv-
   Ritual), also schreibt man weniger Reports. Das ist ein falscher Anreiz:
   Qualität der Dokumentation wird durch Mengenbegrenzung bestraft.
8. **Die Gates sind schnell, aber die *Ausgabe* ist es nicht.** `check-modules`
   läuft im Hintergrund und liefert bei Abbruch/Worktree-Wechsel gar nichts —
   ich habe den Lauf verloren und nicht wiederholt. Ein Gate, dessen Ergebnis
   man verliert, ist kein Gate.

### 8.3 Doku

9. **Die 2026-10-18-Analyse behauptet für Root „casbin (via usermgmt only)“** —
   das ist **falsch**: Root hat `github.com/casbin/casbin/v3` als *direkten*
   Require, und **nur vier `_test.go`-Dateien** importieren es. Root importiert
   `usermgmt` null Mal. Die Aussage ist an zwei Stellen ungenau und hat 13 Tage
   überlebt, weil niemand sie nachgemessen hat. (Für Konsumenten ist der Effekt
   klein: casbin landet in deren `go.sum`, nicht in `go.mod` — ich habe das mit
   einem Wegwerf-Consumer verifiziert.)
10. **`docs/modularization/` enthält sieben Analysen und keine davon sagt, welche
    noch gilt.** `DEPENDENCY_GRAPH.md`, `PROPOSAL.md`, `EXECUTION_PLAN.md` ohne
    Datum im Namen — ein Leser weiß nicht, ob das Historie oder Stand ist. Der
    Rest des Repos macht das (Datum im Namen) richtig.
11. **„Way too full“ ist nirgends definiert.** Die Skill hat Schwellen (15+
    Dateien, 30+ Symbole); das Repo hat keine. Ein Satz in `AGENTS.md`
    („Paket > 20 Dateien = Aufteilung prüfen“) würde solche Sessions verkürzen.
12. **Die Testdatei-Namen sind Cluster-frei gemischt** (`coverage_*`,
    `example_*`, `testing_*`, `benchmark_*`). Nach diesem Umbau ist die
    Zuordnung „Test liegt beim Code“ teilweise verletzt (die `benchmark_*` und
    `coverage_*`-Dateien decken jetzt mehrere Pakete über die Facade ab). Das ist
    eine Doku-/Namens-Schuld, die niemand aufgeschrieben hat.

---

## 9. (g) Die drei wichtigsten Empfehlungen + 4-%-Pareto

### 9.1 Die drei wichtigsten Empfehlungen

1. **Diesen Stand sofort sichern und die Gates fahren** — nicht weiter
   extrahieren, bevor `.#lint`, `.#check-modules`, `.#coverage-gate` und die
   Konsumenten-Tests grün sind. Ein uncommitteter, ungelinteter Umbau an einer
   veröffentlichten Bibliothek ist die größte offene Risk-Position.
2. **Ein Import-Richtungs-Gate bauen** (`check-module-layers.sh`, ~40 Zeilen,
   grep-basiert, mit Fixture-Selbsttest). Es ist das Einzige, was den Umbau
   dauerhaft hält: heute kann jeder versehentlich `htmx → httperr` oder
   `Subpaket → root` importieren, und niemand merkt es, bis ein Zyklus baut.
3. **Parallel-Session-Regel in `AGENTS.md`** („fremde uncommittete Änderungen im
   Arbeitsbaum → `git worktree add`“) plus Pre-Flight-Check. Der Kollisionsvorfall
   dieser Session war der teuerste Moment und der am leichtesten vermeidbare.

### 9.2 4 % Aufwand → 64 % Wirkung (Pareto, konkret)

| # | 4 %-Maßnahme | Wirkung | Aufwand |
| --- | --- | --- | --- |
| P1 | `nix fmt && nix run .#lint` (Root) | verhindert roten CI-Lauf, deckt `revive`/`gochecknoglobals`/`dupl`-Probleme der Facades | 10 min |
| P2 | `go test ./...` in den 9 Konsumenten-Modulen | Verhalten, nicht nur Build — der eigentliche Beweis, dass die Facades halten | 10 min |
| P3 | `git commit` in 4 Stufen | Arbeit ist gesichert, jede Stufe revertierbar | 5 min |
| P4 | Import-Richtungs-Gate + Fixture-Test | hält die neue Struktur für immer | 45 min |
| P5 | `AGENTS.md`-Architektur + Facade-Policy | jeder zukünftige Agent weiß, wo neuer Code hingehört | 20 min |
| ~~P6~~ | ~~Tail-Budget-Archivpass~~ | ~~done (**erledigt** in dieser Session — 4/4, Gate grün)~~ | ~~3 min~~ |

Summe ≈ 110 Minuten für: gesicherte Arbeit, grüne Gates, dauerhafte Richtung,
Doku-Konsistenz — also für alles, was einen Umbau von „passiert“ zu „gilt“
macht.

### 9.3 Was das *nicht* löst

Die Zahl der Dateien unter Root sinkt von 57 auf 50. Wenn das Ziel „deutlich
weniger als 40“ ist, reicht Mechanik nicht: dann braucht es die Entscheidung aus
§7.3 — **Root als reine Facade über einem `core`-Paket**. Das ist ein
v5-Vorhaben mit 17,6k Zeilen Test-Umzug und einer öffentlichen API-Frage
(`cqrshtmx.NewApp` vs. `core.NewApp`), kein Nachmittagsjob.

---

## 10. Anhang

### 10.1 Messbefehle (reproduzierbar, im isolierten Worktree)

```bash
# Modul-Landschaft
find . -name go.mod -not -path './vendor/*' -not -path './scripts/testdata/*' | wc -l   # 29
env GOWORK= go work edit -json | jq -r '.Use[].DiskPath'                                  # 29 use-Zeilen

# Root-Paket-Größe
ls *.go | grep -cv '_test\.go'                     # 57 → 50
ls *_test.go | wc -l                               # 105 → 96
grep -hE '^(type|func|var|const) [A-Z]' $(ls *.go | grep -v _test) | wc -l   # 270 Symbole

# Isolations-Gate pro Modul (mit frischem Cache!)
for gomod in $(find . -name go.mod -not -path './vendor/*' -not -path './scripts/testdata/*'); do
  (cd "$(dirname "$gomod")" && GOWORK=off GOEXPERIMENT=jsonv2 GOTOOLCHAIN=go1.27.2 \
     GOCACHE=$HOME/.cache/go-build-baseline go build ./... ) || echo "FAIL $gomod"
done

# Tests + Coverage
GOEXPERIMENT=jsonv2 GOTOOLCHAIN=go1.27.2 GOCACHE=$HOME/.cache/go-build-baseline \
  go test ./... -count=1 -coverprofile=/tmp/cov.out
go tool cover -func=/tmp/cov.out | tail -1                  # total: 91,5 %

# Konsumenten (je einzeln)
for m in usermgmt adminui dashboardui setup loginpage datastar health systemadapter auditlog; do
  (cd $m && GOEXPERIMENT=jsonv2 GOTOOLCHAIN=go1.27.2 go build ./... ) && echo "OK $m"
done
```

### 10.2 Cluster-/Kanten-Rohdaten (aus dem AST-Werkzeug)

Werkzeug: `~/.cache/modgraph/main.go` (stdlib-only `go/parser`, sammelt pro Datei
die fremden Top-Level-Symbole; lokale Bezeichner werden abgezogen).

```
CLUSTERS                    files    loc   syms
core                          33    4721    293
errors                         3     766     55
sync                           5    1043     60
htmx                           5     513     68
readiness                      3     406     25
sse                            4     365     27
catalog                        2     174     10
openapi                        2     137      6

KANTEN (referenzierende Dateien)
errors    -> core   9      core    -> htmx    5
sync      -> core   9      htmx    -> core    4
readiness -> core   7      catalog -> core    3
                           openapi -> core    3
                           sse     -> core    3
core      -> sync   1      errors  -> htmx    1
                           openapi -> catalog 1
                           sync    -> htmx    1

core -> {errors, sse, catalog, openapi, readiness} = 0 Kanten
```

Bekannte Heuristik-Artefakte (jeweils am Quelltext widerlegt):
`buf` (lokales `bytes.Buffer`), `errFunc`, `errCode` (lokale Variablen).

### 10.3 Neue/verschobene Dateien

**Neu:**

```
internal/hx/{doc,constants,response,immutable,context,panic}.go
internal/hx/{immutable,response}_test.go
htmx/{htmx,embed,extensions,serve}.go
htmx/{core,swap,extensions,serve,suite}_test.go
htmx/htmx.min.js
htmx/extensions/{sse.min.js,idiomorph-ext.min.js}
eventcatalog/{catalog,handler}.go
eventcatalog/{catalog,handler}_test.go
httperr/{errors,status,structured}.go
httperr/{errors,structured,carrier_internal,wrappers_internal,suite}_test.go
htmx_facade.go
httperr_facade.go
eventcatalog_facade.go
serve_conditional_get_test.go        (Ersatz für die mitgewanderten Specs)
```

**Gelöscht:** `safe_write.go` (Inhalt → `internal/hx/response.go`),
`event_catalog_helpers_test.go` (Inhalt → `internal/hx/immutable_test.go`).

### 10.4 Risiko-Register (Stand jetzt)

| Risiko | Schwere | Status |
| --- | --- | --- |
| Uncommittete Arbeit in einem Worktree | hoch | offen (§4.3/§7.2) |
| `.#lint` nie gelaufen; `revive`/`gochecknoglobals`/`dupl` könnten rot sein | mittel | offen |
| `check-docs-tail-budget` | mittel | **behoben** (Archiv-Pass, 4/4 grün) |
| `htmx`-Coverage 77,5 % | mittel | offen |
| Keine Richtungs-Gate → Rückfall in God-Package | mittel | offen (Empfehlung 2) |
| Parallele Session im Hauptbaum | hoch | **entschärft** durch Worktree-Trennung |
| Benchmark-/Coverage-Testdateien decken jetzt mehrere Pakete | niedrig | dokumentiert |
| `go mod tidy`/`go work sync` ungeprüft | niedrig | offen |

### 10.5 Ein Satz zum Schluss

Der Umbau ist technisch grün und strukturell ein echter Gewinn — aber er ist
**nicht fertig**, weil „fertig“ in diesem Repo heißt: Gates grün, Doku
aktualisiert, Arbeit committed. Was fehlt, ist keine Erkenntnis, sondern eine
Stunde Disziplin; und was am teuersten war, war nicht der Code, sondern die
fehlende Vorprüfung, ob jemand anderes gerade dasselbe Verzeichnis bearbeitet.
