{
  description = "balaur's NixOS configuration";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    opencode = {
      url = "github:anomalyco/opencode/v1.18.4";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    { nixpkgs, opencode, ... }:
    {
      nixosConfigurations.nixos = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          { nixpkgs.overlays = [ opencode.overlays.default ]; }
          ./configuration.nix
          ./hardware-configuration.nix
        ];
      };
    };
}
