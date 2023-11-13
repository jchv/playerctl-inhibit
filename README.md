[![Go Report Card](https://goreportcard.com/badge/github.com/jchv/playerctl-inhibit)](https://goreportcard.com/report/github.com/jchv/playerctl-inhibit) [![CI](https://github.com/jchv/playerctl-inhibit/actions/workflows/ci.yaml/badge.svg)](https://github.com/jchv/playerctl-inhibit/actions/workflows/ci.yaml)

# playerctl-inhibit
This is a small Go daemon that connects to playerctld over dbus (via MPRIS, using [go-mpris](https://github.com/leberKleber/go-mpris)) and signals logind to inhibit suspend when media is playing.

This small program is just meant to be a stop-gap solution for mobile computers running Linux so that they can stay awake when music is playing, even if, for example, the lid is closed. (In the future it might be possible to extend this to determine other conditions in which suspend should be inhibited, e.g. certain kinds of devices being connected. The program may be renamed at that point.)

To accomplish this, it polls playerctld each second to check the playback status, so it could probably be made more efficient, however, it does not seem to consume significant resources regardless.

# NixOS
A Nix flake is included. It includes a NixOS flake with a module that will automatically configure a systemd user service to run `playerctl-inhibit`. Note that if `playerctld` is not somehow configured to run already, this will just continually try to find `playerctld` and fail - it does not try to poll media players. To use the NixOS module, just include it; it does not currently have any options.

In your system configuration flake, it might look something like this:

```nix
{
  inputs = {
    # ... other inputs ...
    playerctl-inhibit.url = "/home/john/Code/playerctl-inhibit";
    playerctl-inhibit.inputs.nixpkgs.follows = "nixpkgs";
  };
  outputs = { self, nixpkgs, playerctl-inhibit, ... }: {
    nixosConfiguration.my-machine = nixpkgs.lib.nixosSystem {
      # ... other configuration
      modules = [
        playerctl-inhibit.nixosModules.playerctl-inhibit
      ]
    };
  };
}
```

> [!WARNING]
> This interface should be considered unstable. It may change in the future, for example, to introduce configuration options. The default of automatically setting up a systemd module for playerctl-inhibit may change.
