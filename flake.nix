{
  description = "Development enviroment for go";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    pre-commit-hooks.url = "github:cachix/git-hooks.nix";
  };

  outputs = inputs: let
    version = "0.1.0"; # x-release-please-version
    goVersion = 24;
    lastModified = inputs.self.lastModifiedDate;
    buildTimestamp = "${builtins.substring 0 4 lastModified}-${builtins.substring 4 2 lastModified}-${builtins.substring 6 2 lastModified}T${builtins.substring 8 2 lastModified}:${builtins.substring 10 2 lastModified}:${builtins.substring 12 2 lastModified}Z";
    supportedSystems = ["x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin"];
    forEachSupportedSystem = f:
      inputs.nixpkgs.lib.genAttrs supportedSystems (system:
        f {
          pkgs = import inputs.nixpkgs {
            inherit system;
            overlays = [inputs.self.overlays.default];
          };
        });
  in {
    overlays.default = final: prev: {
      go = final."go_1_${toString goVersion}";
    };

    packages = forEachSupportedSystem ({pkgs}: {
      default = pkgs.buildGoModule {
        pname = "chat-room";
        inherit version;
        src = ./.;
        vendorHash = "sha256-vo1uKHYq8IdEX0YDsIKoRQ0t+RFby74STEhdi3zRh+8=";
        subPackages = ["cmd/chat-room"];
        ldflags = [
          "-X github.com/choffmann/chat-room/internal/config.Version=v${version}"
          "-X github.com/choffmann/chat-room/internal/config.GitCommit=${inputs.self.shortRev or inputs.self.dirtyShortRev or "dirty"}"
          "-X github.com/choffmann/chat-room/internal/config.GitRepository=https://github.com/choffmann/chat-room"
          "-X github.com/choffmann/chat-room/internal/config.BuildTime=${buildTimestamp}"
        ];
      };
    });

    devShells = forEachSupportedSystem ({pkgs}: {
      default = pkgs.mkShell {
        packages = with pkgs; [
          go
          gotools
          golangci-lint

          gnumake
          pkg-config
          yq-go
        ];
      };
    });
  };
}
