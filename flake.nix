{
  description = "A merged playground for Go development and Supervisor (supervisord/supervisorctl)";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, utils }:
    utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        
        # Explicitly pins the Go version for consistency
        goEnv = pkgs.go_1_26;
      in
      {
        # 1. Unified Development Shell (`nix develop`)
        devShells.default = pkgs.mkShell {
          buildInputs = [
            # Go Toolchain & Ecosystem
            goEnv
            pkgs.gopls
            pkgs.gotools
            pkgs.golangci-lint
            pkgs.delve

            # Process Management
            pkgs.python3Packages.supervisor
          ];

          shellHook = ''
            echo "========================================================="
            echo "      Welcome to the Go & Supervisor Playground!         "
            echo "========================================================="
            echo "Go Version:  $(go version)"
            echo "========================================================="
            echo "Available Supervisor commands:"
            echo "  gen-super-conf  → Generate a local supervisor.conf"
            echo "  start-super     → Start supervisord with local conf"
            echo "  super-shell     → Open interactive supervisorctl shell"
            echo "========================================================="

            # Helper alias to generate a working local config
            alias gen-super-conf='cat << EOF > supervisor.conf
[supervisord]
logfile=./supervisord.log
pidfile=./supervisord.pid
nodaemon=false

[unix_http_server]
file=./supervisor.sock

[rpcinterface:supervisor]
supervisor.rpcinterface_factory = supervisor.rpcinterface:make_main_rpcinterface

[supervisorctl]
serverurl=unix://./supervisor.sock

[program:dummy_loop]
command=bash -c "while true; do echo \"Tick \$(date)\"; sleep 2; done"
stdout_logfile=./dummy_loop.log
stderr_logfile=./dummy_loop.err
autostart=false
autorestart=true
EOF
            echo "Created supervisor.conf with a sample dummy_loop program!"
            '

            # Helper alias to start the daemon using the local file
            alias start-super='supervisord -c supervisor.conf'

            # Helper alias to drop straight into the control shell
            alias super-shell='supervisorctl -c supervisor.conf'
          '';
        };

        # 2. Package configuration (`nix build`)
        packages.default = pkgs.buildGoModule {
          pname = "my-go-app"; # Change this to match your actual binary name
          version = "0.1.0";
          src = ./.;

          # Swap pkgs.lib.fakeHash with the true hash output by Nix on the first build failure
          vendorHash = pkgs.lib.fakeHash; 
        };
      });
}
