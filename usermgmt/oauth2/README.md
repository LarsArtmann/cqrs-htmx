# usermgmt/oauth2

OAuth2/OIDC login for [usermgmt](../README.md), built on
[golang.org/x/oauth2] + [coreos/go-oidc]. An independent Go module — import
it only if you want federated login; core usermgmt carries zero auth
dependencies.

```go
import (
    "context"

    usermgmtoauth2 "github.com/larsartmann/cqrs-htmx/usermgmt/oauth2/v4"
)

federated, err := usermgmtoauth2.New(context.Background(), usermgmtoauth2.Config{
    Providers: map[string]usermgmtoauth2.ProviderConfig{
        "google": {
            ClientID:     "...",
            ClientSecret: "...",
            IssuerURL:    "https://accounts.google.com", // OIDC discovery
            RedirectURL:  "https://app.example.com/auth/oauth2/google/callback",
        },
    },
})
if err != nil {
    log.Fatal(err)
}
svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
    OAuth2: federated,
})
```

Inject the provider as `ServiceConfig.OAuth2` (the interface lives in
usermgmt; this package is the reference implementation). Public PKCE clients
(secret-less) are supported via `ProviderConfig.ClientType`, OIDC providers
auto-configure from `IssuerURL` discovery, and `FinishLoginWithToken`
accepts a raw ID token when your frontend already completed the flow.
Config pointers and the full auth-strategy matrix:
[usermgmt/README.md](../README.md).

[golang.org/x/oauth2]: https://pkg.go.dev/golang.org/x/oauth2
[coreos/go-oidc]: https://github.com/coreos/go-oidc
