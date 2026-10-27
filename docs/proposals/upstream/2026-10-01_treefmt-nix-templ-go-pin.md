> [!IMPORTANT]
> This issue was found and reported by GLM (glm-5.3-flash) via Crush independent of me.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

## Problem

`programs/templ.nix` hardcodes nixpkgs' default Go in the templ wrapper:

```nix
// programs/templ.nix:24-32 (master)
command = pkgs.writeShellApplication {
  name = "templ";
  runtimeInputs = [
    pkgs.go
    pkgs.templ
  ];
  text = ''
    exec templ "$@"
  '';
};
```

`templ fmt` shells out to that `go`. When the project's `go.mod` floor is newer than nixpkgs' default Go (here: project `go 1.27.1`, wrapper `pkgs.go` 1.26.x), templ triggers a `GOTOOLCHAIN=auto` toolchain download. Inside a network-free sandbox (nix build / `nix flake check` hermetic checks) that download cannot succeed, and templ fmt hard-fails — taking the whole treefmt run down with it.

## Impact

Every treefmt-nix user whose project Go floor is above nixpkgs' default Go gets red formatting checks in hermetic contexts; outside sandboxes the same invocation silently downloads a toolchain (slow, non-hermetic). Reproduced on NixOS with nixpkgs default Go 1.26.x against a `go 1.27.1` module; the workaround below is validated offline in our flake (`rc=0, changed=0`).

## Fix / Proposal

Make the wrapper's Go opt-in overridable, e.g. `programs.templ.goPackage` (default `pkgs.go`), so a project can pin the same Go its `go.mod` requires — keeping #365's "templ needs a go" fix while letting floor-newer-than-default projects stay hermetic.

Today we work around it by replacing the whole command (`settings.formatter.templ.command` accepts a full override — pin the project's Go + `GOTOOLCHAIN=local` + `GOPROXY=off`), which works but re-implements the wrapper per project.

--
💘 Generated with Crush
