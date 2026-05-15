{
  description = "A playground environment for Supervisor (supervisord/supervisorctl)";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, utils }:
    utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = [ pkgs.python3Packages.supervisor ];

          shellHook = ''
            echo "========================================================="
            echo "      Welcome to the Supervisor Playground!              "
            echo "========================================================="
            echo "Available commands:"
            echo "  gen-super-conf  -> Generate a local supervisor.conf"
            echo "  start-super     -> Start supervisord with local conf"
            echo "  super-shell     -> Open interactive supervisorctl shell"
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
      });
}
