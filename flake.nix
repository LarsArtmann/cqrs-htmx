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
            version = "3.1.0";

            dontUnpack = true;
            dontConfigure = true;
            dontBuild = true;
            dontInstall = true;

            meta = with lib; {
              description = "Go library for go-cqrs-lite with HTMX, templ, and Casbin authorization";
              homepage = "https://github.com/larsartmann/cqrs-htmx";
              license = licenses.mit;
              mainProgram = "cqrs-htmx";
              maintainers = [ ];
              platforms = platforms.unix;
            };
          };

          devShells = {
            default = pkgs.mkShellNoCC {
              packages = [
                pkgs.go_1_26
                pkgs.gopls
                pkgs.golangci-lint
              ];

              GOWORK = "off";
              GONOSUMCHECK = "github.com/larsartmann/*";
            };

            ci = pkgs.mkShellNoCC {
              packages = [
                pkgs.go_1_26
                pkgs.golangci-lint
              ];

              GOWORK = "off";
              GONOSUMCHECK = "github.com/larsartmann/*";
            };
          };

          checks = {
            formatting = config.treefmt.build.check self;
          };

          apps = {
            test = {
              type = "app";
              program = pkgs.writeShellApplication {
                name = "run-tests";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = ''
                  export GOWORK=off
                  export GONOSUMCHECK='github.com/larsartmann/*'
                  echo "==> Root module"
                  go test ./... -count=1 -race
                  echo "==> catalog submodule"
                  (cd catalog && go test ./... -count=1 -race)
                  echo "==> adminui submodule"
                  (cd adminui && go test ./... -count=1 -race)
                  echo "==> usermgmt submodule"
                  (cd usermgmt && go test ./... -count=1 -race)
                  echo "==> integration_test submodule"
                  (cd integration_test && go test ./... -count=1 -race)
                '';
              };
            };

            test-race = {
              type = "app";
              meta.description = "Run all Go tests with the race detector across all modules";
              program = pkgs.writeShellApplication {
                name = "run-tests-race";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = ''
                  export GOWORK=off
                  export GONOSUMCHECK='github.com/larsartmann/*'
                  echo "==> Root module"
                  go test ./... -count=1 -race
                  echo "==> catalog submodule"
                  (cd catalog && go test ./... -count=1 -race)
                  echo "==> adminui submodule"
                  (cd adminui && go test ./... -count=1 -race)
                  echo "==> usermgmt submodule"
                  (cd usermgmt && go test ./... -count=1 -race)
                  echo "==> integration_test submodule"
                  (cd integration_test && go test ./... -count=1 -race)
                '';
              };
            };

            test-fuzz = {
              type = "app";
              meta.description = "Run all Go fuzz tests across all modules (FUZZTIME env var, default 30s)";
              program = pkgs.writeShellApplication {
                name = "run-tests-fuzz";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = ''
                  export GOWORK=off
                  export GONOSUMCHECK='github.com/larsartmann/*'
                  FUZZTIME="''${FUZZTIME:-30s}"

                  echo "==> Root module fuzz tests"
                  for fuzz in $(go test -run='^$' -list='Fuzz.*' ./... | grep '^Fuzz' || true); do
                    echo "    -> $fuzz"
                    go test -run='^$' -fuzz="$fuzz" -fuzztime="$FUZZTIME" ./...
                  done

                  echo "==> catalog submodule fuzz tests"
                  (cd catalog && for fuzz in $(go test -run='^$' -list='Fuzz.*' ./... | grep '^Fuzz' || true); do
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

                  echo "==> integration_test submodule fuzz tests"
                  (cd integration_test && for fuzz in $(go test -run='^$' -list='Fuzz.*' ./... | grep '^Fuzz' || true); do
                    echo "    -> $fuzz"
                    go test -run='^$' -fuzz="$fuzz" -fuzztime="$FUZZTIME" ./...
                  done)
                '';
              };
            };

            lint = {
              type = "app";
              program = pkgs.writeShellApplication {
                name = "run-lint";
                runtimeInputs = [ pkgs.golangci-lint ];
                text = ''
                  echo "==> Root module"
                  golangci-lint run
                  echo "==> catalog submodule"
                  (cd catalog && golangci-lint run)
                  echo "==> adminui submodule"
                  (cd adminui && golangci-lint run)
                  echo "==> usermgmt submodule"
                  (cd usermgmt && golangci-lint run)
                '';
              };
            };

            coverage = {
              type = "app";
              program = pkgs.writeShellApplication {
                name = "run-coverage";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = ''
                  export GOWORK=off
                  export GONOSUMCHECK='github.com/larsartmann/*'
                  echo "==> Root module coverage"
                  go test ./... -count=1 -coverprofile=coverage.out
                  go tool cover -func=coverage.out
                  echo "==> catalog submodule coverage"
                  (cd catalog && go test ./... -count=1 -coverprofile=coverage.out && go tool cover -func=coverage.out)
                  echo "==> adminui submodule coverage"
                  (cd adminui && go test ./... -count=1 -coverprofile=coverage.out && go tool cover -func=coverage.out)
                  echo "==> usermgmt submodule coverage"
                  (cd usermgmt && go test ./... -count=1 -coverprofile=coverage.out && go tool cover -func=coverage.out)
                '';
              };
            };

            build = {
              type = "app";
              program = pkgs.writeShellApplication {
                name = "run-build";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = ''
                  export GOWORK=off
                  export GONOSUMCHECK='github.com/larsartmann/*'
                  echo "==> Root module"
                  go build ./...
                  echo "==> catalog submodule"
                  (cd catalog && go build ./...)
                  echo "==> adminui submodule"
                  (cd adminui && go build ./...)
                  echo "==> usermgmt submodule"
                  (cd usermgmt && go build ./...)
                  echo "==> integration_test submodule"
                  (cd integration_test && go build ./...)
                  echo "==> datastar-demo example"
                  (cd examples/datastar-demo && go build ./...)
                  echo "All modules built successfully."
                '';
              };
            };

            test-root = {
              type = "app";
              meta.description = "Run the root module's Go tests in isolation";
              program = pkgs.writeShellApplication {
                name = "test-root";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = ''
                  export GOWORK=off
                  export GONOSUMCHECK='github.com/larsartmann/*'
                  go test ./... -count=1 -race "$@"
                '';
              };
            };

            test-usermgmt = {
              type = "app";
              meta.description = "Run the usermgmt submodule's Go tests in isolation";
              program = pkgs.writeShellApplication {
                name = "test-usermgmt";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = ''
                  export GOWORK=off
                  export GONOSUMCHECK='github.com/larsartmann/*'
                  cd usermgmt
                  go test ./... -count=1 -race "$@"
                '';
              };
            };

            test-catalog = {
              type = "app";
              meta.description = "Run the catalog submodule's Go tests in isolation";
              program = pkgs.writeShellApplication {
                name = "test-catalog";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = ''
                  export GOWORK=off
                  export GONOSUMCHECK='github.com/larsartmann/*'
                  cd catalog
                  go test ./... -count=1 -race "$@"
                '';
              };
            };

            test-adminui = {
              type = "app";
              meta.description = "Run the adminui submodule's Go tests in isolation";
              program = pkgs.writeShellApplication {
                name = "test-adminui";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = ''
                  export GOWORK=off
                  export GONOSUMCHECK='github.com/larsartmann/*'
                  cd adminui
                  go test ./... -count=1 -race "$@"
                '';
              };
            };

            test-integration = {
              type = "app";
              meta.description = "Run the integration_test module's Go tests in isolation";
              program = pkgs.writeShellApplication {
                name = "test-integration";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = ''
                  export GOWORK=off
                  export GONOSUMCHECK='github.com/larsartmann/*'
                  cd integration_test
                  go test ./... -count=1 -race "$@"
                '';
              };
            };

            build-datastar-demo = {
              type = "app";
              meta.description = "Build the datastar-demo example binary";
              program = pkgs.writeShellApplication {
                name = "build-datastar-demo";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = ''
                  export GOWORK=off
                  cd examples/datastar-demo
                  go build ./... "$@"
                '';
              };
            };

            build-admin-demo = {
              type = "app";
              meta.description = "Build the admin-demo example binary (runnable admin panel showcase)";
              program = pkgs.writeShellApplication {
                name = "build-admin-demo";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = ''
                  export GOWORK=off
                  export GONOSUMCHECK='github.com/larsartmann/*'
                  cd examples/admin-demo
                  go build ./... "$@"
                '';
              };
            };

            build-catalog-demo = {
              type = "app";
              meta.description = "Build the catalog-demo example binary";
              program = pkgs.writeShellApplication {
                name = "build-catalog-demo";
                runtimeInputs = [ pkgs.go_1_26 ];
                text = ''
                  export GOWORK=off
                  cd examples/catalog-demo
                  go build ./... "$@"
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
                text = ''
                  echo "==> Root module"
                  branching-flow errorfamily .
                  echo "==> usermgmt submodule"
                  branching-flow errorfamily usermgmt
                  echo "==> adminui submodule"
                  branching-flow errorfamily adminui
                  echo "All modules pass errorfamily check."
                '';
              };
            };

            coverage-gate = {
              type = "app";
              meta.description = "Run tests and fail if coverage drops below thresholds (root 90%, usermgmt 75%, catalog 90%)";
              program = pkgs.writeShellApplication {
                name = "coverage-gate";
                runtimeInputs = [
                  pkgs.go_1_26
                  pkgs.bc
                ];
                text = ''
                  export GOWORK=off
                  export GONOSUMCHECK='github.com/larsartmann/*'
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
                  check_cov usermgmt 75
                  check_cov catalog 90
                  if [ "$fail" -eq 1 ]; then
                    echo "Coverage gate FAILED"
                    exit 1
                  fi
                  echo "Coverage gate PASSED"
                '';
              };
            };
          };

        };
    };
}
