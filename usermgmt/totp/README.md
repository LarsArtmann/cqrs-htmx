# usermgmt/totp

TOTP MFA for [usermgmt](../README.md), built on [pquerna/otp]. An independent
Go module — import it only if you want TOTP; core usermgmt carries zero auth
dependencies.

```go
import usermgmttotp "github.com/larsartmann/cqrs-htmx/usermgmt/totp/v4"

svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
    TOTP: usermgmttotp.New(usermgmttotp.Config{Issuer: "MyApp"}),
})
```

Inject the provider as `ServiceConfig.TOTP` (the interface lives in usermgmt;
this package is the reference implementation). Users enroll via the auth
routes; secrets persist through the user read model. Config pointers and the
full auth-strategy matrix: [usermgmt/README.md](../README.md).

[pquerna/otp]: https://github.com/pquerna/otp
