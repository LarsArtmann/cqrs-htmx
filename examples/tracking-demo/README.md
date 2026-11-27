# tracking-demo

A runnable demo of cqrs-htmx client tracking on the [`setup/v4`](../../setup)
bundle: the bundle serves the embedded tracker, stores every page view,
click and impression in its engine-owned signals journal, and a
read-your-signals page lists what was recorded.

```bash
GOEXPERIMENT=jsonv2 go run .   # open http://localhost:8094
```

## The two-minute walkthrough

1. **Open the shop** at `/`. The page carries one line, the tracker tag;
   nothing else is needed to collect. A page view is recorded.
2. **Open `/signals` in a second tab.** It lists every signal, newest first:
   your page view with the `device` column filled (the key the tracker keeps
   in `localStorage["cqrs-htmx-client-id"]`).
3. **Click "Add to cart"** on the shop tab and reload `/signals`: a `click`
   row with the name `add-to-cart` and the target `button`. The three cards
   are `impression` rows (`data-track-impression`).
4. **Follow "Go to checkout"** and reload `/signals`: the shop page view is
   closed by one `engagement` row carrying its visible time, and the checkout
   page view carries the shop as its referrer.
5. **Sign in as the shopper** (`/dev-login`), visit a page, then open
   "Only the shopper" on `/signals`: the new signals carry the `user` column.

## Routes

| Route           | Auth   | What it is                                                            |
| --------------- | ------ | --------------------------------------------------------------------- |
| `/`             | public | The shop page: the tracker tag, tracked buttons and cards             |
| `/checkout`     | public | A second page, so a navigation and its referrer can be watched        |
| `/dev-login`    | public | Demo shortcut: signs the demo shopper in (never in a real app)        |
| `/signals`      | public | Every recorded signal, newest first; `?user=<id>` or `?device=<id>`   |
| `/-/tracker.js` | public | The embedded tracker script (setup's `TrackerScriptPath` default)     |
| `POST /-/track` | public | The tracking endpoint (first-party origin proved, every field capped) |
| `/health`       | public | Readiness check                                                       |

## What the app had to do

```go
bundle, err := setup.New(setup.Config{
	TrackingBackend: streamLog(metaengine.NewMemoryEngine()),
})
```

That is the whole wiring. The bundle mounts the tracker script and the
endpoint, and builds the signals journal over the given engine. An app whose
event store is the one go-cqrs-lite's `system` builds passes nothing: the
journal then shares that store's engine. The pages add the tracker tag and
the two attributes:

```html
<script src="/-/tracker.js" defer></script>
<button data-track="add-to-cart">Add to cart</button>
<div data-track-impression="card:coffee">...</div>
```

## Reading signals back

`bundle.TrackingJournal()` is the journal: `All(ctx)` lists everything,
`ForUser(ctx, id)` and `ForClient(ctx, id)` read one subject's stream. A
signal with a signed-in user lives in the user's stream; otherwise it lives
in the device's stream; a visitor known by neither lands in the shared
anonymous stream. One signal, one subject.

## Privacy, by construction

- Identity and time come from the server: the tracker sends neither, and the
  endpoint ignores both in a request body.
- Only the path name is recorded; query strings and fragments are cut, and a
  referrer keeps its origin and path only.
- Element text, links and attribute values other than `data-track` and
  `data-track-impression` are never recorded.
- Nothing is collected for a visitor who opted out: Global Privacy Control,
  `<html data-tracking="off">`, or `window.cqrsTracker.disable()`.

This is a demo: the engine is in memory, so signals are gone when the process
stops, and `/dev-login` is a shortcut without a password.
