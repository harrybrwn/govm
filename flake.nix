{
  description = "A simple Go package";

  # Nixpkgs / NixOS version to use.
  inputs.nixpkgs.url = "nixpkgs/nixos-22.11";

  outputs = { self, nixpkgs }:
    let
      version = "0.0.1";
      lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";
      commit = if (self ? rev) then self.rev else "dirty";
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    {

      # Provide some binary packages for selected system types.
      packages = forAllSystems (system:
        let pkgs = nixpkgsFor.${system};
        in
        {
          govm = pkgs.buildGo120Module {
            pname = "govm";
            inherit version;
            src = ./.;
            ldflags = [
              "-s"
              "-w"
              "-X"
              "main.version=${version}"
              "-X"
              "main.commit=${commit}"
              "-X"
              "main.built=${lastModifiedDate}"
            ];
            #vendorSha256 = pkgs.lib.fakeSha256;
            vendorSha256 = "sha256-ucXY/yplVut6wvVRProB4l1Hcx8dCym0EC1hgiCRTZ8=";
          };
        });

      # Add dependencies that are only needed for development
      devShells = forAllSystems (system:
        let pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go_1_20
              gopls
              gotools
              go-tools
            ];
          };
        });

      # The default package for 'nix build'. This makes sense if the
      # flake provides only one package or there is a clear "main"
      # package.
      defaultPackage = forAllSystems (system: self.packages.${system}.govm);
    };
}
