# admin-demo

Runnable showcase of the [adminui](../../adminui/README.md) panel: an
in-memory usermgmt service with seeded demo users and tenants, the admin
panel mounted at `/admin` behind cookie session auth. Open the root URL and
you are signed in as the demo admin automatically.

```bash
GOEXPERIMENT=jsonv2 go run .   # http://localhost:18930
```
