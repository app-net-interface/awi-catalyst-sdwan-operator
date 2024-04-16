#!/bin/bash

###############################################################################
#
# demo-magic.sh
#
# Copyright (c) 2015 Paxton Hare
#
# This script lets you script demos in bash. It runs through your demo script when you press
# ENTER. It simulates typing and runs commands.
#
###############################################################################

# the speed to "type" the text
TYPE_SPEED=20

# no wait after "p" or "pe"
NO_WAIT=true

# if > 0, will pause for this amount of seconds before automatically proceeding with any p or pe
PROMPT_TIMEOUT=0

# don't show command number unless user specifies it
SHOW_CMD_NUMS=false


# handy color vars for pretty prompts
BLACK="\033[0;30m"
BLUE="\033[0;34m"
GREEN="\033[0;32m"
GREY="\033[0;90m"
CYAN="\033[0;36m"
RED="\033[0;31m"
PURPLE="\033[0;35m"
BROWN="\033[0;33m"
WHITE="\033[1;37m"
COLOR_RESET="\033[0m"

C_NUM=0

# prompt and command color which can be overridden
DEMO_PROMPT="$ "
DEMO_CMD_COLOR=$WHITE
DEMO_COMMENT_COLOR=$GREY

##
# prints the script usage
##
function usage() {
  echo -e ""
  echo -e "Usage: $0 [options]"
  echo -e ""
  echo -e "\tWhere options is one or more of:"
  echo -e "\t-h\tPrints Help text"
  echo -e "\t-d\tDebug mode. Disables simulated typing"
  echo -e "\t-n\tNo wait"
  echo -e "\t-w\tWaits max the given amount of seconds before proceeding with demo (e.g. '-w5')"
  echo -e ""
}

##
# wait for user to press ENTER
# if $PROMPT_TIMEOUT > 0 this will be used as the max time for proceeding automatically
##
function wait() {
  if [[ "$PROMPT_TIMEOUT" == "0" ]]; then
    read -rs
  else
    read -rst "$PROMPT_TIMEOUT"
  fi
}

##
# print command only. Useful for when you want to pretend to run a command
#
# takes 1 param - the string command to print
#
# usage: p "ls -l"
#
##
function p() {
  if [[ ${1:0:1} == "#" ]]; then
    cmd=$DEMO_COMMENT_COLOR$1$COLOR_RESET
  else
    cmd=$DEMO_CMD_COLOR$1$COLOR_RESET
  fi

  # render the prompt
  x=$(PS1="$DEMO_PROMPT" "$BASH" --norc -i </dev/null 2>&1 | sed -n '${s/^\(.*\)exit$/\1/p;}')

  # show command number is selected
  if $SHOW_CMD_NUMS; then
   printf "[$((++C_NUM))] $x"
  else
   printf "$x"
  fi

  # wait for the user to press a key before typing the command
  if [ $NO_WAIT = false ]; then
    wait
  fi

  if [[ -z $TYPE_SPEED ]]; then
    echo -en "$cmd"
  else
    echo -en "$cmd" | pv -qL $[$TYPE_SPEED+(-2 + RANDOM%5)];
  fi

  # wait for the user to press a key before moving on
  if [ $NO_WAIT = false ]; then
    wait
  fi
  echo ""
}

##
# Prints and executes a command
#
# takes 1 parameter - the string command to run
#
# usage: pe "ls -l"
#
##
function pe() {
  # print the command
  p "$@"
  run_cmd "$@"
}

##
# print and executes a command immediately
#
# takes 1 parameter - the string command to run
#
# usage: pei "ls -l"
#
##
function pei {
  NO_WAIT=true pe "$@"
}

##
# Enters script into interactive mode
#
# and allows newly typed commands to be executed within the script
#
# usage : cmd
#
##
function cmd() {
  # render the prompt
  x=$(PS1="$DEMO_PROMPT" "$BASH" --norc -i </dev/null 2>&1 | sed -n '${s/^\(.*\)exit$/\1/p;}')
  printf "$x\033[0m"
  read command
  run_cmd "${command}"
}

