#!/usr/bin/env bash

# Derived from https://github.com/alphabetum/bash-boilerplate

# Strict Mode
set -o nounset

# Exit immediately if a pipeline returns non-zero.
#set -o errexit

# Print a helpful message if a pipeline with non-zero exit code causes the
# script to exit as described above.
trap 'echo "Aborting due to errexit on line $LINENO. Exit code: $?" >&2' ERR

# Allow the above trap be inherited by all functions in the script.
# Short form: set -E
set -o errtrace

# Return value of a pipeline is the value of the last (rightmost) command to
# exit with a non-zero status, or zero if all commands in the pipeline exit
# successfully.
set -o pipefail

# Set IFS to just newline and tab at the start
DEFAULT_IFS="${IFS}"
SAFER_IFS=$'\n\t'
IFS="${SAFER_IFS}"

###############################################################################
# Environment
###############################################################################

# $_ME
#
# Set to the program's basename.
_ME=$(basename "${0}")

###############################################################################
# Help
###############################################################################

# _print_help()
#
# Usage:
#   _print_help
#
# Print the program help information.
_print_help() {
  cat <<HEREDOC
MAINFLUX-DOCKER

Starts or stops Mainflux Docker composition.

Commands:
    start       Start Docker composition
    stop        Stop Docker composition

Options:
    -h, --help  Show this screen.
HEREDOC
}

###############################################################################
# Program Functions
###############################################################################

_start() {

  # Start NATS, Cassandra and Nginx
  printf "Starting NATS and Cassandra...\n\n"

  NB_DOCKERS=$(docker ps -a -f name=mainflux-nats -f name=mainflux-cassandra | wc -l)
  if [[ $NB_DOCKERS -lt 3 ]]
  then
    docker-compose -f docker/docker-compose.infrastructure.yml pull
    docker-compose -f docker/docker-compose.infrastructure.yml create
  fi
  docker-compose -f docker/docker-compose.infrastructure.yml start

  # Check if C* is alive
  printf "\nWaiting for Cassandra to start. This takes time, please be patient...\n"

  # Wait until Cassandra is ready to accept cqlsh commands
  # or timeout after 15 sec
  c_on=0
  retries=0
  while [[ $retries -lt 15 ]]
  do
    # Check if C* port 9042 is open
    if $(docker exec -it mainflux-cassandra cqlsh -e "exit" > /dev/null 2>&1)
    then
      # Sucess
      c_on=1
      break
    fi

    sleep 1
    printf "."
    retries=$((retries + 1))
  done

  # If Cassandra did not start then shut down everything and exit
  if [[ $c_on -eq 0 ]]
  then
    printf "\nCassandra did not start - shuting down everything.\n"
    docker-compose -f docker/docker-compose.infrastructure.yml stop
    exit 0
  else
    printf "OK\n"
  fi

  # Create C* keyspaces, if missing
  docker exec -it mainflux-cassandra cqlsh -e "CREATE KEYSPACE IF NOT EXISTS manager WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };"
  docker exec -it mainflux-cassandra cqlsh -e "CREATE KEYSPACE IF NOT EXISTS message_writer WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };"

  # Start Mainflux
  printf "\nStarting Mainflux composition...\n\n"

  NB_DOCKERS=$(docker ps -a -f name=mainflux-manager -f name=mainflux-http -f name=mainflux-mqtt -f name=mainflux-coap -f name=mainflux-message-writer | wc -l)
  if [[ $NB_DOCKERS -lt 6 ]]
  then
    docker-compose -f docker/docker-compose.mainflux.yml pull
    docker-compose -f docker/docker-compose.mainflux.yml create
  fi
  docker-compose -f docker/docker-compose.mainflux.yml start

  # Start Nginx
  printf "\nStarting Nginx...\n\n"

  NB_DOCKERS=$(docker ps -a -f name=nginx | wc -l)
  if [[ $NB_DOCKERS -lt 2 ]]
  then
    docker-compose -f docker/docker-compose.nginx.yml pull
    docker-compose -f docker/docker-compose.nginx.yml create
  fi
  docker-compose -f docker/docker-compose.nginx.yml start

  if [[ $? -ne 0 ]]
  then
    _stop
    exit 1
  fi

  printf "\n*** MAINFLUX IS ON ***\n\n"

  docker ps
}

_stop() {
  printf "\nStopping Nginx...\n\n"
  docker-compose -f docker/docker-compose.nginx.yml stop

  printf "Stopping Mainflux composition...\n\n"
  docker-compose -f docker/docker-compose.mainflux.yml stop

  printf "\nStopping NATS and Cassandra...\n\n"
  docker-compose -f docker/docker-compose.infrastructure.yml stop

  printf "\n*** MAINFLUX IS OFF ***\n\n"
}

_mainflux_docker() {
  if [[ $1 == "start" ]]
  then
    _start
  elif [[ $1 == "stop" ]]
  then
    _stop
  else
    printf "Unknown command.\n"
  fi
}

###############################################################################
# Main
###############################################################################

# _main()
#
# Usage:
#   _main [<options>] [<arguments>]
#
# Description:
#   Entry point for the program, handling basic option parsing and dispatching.
_main() {

  # No arguments provided
  if [[ $# -eq 0 ]] ; then
    _print_help
  fi

  # Avoid complex option parsing when only one program option is expected.
  if [[ "${1:-}" =~ ^-h|--help$  ]]
  then
    _print_help
  else
    _mainflux_docker "$@"
  fi
}

# Call `_main` after everything has been defined.
_main "$@"
