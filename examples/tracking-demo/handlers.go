package main

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"slices"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// The demo shopper and the session cookie the bundle reads.
const (
	cookieName   = "session"
	shopperID    = "01JXTRACKINGDEMO1"
	shopperEmail = "shopper@demo.dev"
)

// trackerScript is where the bundle serves the tracker (setup's default
// TrackerScriptPath).
const trackerScript = "/-/tracker.js"

// layout is the shell of every page. The tracker tag is the one line an app
// adds to a layout; the bodies only add elements worth tracking.
var layout = template.Must(template.New("layout").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{.Title}} · cqrs-htmx tracking demo</title>
<style>
body { font-family: system-ui, sans-serif; margin: 2rem auto; max-width: 52rem; padding: 0 1rem; line-height: 1.5; }
nav a { margin-right: 1rem; }
.cards { display: grid; grid-template-columns: repeat(3, 1fr); gap: 1rem; }
.card { border: 1px solid #ddd; border-radius: 8px; padding: 1rem; }
table { border-collapse: collapse; width: 100%; font-size: 0.9rem; }
th, td { text-align: left; padding: 0.35rem 0.5rem; border-bottom: 1px solid #eee; }
td.num { text-align: right; font-variant-numeric: tabular-nums; }
</style>
</head>
<body>
<nav><a href="/">Shop</a><a href="/signals">Signals</a><a href="/dev-login">Sign in as the shopper</a></nav>
{{.Body}}
<script src="{{.Tracker}}" defer></script>
</body>
</html>
`))

// shopBody is the landing page: the three cards are impressions, the buttons
// are clicks, and leaving for /checkout ends the page view (its visible time
// becomes one engagement signal).
const shopBody = `<h1>Tracking demo shop</h1>
<p>Open <a href="/signals">Signals</a> in a second tab and click around.</p>
<div class="cards">
	<div class="card" id="coffee" data-track-impression="card:coffee"><h2>Coffee</h2><button data-track="add-to-cart">Add to cart</button></div>
	<div class="card" id="tea" data-track-impression="card:tea"><h2>Tea</h2><button data-track="add-to-cart">Add to cart</button></div>
	<div class="card" id="cocoa" data-track-impression="card:cocoa"><h2>Cocoa</h2><button data-track="add-to-cart">Add to cart</button></div>
</div>
<p><a href="/checkout">Go to checkout</a></p>`

const checkoutBody = `<h1>Checkout</h1><p>Nothing is sold here; the page view and its referrer are the point.</p>`

// pageData is the layout's data.
type pageData struct {
	Title   string
	Tracker string
	Body    template.HTML
}

// page renders the layout around a body that is part of this program.
func (d *demo) page(title, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		d.render(w, pageData{Title: title, Tracker: trackerScript, Body: template.HTML(body)})
	}
}

// devLogin signs the demo shopper in. A demo shortcut only: a real app signs
// visitors in through its login page.
func (d *demo) devLogin(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{ //nolint:exhaustruct // a demo cookie
		Name: cookieName, Value: d.token, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// signalsTable is the body of /signals: one row per recorded signal.
var signalsTable = template.Must(template.New("signals").Parse(`<h1>Recorded signals</h1>
<p>
	{{if .Subject}}
		Showing <b>{{.Subject}}</b>. <a href="/signals">Everyone</a>
	{{else}}
		<a href="/signals?user={{.Shopper}}">Only the shopper</a> (sign in first at /dev-login)
	{{end}}
</p>
<p>
	Every row is one signal, newest first. <code>user</code> is set for signed-in visitors,
	<code>device</code> for every browser that sent its key, and <code>visible ms</code> is the
	foreground time of a finished page view. To see one browser's stream, take its key from
	<code>localStorage["cqrs-htmx-client-id"]</code> and open <code>/signals?device=&lt;key&gt;</code>.
</p>
<table>
	<thead><tr><th>at</th><th>kind</th><th>path</th><th>name</th><th>target</th><th>visible ms</th><th>user</th><th>device</th></tr></thead>
	<tbody>
	{{range .Signals}}
		<tr>
			<td>{{.At.Format "15:04:05.000"}}</td>
			<td>{{.Kind}}</td>
			<td>{{.Path}}</td>
			<td>{{.Name}}</td>
			<td>{{.Target}}</td>
			<td class="num">{{.VisibleMS}}</td>
			<td>{{if not .UserID.IsZero}}{{.UserID}}{{end}}</td>
			<td>{{if not .ClientID.IsZero}}{{.ClientID}}{{end}}</td>
		</tr>
	{{else}}
		<tr><td colspan="8">Nothing recorded yet. <a href="/">Visit the shop</a> and come back.</td></tr>
	{{end}}
	</tbody>
</table>`))

// signalsData is the table's data.
type signalsData struct {
	Signals []cqrshtmx.TrackedEvent
	Subject string // "" = everyone; "user <id>" or "device <id>" otherwise
	Shopper string // the demo shopper's user key, for the only-the-shopper link
}

// errNoSubject marks a ?user= or ?device= value that is not an ID.
var errNoSubject = errors.New("not a user or device id")

// signals lists the journal, newest first: everyone's signals, or one
// subject's stream when ?user= or ?device= names one.
func (d *demo) signals(w http.ResponseWriter, r *http.Request) {
	subject, signals, err := d.subjectSignals(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	slices.Reverse(signals)

	d.render(w, pageData{
		Title:   "Signals",
		Tracker: trackerScript,
		Body:    d.signalsBody(signalsData{Signals: signals, Subject: subject, Shopper: d.shopper.ID.String()}),
	})
}

// subjectSignals reads what the query names: ForUser, ForClient, or All.
func (d *demo) subjectSignals(r *http.Request) (string, []cqrshtmx.TrackedEvent, error) {
	journal := d.bundle.TrackingJournal()
	query := r.URL.Query()

	user, device := query.Get("user"), query.Get("device")

	switch {
	case user != "":
		userID, err := id.ParseUserID(user)
		if err != nil {
			return "", nil, errNoSubject
		}

		signals, err := journal.ForUser(r.Context(), userID)

		return "user " + userID.String(), signals, err
	case device != "":
		clientID, err := id.ParseClientID(device)
		if err != nil {
			return "", nil, errNoSubject
		}

		signals, err := journal.ForClient(r.Context(), clientID)

		return "device " + clientID.String(), signals, err
	}

	signals, err := journal.All(r.Context())

	return "", signals, err
}

// signalsBody renders the table. Its output is this program's own template
// over escaped fields, so it is safe to hand to the layout as HTML.
func (d *demo) signalsBody(data signalsData) template.HTML {
	var body strings.Builder

	if err := signalsTable.Execute(&body, data); err != nil {
		log.Printf("tracking-demo: render the signals table: %v", err)
	}

	return template.HTML(body.String())
}

// render executes the layout; a rendering failure is logged, never hidden.
func (d *demo) render(w http.ResponseWriter, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := layout.Execute(w, data); err != nil {
		log.Printf("tracking-demo: render %s: %v", data.Title, err)
	}
}
