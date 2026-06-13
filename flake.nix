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
        { config, pkgs, ... }:
        {
          treefmt = {
            projectRootFile = "go.mod";
            programs.nixfmt.enable = true;
            programs.templ.enable = true;
            programs.gofmt.enable = true;
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
                  echo "==> usermgmt submodule"
                  (cd usermgmt && go test ./... -count=1 -race)
                  echo "==> integration_test submodule"
                  (cd integration_test && go test ./... -count=1 -race)
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
          };

        };
    };
}
