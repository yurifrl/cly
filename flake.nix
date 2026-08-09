{
  description = "cly — personal CLI for session management, Zellij integration, and dev workflows";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = if (self ? shortRev) then self.shortRev else "dev";
      in
      {
        packages.cly = pkgs.buildGoModule {
          pname = "cly";
          inherit version;
          src = self;

          # go.mod requires go >= 1.25.8; unstable's default `go` tracks the
          # latest stable. Pin explicitly if the default ever lags behind.
          # go = pkgs.go_1_25;

          # Deterministic hash of the Go module dependencies. Computed by nix,
          # so it must be locked on a machine with nix: set this to
          # pkgs.lib.fakeHash, run `nix build .#cly`, and paste the hash nix
          # prints back here.
          vendorHash = "sha256-WmtDroNAMtHXVTH3tWqUFEq98+RbEfjkWQ2etDS/W8g=";

          # Root package -> `cly`; cmd/mcp -> `mcp`.
          subPackages = [ "." "cmd/mcp" ];

          # No CGO deps; modernc.org/sqlite is pure Go.
          env.CGO_ENABLED = "0";

          ldflags = [
            "-s"
            "-w"
            "-X=github.com/yurifrl/cly/cmd.Version=${version}"
            "-X=main.version=${version}"
          ];

          # The empty knownWebBundles list means there is no frontend to build;
          # all //go:embed targets are committed in-tree.
          doCheck = false;

          meta = with pkgs.lib; {
            description = "Personal CLI for session management, Zellij integration, and dev workflows";
            homepage = "https://github.com/yurifrl/cly";
            license = licenses.mit;
            mainProgram = "cly";
          };
        };

        packages.default = self.packages.${system}.cly;

        devShells.default = pkgs.mkShell {
          packages = [ pkgs.go pkgs.gopls pkgs.go-task ];
        };
      })
    // {
      overlays.default = final: prev: {
        cly = self.packages.${prev.system}.cly;
      };
    };
}
