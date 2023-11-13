{
  description = "Daemon that inhibits suspend when media playback is reported via playerctld.";
  outputs = { self, nixpkgs, ... }:
    let 
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
      forEachSystem = nixpkgs.lib.genAttrs systems;
    in {
      overlays.default = final: prev: {
        playerctl-inhibit = final.buildGoModule {
          pname = "playerctl-inhibit";
          version = "0.0.0";
          src = ./.;
          vendorSha256 = "sha256-nxRYsiyq9zWc+e9uVqoG/NJViM1NenQF4c2LvhPIA9w=";
        };
      };
      packages = forEachSystem (system:
        let
          pkgs = (import nixpkgs {
            inherit system;
            overlays = [ self.overlays.default ];
          });
        in rec {
          inherit (pkgs) playerctl-inhibit;
          default = playerctl-inhibit;
        });
      nixosModules.playerctl-inhibit = { pkgs, ... }: {
        nixpkgs.overlays = [ self.overlays.default ];
        systemd.user.services."playerctl-inhibit" = {
          description = "inhibits sleep when media is playing";
          wantedBy = [ "graphical-session.target" ];
          partOf = [ "graphical-session.target" ];
          serviceConfig = {
            Restart = "on-failure";
            ExecStart = "${pkgs.playerctl-inhibit}/bin/playerctl-inhibit";
          };
        };
      };
    };
}