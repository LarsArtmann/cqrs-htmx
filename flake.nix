{
  description = "Go library for go-cqrs-lite with HTMX, templ, and Casbin authorization";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
    systems.url = "github:nix-systems/default";
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{ self, flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = import inputs.systems;

      imports = [ inputs.treefmt-nix.flakeModule ];

      perSystem =
        {
          config,
          pkgs,
          lib,
          ...
        }:
        let
          goPkg = pkgs.go_1_26;

          goEnv = ''
            export GOWORK=off
            export GOPRIVATE='github.com/larsartmann/*,github.com/LarsArtmann/*'
            export GOEXPERIMENT=jsonv2
            export GOTOOLCHAIN=local
          '';

          goApp =
            {
              name,
              text,
              description ? null,
              runtimeInputs ? [ ],
            }:
            {
              type = "app";
              meta = lib.optionalAttrs (description != null) { inherit description; };
              program = pkgs.writeShellApplication {
                inherit name;
                runtimeInputs = [ goPkg ] ++ runtimeInputs;
                text = goEnv + text;
              };
            };
        in
        {
          treefmt = {
            projectRootFile = "go.mod";
            programs = {
              nixfmt.enable = true;
              templ.enable = true;
              gofmt.enable = true;
            };
          };

          packages.default = pkgs.stdenvNoCC.mkDerivation {
            pname = "cqrs-htmx";
            version = self.rev or self.dirtyRev or "dev";

            dontUnpack = true;
            dontConfigure = true;
            dontBuild = true;

            installPhase = ''
              mkdir -p $out
            '';

            meta = with lib; {
              description = "Go library for go-cqrs-lite with HTMX, templ, and Casbin authorization";
              homepage = "https://github.com/larsartmann/cqrs-htmx";
              license = licenses.mit;
              maintainers = [
                {
                  name = "Lars Artmann";
                  github = "LarsArtmann";
                }
              ];
              platforms = platforms.unix;
            };
          };

          devShells = {
            default = pkgs.mkShellNoCC {
              packages = [
                goPkg
                pkgs.gopls
                pkgs.golangci-lint
                pkgs.tailwindcss_4
                pkgs.templ
              ];

              GOWORK = "off";
              GOPRIVATE = "github.com/larsartmann/*,github.com/LarsArtmann/*";
              GOEXPERIMENT = "jsonv2";
              GOTOOLCHAIN = "local";
            };

            ci = pkgs.mkShellNoCC {
              packages = [
                goPkg
                pkgs.golangci-lint
                pkgs.templ
              ];

              GOWORK = "off";
              GOPRIVATE = "github.com/larsartmann/*,github.com/LarsArtmann/*";
              GOEXPERIMENT = "jsonv2";
              GOTOOLCHAIN = "local";
            };
          };

          checks = {
            build = config.packages.default;
            formatting = config.treefmt.build.check self;
          };

          apps = {
            test = goApp {
              name = "run-tests";
              text = ''
                echo "==> Root module"
                go test ./... -count=1 -race
                echo "==> identity-model submodule"
                (cd identity-model && go test ./... -count=1 -race)
                echo "==> usermgmt submodule"
                (cd usermgmt && go test ./... -count=1 -race)
                echo "==> usermgmt/totp submodule"
                (cd usermgmt/totp && go test ./... -count=1 -race)
                echo "==> usermgmt/webauthn submodule"
                (cd usermgmt/webauthn && go test ./... -count=1 -race)
                echo "==> usermgmt/oauth2 submodule"
                (cd usermgmt/oauth2 && go test ./... -count=1 -race)
                echo "==> adminui submodule"
                (cd adminui && go test ./... -count=1 -race)
                echo "==> loginpage submodule"
                (cd loginpage && go test ./... -count=1 -race)
                echo "==> dashboardui submodule"
                (cd dashboardui && go test ./... -count=1 -race)
                echo "==> integration_test submodule"
                (cd integration_test && go test ./... -count=1 -race)
              '';
            };

            test-race = goApp {
              name = "run-tests-race";
              description = "Run all Go tests with the race detector across all modules";
              text = ''
                echo "==> Root module"
                go test ./... -count=1 -race
                echo "==> identity-model submodule"
                (cd identity-model && go test ./... -count=1 -race)
                echo "==> adminui submodule"
                (cd adminui && go test ./... -count=1 -race)
                echo "==> loginpage submodule"
                (cd loginpage && go test ./... -count=1 -race)
                echo "==> dashboardui submodule"
                (cd dashboardui && go test ./... -count=1 -race)
                echo "==> usermgmt submodule"
                (cd usermgmt && go test ./... -count=1 -race)
                echo "==> integration_test submodule"
                (cd integration_test && go test ./... -count=1 -race)
              '';
            };

            test-flake = goApp {
              name = "run-tests-flake";
              description = "Run all Go tests 3x with race detector to detect flaky tests";
              text = ''
                echo "==> Root module (3 iterations)"
                go test ./... -count=3 -race
                echo "==> identity-model submodule (3 iterations)"
                (cd identity-model && go test ./... -count=3 -race)
                echo "==> usermgmt submodule (3 iterations)"
                (cd usermgmt && go test ./... -count=3 -race)
                echo "==> usermgmt/totp submodule (3 iterations)"
                (cd usermgmt/totp && go test ./... -count=3 -race)
                echo "==> usermgmt/webauthn submodule (3 iterations)"
                (cd usermgmt/webauthn && go test ./... -count=3 -race)
                echo "==> usermgmt/oauth2 submodule (3 iterations)"
                (cd usermgmt/oauth2 && go test ./... -count=3 -race)
                echo "==> adminui submodule (3 iterations)"
                (cd adminui && go test ./... -count=3 -race)
                echo "==> dashboardui submodule (3 iterations)"
                (cd dashboardui && go test ./... -count=3 -race)
                echo "==> integration_test submodule (3 iterations)"
                (cd integration_test && go test ./... -count=3 -race)
              '';
            };

            test-fuzz = goApp {
              name = "run-tests-fuzz";
              description = "Run all Go fuzz tests across all modules (FUZZTIME env var, default 30s)";
              text = ''
                FUZZTIME="''${FUZZTIME:-30s}"

                echo "==> Root module fuzz tests"
                for fuzz in $(go test -run='^$' -list='Fuzz.*' ./... | grep '^Fuzz' || true); do
                  echo "    -> $fuzz"
                  go test -run='^$' -fuzz="$fuzz" -fuzztime="$FUZZTIME" ./...
                done

                echo "==> identity-model submodule fuzz tests"
                (cd identity-model && for fuzz in $(go test -run='^$' -list='Fuzz.*' ./... | grep '^Fuzz' || true); do
                  echo "    -> $fuzz"
                  go test -run='^$' -fuzz="$fuzz" -fuzztime="$FUZZTIME" ./...
                done)

                echo "==> adminui submodule fuzz tests"
                (cd adminui && for fuzz in $(go test -run='^$' -list='Fuzz.*' ./... | grep '^Fuzz' || true); do
                  echo "    -> $fuzz"
                  go test -run='^$' -fuzz="$fuzz" -fuzztime="$FUZZTIME" ./...
                done)

                echo "==> usermgmt submodule fuzz tests"
                (cd usermgmt && for fuzz in $(go test -run='^$' -list='Fuzz.*' ./... | grep '^Fuzz' || true); do
                  echo "    -> $fuzz"
                  go test -run='^$' -fuzz="$fuzz" -fuzztime="$FUZZTIME" ./...
                done)

                echo "==> usermgmt/totp submodule fuzz tests"
                (cd usermgmt/totp && for fuzz in $(go test -run='^$' -list='Fuzz.*' ./... | grep '^Fuzz' || true); do
                  echo "    -> $fuzz"
                  go test -run='^$' -fuzz="$fuzz" -fuzztime="$FUZZTIME" ./...
                done)

                echo "==> usermgmt/webauthn submodule fuzz tests"
                (cd usermgmt/webauthn && for fuzz in $(go test -run='^$' -list='Fuzz.*' ./... | grep '^Fuzz' || true); do
                  echo "    -> $fuzz"
                  go test -run='^$' -fuzz="$fuzz" -fuzztime="$FUZZTIME" ./...
                done)

                echo "==> usermgmt/oauth2 submodule fuzz tests"
                (cd usermgmt/oauth2 && for fuzz in $(go test -run='^$' -list='Fuzz.*' ./... | grep '^Fuzz' || true); do
                  echo "    -> $fuzz"
                  go test -run='^$' -fuzz="$fuzz" -fuzztime="$FUZZTIME" ./...
                done)

                echo "==> dashboardui submodule fuzz tests"
                (cd dashboardui && for fuzz in $(go test -run='^$' -list='Fuzz.*' ./... | grep '^Fuzz' || true); do
                  echo "    -> $fuzz"
                  go test -run='^$' -fuzz="$fuzz" -fuzztime="$FUZZTIME" ./...
                done)

                echo "==> integration_test submodule fuzz tests"
                (cd integration_test && for fuzz in $(go test -run='^$' -list='Fuzz.*' ./... | grep '^Fuzz' || true); do
                  echo "    -> $fuzz"
                  go test -run='^$' -fuzz="$fuzz" -fuzztime="$FUZZTIME" ./...
                done)
              '';
            };

            lint = goApp {
              name = "run-lint";
              description = "Run golangci-lint across all modules";
              runtimeInputs = [ pkgs.golangci-lint ];
              text = ''
                echo "==> Root module"
                golangci-lint run
                echo "==> identity-model submodule"
                (cd identity-model && golangci-lint run)
                echo "==> usermgmt submodule"
                (cd usermgmt && golangci-lint run)
                echo "==> usermgmt/totp submodule"
                (cd usermgmt/totp && golangci-lint run)
                echo "==> usermgmt/webauthn submodule"
                (cd usermgmt/webauthn && golangci-lint run)
                echo "==> usermgmt/oauth2 submodule"
                (cd usermgmt/oauth2 && golangci-lint run)
                echo "==> adminui submodule"
                (cd adminui && golangci-lint run)
                echo "==> loginpage submodule"
                (cd loginpage && golangci-lint run)
                echo "==> dashboardui submodule"
                (cd dashboardui && golangci-lint run)
                echo "==> integration_test submodule"
                (cd integration_test && golangci-lint run)
              '';
            };

            coverage = goApp {
              name = "run-coverage";
              text = ''
                echo "==> Root module coverage"
                go test ./... -count=1 -coverprofile=coverage.out
                go tool cover -func=coverage.out
                echo "==> identity-model submodule coverage"
                (cd identity-model && go test ./... -count=1 -coverprofile=coverage.out && go tool cover -func=coverage.out)
                echo "==> usermgmt submodule coverage"
                (cd usermgmt && go test ./... -count=1 -coverprofile=coverage.out && go tool cover -func=coverage.out)
                echo "==> usermgmt/totp submodule coverage"
                (cd usermgmt/totp && go test ./... -count=1 -coverprofile=coverage.out && go tool cover -func=coverage.out)
                echo "==> usermgmt/webauthn submodule coverage"
                (cd usermgmt/webauthn && go test ./... -count=1 -coverprofile=coverage.out && go tool cover -func=coverage.out)
                echo "==> usermgmt/oauth2 submodule coverage"
                (cd usermgmt/oauth2 && go test ./... -count=1 -coverprofile=coverage.out && go tool cover -func=coverage.out)
                echo "==> adminui submodule coverage"
                (cd adminui && go test ./... -count=1 -coverprofile=coverage.out && go tool cover -func=coverage.out)
                echo "==> loginpage submodule coverage"
                (cd loginpage && go test ./... -count=1 -coverprofile=coverage.out && go tool cover -func=coverage.out)
                echo "==> dashboardui submodule coverage"
                (cd dashboardui && go test ./... -count=1 -coverprofile=coverage.out && go tool cover -func=coverage.out)
              '';
            };

            build = goApp {
              name = "run-build";
              text = ''
                echo "==> Root module"
                go build ./...
                echo "==> identity-model submodule"
                (cd identity-model && go build ./...)
                echo "==> usermgmt submodule"
                (cd usermgmt && go build ./...)
                echo "==> usermgmt/totp submodule"
                (cd usermgmt/totp && go build ./...)
                echo "==> usermgmt/webauthn submodule"
                (cd usermgmt/webauthn && go build ./...)
                echo "==> usermgmt/oauth2 submodule"
                (cd usermgmt/oauth2 && go build ./...)
                echo "==> adminui submodule"
                (cd adminui && go build ./...)
                echo "==> loginpage submodule"
                (cd loginpage && go build ./...)
                echo "==> dashboardui submodule"
                (cd dashboardui && go build ./...)
                echo "==> integration_test submodule"
                (cd integration_test && go build ./...)
                echo "==> datastar-demo example"
                (cd examples/datastar-demo && go build ./...)
                echo "==> admin-demo example"
                (cd examples/admin-demo && go build ./...)
                echo "==> basic example"
                (cd examples/basic && go build ./...)
                echo "==> dashboard-demo example"
                (cd examples/dashboard-demo && go build ./...)
                echo "All modules built successfully."
              '';
            };

            test-root = goApp {
              name = "test-root";
              description = "Run the root module's Go tests in isolation";
              text = ''
                go test ./... -count=1 -race "$@"
              '';
            };

            test-usermgmt = goApp {
              name = "test-usermgmt";
              description = "Run the usermgmt submodule's Go tests in isolation";
              text = ''
                cd usermgmt
                go test ./... -count=1 -race "$@"
              '';
            };

            test-adminui = goApp {
              name = "test-adminui";
              description = "Run the adminui submodule's Go tests in isolation";
              text = ''
                cd adminui
                go test ./... -count=1 -race "$@"
              '';
            };

            test-loginpage = goApp {
              name = "test-loginpage";
              description = "Run the loginpage submodule's Go tests in isolation";
              text = ''
                cd loginpage
                go test ./... -count=1 -race "$@"
              '';
            };

            test-dashboardui = goApp {
              name = "test-dashboardui";
              description = "Run the dashboardui submodule's Go tests in isolation";
              text = ''
                cd dashboardui
                go test ./... -count=1 -race "$@"
              '';
            };

            test-integration = goApp {
              name = "test-integration";
              description = "Run the integration_test module's Go tests in isolation";
              text = ''
                cd integration_test
                go test ./... -count=1 -race "$@"
              '';
            };

            test-totp = goApp {
              name = "test-totp";
              description = "Run the usermgmt/totp submodule's Go tests in isolation";
              text = ''
                cd usermgmt/totp
                go test ./... -count=1 -race "$@"
              '';
            };

            test-webauthn = goApp {
              name = "test-webauthn";
              description = "Run the usermgmt/webauthn submodule's Go tests in isolation";
              text = ''
                cd usermgmt/webauthn
                go test ./... -count=1 -race "$@"
              '';
            };

            test-oauth2 = goApp {
              name = "test-oauth2";
              description = "Run the usermgmt/oauth2 submodule's Go tests in isolation";
              text = ''
                cd usermgmt/oauth2
                go test ./... -count=1 -race "$@"
              '';
            };

            build-datastar-demo = goApp {
              name = "build-datastar-demo";
              description = "Build the datastar-demo example binary";
              text = ''
                cd examples/datastar-demo
                go build ./... "$@"
              '';
            };

            build-admin-demo = goApp {
              name = "build-admin-demo";
              description = "Build the admin-demo example binary (runnable admin panel showcase)";
              text = ''
                cd examples/admin-demo
                go build ./... "$@"
              '';
            };

            build-dashboard-demo = goApp {
              name = "build-dashboard-demo";
              description = "Build the dashboard-demo example binary";
              text = ''
                cd examples/dashboard-demo
                go build ./... "$@"
              '';
            };

            build-catalog-demo = goApp {
              name = "build-catalog-demo";
              description = "Build the catalog-demo example binary";
              text = ''
                cd examples/catalog-demo
                go build ./... "$@"
              '';
            };

            build-adminui-css = {
              type = "app";
              meta.description = "Compile adminui Tailwind v4 CSS (tailwind.css → assets/admin-tw.css)";
              program = pkgs.writeShellApplication {
                name = "build-adminui-css";
                runtimeInputs = [
                  pkgs.tailwindcss_4
                  goPkg
                ];
                text = ''
                  cd adminui
                  # Resolve templ-components module dir at build time.
                  TC_DIR=$(GOWORK=off go list -m -f '{{.Dir}}' github.com/larsartmann/templ-components 2>/dev/null || true)

                  TMP_CSS=$(mktemp --suffix=.css)
                  cp tailwind.css "$TMP_CSS"

                  # Add @source for adminui itself (the temp CSS lives in
                  # /tmp so Tailwind's auto-detection won't find our .templ
                  # files without this).
                  ADMINUI_DIR=$(pwd)
                  echo "@source \"$ADMINUI_DIR\";" >> "$TMP_CSS"

                  if [ -n "$TC_DIR" ]; then
                    # Copy ONLY .templ files to a temp dir.
                    # _templ.go are generated mirrors (same class strings,
                    # 3x larger). Scanning the full module cache causes
                    # 55 GB RAM usage — this approach uses <500 MB.
                    SCAN_DIR=$(mktemp -d)
                    for pkg in display errorpage feedback forms htmx icons layout navigation; do
                      if [ -d "$TC_DIR/$pkg" ]; then
                        cp "$TC_DIR/$pkg/"*.templ "$SCAN_DIR/" 2>/dev/null || true
                      fi
                    done
                    echo "@source \"$SCAN_DIR\";" >> "$TMP_CSS"
                  fi

                  tailwindcss -i "$TMP_CSS" -o assets/admin-tw.css --minify

                  rm -f "$TMP_CSS"
                  [ -n "''${SCAN_DIR:-}" ] && rm -rf "$SCAN_DIR"
                  echo "Done: adminui/assets/admin-tw.css"
                '';
              };
            };

            gen = {
              type = "app";
              meta.description = "Regenerate adminui templ components and normalize formatting";
              program = pkgs.writeShellApplication {
                name = "templ-generate";
                runtimeInputs = [
                  goPkg
                  pkgs.templ
                ];
                text = ''
                  cd adminui
                  templ generate
                  gofmt -w *_templ.go
                  echo "Done: adminui templ components regenerated and formatted"
                '';
              };
            };

            render-diagrams = {
              type = "app";
              meta.description = "Render all .d2 source files under docs/ to SVG (dark canvas → theme 200, light → default)";
              program = pkgs.writeShellApplication {
                name = "render-diagrams";
                runtimeInputs = [ pkgs.d2 ];
                text = ''
                  shopt -s nullglob
                  found=0
                  for file in docs/**/*.d2 docs/*.d2; do
                    [ -f "$file" ] || continue
                    found=1
                    out="''${file%.d2}.svg"
                    if sed -n '1,10p' "$file" | grep -qE 'style:\s*\{[^}]*fill:\s*"#[01][0-9a-fA-F]{5}"'; then
                      echo "[dark]  $file"
                      d2 --layout=elk --theme=200 "$file" "$out"
                    else
                      echo "[light] $file"
                      d2 --layout=elk "$file" "$out"
                    fi
                  done
                  if [ "$found" -eq 0 ]; then
                    echo "No .d2 files found under docs/"
                    exit 1
                  fi
                '';
              };
            };

            errorfamily = {
              type = "app";
              meta.description = "Verify all errors use go-error-family constructors (no stdlib errors.New/fmt.Errorf/errors.Join)";
              program = pkgs.writeShellApplication {
                name = "check-errorfamily";
                runtimeInputs = [ pkgs.ripgrep ];
                text = ''
                  # Root + usermgmt + adminui + identity-model + dashboardui + loginpage:
                  # error-family constructors are mandatory in non-test code.
                  # Auth sub-modules (totp/webauthn/oauth2) are intentionally exempt:
                  # they don't import go-cqrs-lite/event/v4 (keeping deps minimal), and
                  # the Service layer wraps all provider errors with event.Wrapf at the
                  # boundary — so error families are assigned at the correct layer.
                  set -euo pipefail

                  check_module() {
                    local dir="$1"
                    local name="$2"
                    echo "==> $name"
                    local violations
                    violations=$(rg --glob='*.go' --glob='!*_test.go' --glob='!*_templ.go' \
                      --glob='!vendor/' --glob='!examples/' \
                      'errors\.New\(|fmt\.Errorf\(|errors\.Join\(' "$dir" 2>/dev/null || true)
                    if [ -n "$violations" ]; then
                      echo "FAIL: stdlib error constructors found in $name:"
                      echo "$violations"
                      exit 1
                    fi
                    echo "  OK"
                  }

                  check_module "." "Root module"
                  check_module "usermgmt" "usermgmt submodule"
                  check_module "adminui" "adminui submodule"
                  check_module "identity-model" "identity-model submodule"
                  check_module "dashboardui" "dashboardui submodule"
                  check_module "loginpage" "loginpage submodule"

                  echo "All modules pass errorfamily check."
                '';
              };
            };

            check-modules = {
              type = "app";
              meta.description = "Run all module architecture checks (isolation, dep budgets, version drift, replace directives)";
              program = pkgs.writeShellApplication {
                name = "check-modules";
                runtimeInputs = [ goPkg ];
                text = ''
                  cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                  bash scripts/check-module-isolation.sh
                  bash scripts/check-dep-budgets.sh
                  bash scripts/check-version-drift.sh --strict
                  bash scripts/check-replace-directives.sh
                  bash scripts/check-docs-freshness.sh
                  echo ""
                  echo "✓ All module architecture checks passed"
                '';
              };
            };

            check-docs-freshness = {
              type = "app";
              meta.description = "Scan .md files for version strings that don't match go.mod";
              program = pkgs.writeShellApplication {
                name = "check-docs-freshness";
                runtimeInputs = [ goPkg ];
                text = ''
                  cd "''${BUILD_ROOT:-$(git rev-parse --show-toplevel)}"
                  bash scripts/check-docs-freshness.sh
                '';
              };
            };

            check-codegen = {
              type = "app";
              meta.description = "Verify adminui + loginpage _templ.go files match .templ sources (no codegen drift)";
              program = pkgs.writeShellApplication {
                name = "check-codegen";
                runtimeInputs = [
                  goPkg
                  pkgs.templ
                ];
                text = ''
                  for mod in adminui loginpage; do
                    echo "==> $mod"
                    (cd "$mod" && templ generate && gofmt -w *_templ.go)
                    if ! git diff --exit-code -- "$mod"/*_templ.go; then
                      echo ""
                      echo "FAIL: Generated _templ.go files in $mod differ from committed versions."
                      echo "Run 'templ generate' in $mod/ and commit the result."
                      exit 1
                    fi
                  done
                  echo "Codegen drift check PASSED"
                '';
              };
            };

            coverage-gate = goApp {
              name = "coverage-gate";
              description = "Run tests and fail if coverage drops below thresholds";
              runtimeInputs = [ pkgs.bc ];
              text = ''
                fail=0
                check_cov() {
                  local mod="$1" threshold="$2"
                  local cov
                  cov=$(cd "$mod" && go test ./... -count=1 -coverprofile=/tmp/cov >/dev/null 2>&1 && go tool cover -func=/tmp/cov | tail -1 | grep -oP '\d+\.\d+(?=%)')
                  echo "$mod coverage: ''${cov}% (threshold: ''${threshold}%)"
                  if (( $(echo "$cov < $threshold" | bc -l) )); then
                    echo "FAIL: $mod coverage ''${cov}% < ''${threshold}%"
                    fail=1
                  fi
                }
                check_cov . 90
                check_cov identity-model 70
                check_cov usermgmt 74
                check_cov usermgmt/totp 80
                check_cov usermgmt/webauthn 80
                check_cov usermgmt/oauth2 80
                check_cov adminui 66
                check_cov loginpage 79
                check_cov dashboardui 60
                if [ "$fail" -eq 1 ]; then
                  echo "Coverage gate FAILED"
                  exit 1
                fi
                echo "Coverage gate PASSED"
              '';
            };

            release-checklist = {
              type = "app";
              meta.description = "Pre-release verification: CHANGELOG, versions, builds, git status";
              program = pkgs.writeShellApplication {
                name = "release-checklist";
                runtimeInputs = [ goPkg ];
                text = ''
                  bash scripts/release-checklist.sh
                '';
              };
            };

            e2e = {
              type = "app";
              meta.description = "Run Playwright E2E tests (offline sync) against the local Go test server";
              program = pkgs.writeShellApplication {
                name = "e2e";
                runtimeInputs = [
                  goPkg
                  pkgs.nodejs
                  pkgs.curl
                ]
                ++ pkgs.lib.optional (pkgs ? chromium) pkgs.chromium;
                text = ''
                  export GOEXPERIMENT=jsonv2
                  # On NixOS, Playwright's downloaded Chromium cannot run (no FHS linker).
                  # Use the Nix-packaged Chromium via E2E_BROWSER_PATH.
                  if [ -z "''${E2E_BROWSER_PATH:-}" ] && command -v chromium >/dev/null 2>&1; then
                    E2E_BROWSER_PATH="$(command -v chromium)"
                    export E2E_BROWSER_PATH
                  fi
                  cd "''${BUILD_ROOT:-$(pwd)}"
                  echo "==> Building E2E test server"
                  (cd e2e/server && go build -o /tmp/cqrs-htmx-e2e-server .)

                  echo "==> Starting E2E test server"
                  /tmp/cqrs-htmx-e2e-server &
                  SERVER_PID=$!

                  cleanup() {
                    kill "$SERVER_PID" 2>/dev/null || true
                    wait "$SERVER_PID" 2>/dev/null || true
                  }
                  trap cleanup EXIT

                  sleep 1

                  if ! curl -sf http://localhost:18923/ >/dev/null 2>&1; then
                    echo "FAIL: E2E server did not start on :18923"
                    exit 1
                  fi

                  echo "==> Running Playwright tests"
                  cd e2e

                  if command -v bun >/dev/null 2>&1; then
                    bun install --frozen-lockfile 2>/dev/null || bun install
                    bun run test
                  elif command -v npx >/dev/null 2>&1; then
                    npx playwright install chromium
                    npx playwright test
                  else
                    echo "FAIL: Neither bun nor npx found. Install Node.js or Bun to run E2E tests."
                    exit 1
                  fi
                '';
              };
            };
          };
        };
    };
}