function run_cmd() {
  function handle_cancel() {
    printf ""
  }

  trap handle_cancel SIGINT
  stty -echoctl
  eval "$@"
  stty echoctl
  trap - SIGINT
}


function check_pv() {
  command -v pv >/dev/null 2>&1 || {

    echo ""
    echo -e "${RED}##############################################################"
    echo "# HOLD IT!! I require pv but it's not installed.  Aborting." >&2;
    echo -e "${RED}##############################################################"
    echo ""
    echo -e "${COLOR_RESET}Installing pv:"
    echo ""
    echo -e "${BLUE}Mac:${COLOR_RESET} $ brew install pv"
    echo ""
    echo -e "${BLUE}Other:${COLOR_RESET} http://www.ivarch.com/programs/pv.shtml"
    echo -e "${COLOR_RESET}"
    exit 1;
  }
}

check_pv
#
# handle some default params
# -h for help
# -d for disabling simulated typing
#
while getopts ":dhncw:" opt; do
  case $opt in
    d)
      unset TYPE_SPEED
      ;;
    n)
      NO_WAIT=true
      ;;
    c)
      SHOW_CMD_NUMS=true
      ;;
    w)
      PROMPT_TIMEOUT=$OPTARG
      ;;
    *)
      usage
      exit 1
      ;;
  esac
done

clear

cat << EOM

In this demo we will show how to setup Inter Network Domain connections and Application connections
using Application WAN Interface system deployed in kubernetes cluster.
We will use kubectl to create CRD objects and AWI system controllers will take care of establishing connection.
EOM

read -p ""
read -p "We have AWI system deployed in kubernetes cluster"
pe "kubectl get pods -n awi-system"
pe "kubectl get crd -A"
read -p ""
clear
read -p "We have couple of network domains present"
pe "kubectl get networkdomains -A"
read -p ""
read -p "In our demo we will create InterNetworkDomain object to create connection between
VRF which is site 201 in 'VPN 10' managed by Cisco SD-WAN to VPC 'staging' managed by AWS"
read -p "Let's see the InterNetworkDomain CRD Config that we will use to request this connectivity"
pe "cat config/samples/awi_v1_internetworkdomain_vpn10_staging.yaml"
read -p ""
read -p "Now let's run kubectl to create connection..."
pe "kubectl apply -f config/samples/awi_v1_internetworkdomain_vpn10_staging.yaml"
read -p ""
read -p "After successful connection status of created CRD should be updated"
pe "kubectl get internetworkdomains vpn10-to-staging -o yaml"
read -p ""
clear
read -p "In next step we will demonstrate how to enable app connectivity"
read -p "We will request connectivity between staging VPC database VMs and all VMs from VPN 10"
read -p ""
read -p "We can check what instances there are in staging VPC, by checking instances CRD
and filtering them by VPC ID"
pe "kubectl get instances -A -o=jsonpath=\"{range .items[?(@.spec.VPCID=='vpc-08532095b01548260')]}{.spec.Name}{' '}{.spec.PrivateIP}{'\n'}{end}\""
read -p ""
read -p "To create app connectivity we will create AppConnection CRD object"
read -p ""
read -p "We will need network domain connection ID, we can find it in InterNetworkDomain CRD"
pe "kubectl get internetworkdomains vpn10-to-staging -o yaml"
read -p ""
read -p "Let's see the AppConnection CRD Config that we will use to request app connectivity"
pe "cat config/samples/awi_v1_appconnection_vpn10_staging.yaml"
read -p ""
read -p "Now let's run kubectl to create connection..."
pe "kubectl apply -f config/samples/awi_v1_appconnection_vpn10_staging.yaml"
read -p ""
read -p "When connection will be successfully established status of created CRD should be updated"
pe "kubectl get appconnections vpn10-to-database-staging -o yaml"
read -p ""
clear
read -p "After removing CRD objects app connection and network domain connection will be disabled"
pe "kubectl delete appconnections vpn10-to-database-staging"
pe "kubectl delete internetworkdomains vpn10-to-staging"
read -p ""
clear
read -p "In this demo we showed how to create netwrok domain connection and app connection using
Application WAN Interface system deployed in kubernetes cluster.

Thank you for watching"


