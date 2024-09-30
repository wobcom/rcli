{
  description = "rcli";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/master";

  outputs = { self, nixpkgs }:
    let
      systems =
        [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f system);

      # Memoize nixpkgs for different platforms for efficiency.
      nixpkgsFor = forAllSystems (system:
        import nixpkgs {
          inherit system;
          overlays = [ self.overlays.default ];
        });

    in {
      overlays.default = final: prev: {
        rcli = final.callPackage (
          { buildGoModule }:
            buildGoModule {
              pname = "rcli";
              version = "0.1.0";
              src = self;
              vendorHash = "sha256-TOJJfgTvXl/5HGcBM+MgYAX9GDzhKQoTYxEek+TlIys=";
              CGO_ENABLED = 0;
            }
          ) { };
      };

      packages =
        forAllSystems (system: { inherit (nixpkgsFor.${system}) rcli; });
      defaultPackage = forAllSystems (system: self.packages.${system}.rcli);
    };
}