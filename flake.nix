{
  description = "Go development and Supervisor environment with FHS Nginx";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, utils }:
    utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        # Go Toolchain
        goEnv = pkgs.go_1_26;

        # Define FHS wrapper using the updated Nixpkgs function name
        FHS = pkgs.buildFHSEnv {
          name = "fhs";
          targetPkgs = pkgs: [ pkgs.nginx ];
        };

      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = [
            goEnv
            pkgs.gopls
            pkgs.gotools
            pkgs.golangci-lint
            pkgs.delve
            pkgs.python3Packages.supervisor
            
            # Include the FHS environment directly
            FHS
          ];

          shellHook = ''
            fhs
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
command=bash -c "while true; do echo \"Tick \$(date)\"; sleep 5; done"
stdout_logfile=./dummy_loop.log
stderr_logfile=./dummy_loop.err
autostart=false
autorestart=true
EOF
            echo "Created supervisor.conf"'

            alias start-super='supervisord -c supervisor.conf'
            alias super-shell='supervisorctl -c supervisor.conf'
          '';
        };

        packages.default = pkgs.buildGoModule {
          pname = "taskmaster";
          version = "0.1.0";
          src = ./cmd/taskmaster;
          vendorHash = "g3xhqpq09nnldk02g9czrlapd0qsrlws"; 
        };
      });
}
