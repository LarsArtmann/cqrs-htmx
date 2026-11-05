# Route postures: declaring what every route requires

> Every HTTP surface this library ships declares, in code, what each of its
> routes requires of a caller — and a gate (`nix run .#check-modules`, stage
> `route-postures`) fails when a module's manifest drifts from its pin.
> Related: [authorization-system](authorization-system.md) (the decision
> contract), [`../PERMISSIONS.md`](../PERMISSIONS.md) (the maturity levels —
> this is the L4 exit criterion).

## The vocabulary

`cqrshtmx.Posture` (root, `route_posture.go`) is the one vocabulary every
manifest speaks:

| Posture                | Meaning                                                                                         | The matrix proves                    |
| ---------------------- | ----------------------------------------------------------------------------------------------- | ------------------------------------ |
| `PosturePublic`        | answers an anonymous caller                                                                     | anonymous request is not 401/403     |
| `PostureAuthenticated` | refuses anonymous, admits any signed-in caller                                                  | anonymous 401/403, authenticated 200 |
| `PostureRoleGated`     | refuses anonymous, 403s authenticated-without-role, admits the role                             | all three cases                      |
| `PostureUngated`       | **this layer adds no gate** — the routes beneath, or the consumer's middleware, own the posture | nothing — the matrix refuses it      |

`PostureUngated` exists so nobody has to lie: a dashboard mounted without an
authorizer enforces nothing itself, and `setup`'s `/auth/*` layer delegates to
usermgmt's per-route gates. A manifest states what its layer actually enforces.
`authztest` rejects `Ungated` declarations — probe the real routes through
your actual wiring and declare the posture that wiring enforces.

## The manifests

| Module        | Accessor                                         | Truth it reports                                               |
| ------------- | ------------------------------------------------ | -------------------------------------------------------------- |
| `adminui`     | `Handler.Routes() []adminui.Route`               | per-route, this instance's Mode and `Config.AuthorizerPosture` |
| `dashboardui` | `Dashboard.Routes() []dashboardui.Route`         | per-route, this instance's capabilities/ReadOnly/authorizer    |
| `loginpage`   | `Handler.Routes() []cqrshtmx.RoutePosture`       | one public row                                                 |
| `setup`       | `Bundle.RoutePostures() []cqrshtmx.RoutePosture` | one row per mounted layer, honoring Disable*/paths             |

Package ownership is deliberate: the shared vocabulary (`Posture`, `RoutePosture`)
is owned by the root `cqrshtmx` package so every manifest row — and
`authztest`'s matrix, which aliases the same constants — speaks one language;
`adminui.Route` and `dashboardui.Route` are the UI modules' own richer row types
(carrying Mode/guard detail), each exposing derived `Posture` + `Why` fields
rather than embedding the root type. Importing root for the vocabulary is the
intended dependency direction; UI modules importing each other is not.

In adminui and dashboardui the route **table is the registration source**: the
mux and the manifest are generated from the same rows, so a route outside the
manifest cannot be served. A row that forgets its mode (adminui) or guard
(dashboardui) is served by no mode rather than silently by both — absence
fails loudly at runtime; over-exposure fails only in an audit.

Every row carries a `Why`. A posture without a reason is a guess, and a guess
reviewed later is indistinguishable from a decision.

## Custom authorizers: say what they enforce

adminui's default authorizer is role-based, so its manifest defaults to
`PostureRoleGated`. A custom `Config.Authorizer` is an opaque function — the
manifest cannot know what it enforces, so you declare it:

```go
panel, err := adminui.New(adminui.Config{
    Service:           svc,
    Authorizer:        adminui.RequireAuthenticated(),
    AuthorizerPosture: cqrshtmx.PostureAuthenticated, // the truth for this authorizer
})
```

Values other than empty / `PostureAuthenticated` / `PostureRoleGated` are
rejected at `New`: the guard always refuses anonymous callers, so `public` and
`ungated` would be lies.

## The gate

`scripts/checks/check-route-postures.sh` (in `check-modules` and CI) runs every
module's manifest pin tests in workspace mode — the submodule `go.mod` files
lag the family train, so hermetic `GOWORK=off` runs would compile against the
previous published tag. A module whose test pattern matches nothing fails
loudly: a green that verified nothing is the worst possible outcome.

Changing a route means updating the golden in the same commit — the diff IS
the review.

## Probing your own wiring: authztest

A manifest declares; [`authztest`](../../authztest/README.md) proves. Feed the
manifest rows into `authztest.Suite` declarations (through your fully wired
handler, middleware included) and the matrix probes every posture. Use
`authztest.Undeclared(manifestRoutes, registered)` to catch the other
direction: a route nobody declared.
