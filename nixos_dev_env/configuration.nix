{ config, pkgs, ... }:

{
  boot.loader.systemd-boot.enable = true;
  boot.loader.efi.canTouchEfiVariables = true;

  networking.hostName = "nixos";
  networking.networkmanager.enable = true;
  networking.firewall.interfaces.netbird0.allowedTCPPorts = [
    4096
    8080
    8081
  ];

  services.openssh.enable = false;

  services.netbird.clients.default = {
    name = "netbird";
    interface = "netbird0";
    port = 51820;
    hardened = false;
    config = {
      ServerSSHAllowed = true;
      DisableSSHAuth = false;
      EnableSSHRoot = false;
      EnableSSHSFTP = false;
      EnableSSHLocalPortForwarding = false;
      EnableSSHRemotePortForwarding = false;
    };
    login = {
      enable = true;
      setupKeyFile = "/etc/netbird/setup-key";
    };
  };

  # The setup key is needed only for initial enrollment, not normal startup.
  systemd.services.netbird-login.unitConfig.ConditionPathExists = "/etc/netbird/setup-key";

  environment.etc."opencode/opencode.json".source = ./opencode.json;

  systemd.services.opencode = {
    description = "OpenCode web interface";
    after = [ "network-online.target" ];
    wants = [ "network-online.target" ];
    wantedBy = [ "multi-user.target" ];
    unitConfig.ConditionPathExists = "/etc/opencode/env";
    path = with pkgs; [
      git
      gh
      nodejs_24
      nix
      nixd
      nixfmt
      ripgrep
    ];

    environment = {
      HOME = "/home/balaur";
      OPENCODE_CONFIG = "/etc/opencode/opencode.json";
      OPENCODE_ENABLE_EXA = "1";
    };

    serviceConfig = {
      User = "balaur";
      Group = "users";
      WorkingDirectory = "/home/balaur/projects";
      ExecStart = "${pkgs.opencode}/bin/opencode web --hostname 0.0.0.0 --port 4096";
      EnvironmentFile = "-/etc/opencode/env";
      Restart = "on-failure";
      RestartSec = "5s";
      UMask = "0077";

      NoNewPrivileges = false;
      PrivateTmp = true;
      ProtectHome = "tmpfs";
      ProtectSystem = "strict";
      # Share the same OpenCode state, credentials, and tool configuration as the TUI.
      BindPaths = [ "/home/balaur" ];
      ReadWritePaths = [
        "/nix/var"
        "/nix/store"
      ];
    };
  };

  systemd.services.balaur-main = {
    description = "Balaur main instance";
    after = [ "network-online.target" ];
    wants = [ "network-online.target" ];
    wantedBy = [ "multi-user.target" ];

    serviceConfig = {
      User = "balaur";
      Group = "users";
      WorkingDirectory = "/home/balaur/projects/balaur";
      ExecStart = "${pkgs.nodejs_24}/bin/node server.mjs";
      Environment = [
        "HOST=0.0.0.0"
        "PORT=8080"
      ];
      Restart = "on-failure";
      RestartSec = "5s";

      NoNewPrivileges = true;
      PrivateTmp = true;
      ProtectSystem = "strict";
      ProtectHome = "tmpfs";
      BindPaths = [ "/home/balaur/projects/balaur" ];
    };
  };

  systemd.services.balaur-dev = {
    description = "Balaur development instance";
    after = [ "network-online.target" ];
    wants = [ "network-online.target" ];
    wantedBy = [ "multi-user.target" ];

    serviceConfig = {
      User = "balaur";
      Group = "users";
      WorkingDirectory = "/home/balaur/projects/balaur";
      ExecStart = "${pkgs.nodejs_24}/bin/node server.mjs";
      Environment = [
        "HOST=0.0.0.0"
        "PORT=8081"
      ];
      Restart = "on-failure";
      RestartSec = "5s";

      NoNewPrivileges = true;
      PrivateTmp = true;
      ProtectSystem = "strict";
      ProtectHome = "tmpfs";
      BindPaths = [ "/home/balaur/projects/balaur" ];
    };
  };

  time.timeZone = "Europe/Bucharest";

  i18n.defaultLocale = "en_US.UTF-8";
  i18n.extraLocaleSettings = {
    LC_ADDRESS = "ro_RO.UTF-8";
    LC_IDENTIFICATION = "ro_RO.UTF-8";
    LC_MEASUREMENT = "ro_RO.UTF-8";
    LC_MONETARY = "ro_RO.UTF-8";
    LC_NAME = "ro_RO.UTF-8";
    LC_NUMERIC = "ro_RO.UTF-8";
    LC_PAPER = "ro_RO.UTF-8";
    LC_TELEPHONE = "ro_RO.UTF-8";
    LC_TIME = "ro_RO.UTF-8";
  };

  services.xserver.xkb = {
    layout = "us";
    variant = "";
  };

  users.users.balaur = {
    isNormalUser = true;
    description = "balaur";
    extraGroups = [
      "networkmanager"
      "wheel"
    ];
    packages = with pkgs; [ ];
  };

  security.sudo.wheelNeedsPassword = false;

  nixpkgs.config.allowUnfree = true;
  nix.settings.experimental-features = [
    "nix-command"
    "flakes"
  ];

  environment.systemPackages = with pkgs; [
    opencode
    nodejs_24
    nixd
    nixfmt
    git
    gh
  ];

  system.stateVersion = "26.05";
}
