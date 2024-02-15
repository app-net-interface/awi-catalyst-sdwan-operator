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
In this demo, we have two VPCs: ML Model training VPC and ML Data Set VPCs.

We need to access data from ML app cluster (source) in Training VPC to data cluster (destination) in Data Set VPC for training ML models.
Training data VPC is dynamically created – therefore the need of on-demand connectivity.

Data needs to be accessed over private link because of compliance requirement. Inter cluster private connectivity is needed.

In this demo we will present how developers can setup application connectivity between ML training pods and
ML dataset services running in different Kuberenetes clusters using Application WAN Interface (AWI) system.
EOM


read -p ""
clear
read -p "In our demo we have 2 GKE kubernetes clusters: 'ml-training-cluster' and 'ml-dataset-cluster' running in two different VPCs."
# show VPCs here
pe "gcloud container clusters list 2>&1 | grep ml-.*cluster"
read -p ""
pe "gcloud container clusters describe ml-training-cluster --zone us-west1-a 2>&1 | grep ^network:"
read -p ""
pe "gcloud compute networks describe ml-training-network | grep id"
read -p ""
pe "gcloud container clusters describe ml-dataset-cluster --zone us-east1-b 2>&1 |  grep  ^network:"
read -p ""
pe "gcloud compute networks describe ml-dataset-network | grep id"
read -p ""
read -p "In ml-training-cluster we have a ML training application running."
# switch context to ml-training

#ml-training-cluster> kubectl get pods -n ml-training
#ml-front-end                              1/1     Running   0          8m5s
#ml-training-deployment-5dc695d4dc-2mdcf   1/1     Running   0          22h
#ml-training-deployment-5dc695d4dc-dwj5j   1/1     Running   0          22h
read -p "In ml-dataset-cluster we have a service which exposes data set, this is internal service with private IP only."
#ml-dataset-cluster> kubectl get service -n ml-dataset
#NAME                 TYPE           CLUSTER-IP      EXTERNAL-IP                                                                       PORT(S)        AGE
#ml-dataset-service   LoadBalancer   172.20.95.103   internal-a07091c66834a4587bc73f7e5ddc8f38-417755801.us-west-2.elb.amazonaws.com   80:32459/TCP   23h
# nslookup internal-a07091c66834a4587bc73f7e5ddc8f38-417755801.us-west-2.elb.amazonaws.com
# Server:         1.1.1.1
# Address:        1.1.1.1#53
#
# Non-authoritative answer:
# Name:   internal-a07091c66834a4587bc73f7e5ddc8f38-417755801.us-west-2.elb.amazonaws.com
# Address: 10.30.124.216
# Name:   internal-a07091c66834a4587bc73f7e5ddc8f38-417755801.us-west-2.elb.amazonaws.com
# Address: 10.30.165.31
read -p ""
clear
read -p "Let's try to connect from pod in source cluster to service in destination cluster."

#ml-training-cluster> kubectl exec -ti -n ml-training ml-training-deployment-5dc695d4dc-2mdcf -- /bin/sh
#/ $ curl internal-a07091c66834a4587bc73f7e5ddc8f38-417755801.us-west-2.elb.amazonaws.com/data
#
#^C
read -p "At the beginning there's no connection between them, because VPCs in which clusters are deployed are not connected.
We can check this in vManage connectivity page."
# Go to vManage and show
read -p "To enable connectivity between clusters, we need to connect the VPCs first.
We have awi-system running in a cluster, which can be used for this purpose."
pe "kubectl get pods -n awi-system"
read -p ""
clear
#xxx>> kubectl get pods -n awi-system
#NAME                                           READY   STATUS    RESTARTS   AGE
#awi-grpc-catalyst-sdwan-7ff85978ff-9cjj6              1/1     Running   0          24h
#kube-awi-controller-manager-6759ff4966-4ck8h   2/2     Running   0          23h
read -p "A cloudOps admin can now use Kubectl to provision connectivity between VPCs.
For that they need to provision an AWI inter-network-domain YAML file (CRD) to establish connectivity between VPCs. Let’s see this file."
pe "cat config/samples/gcloud/internetworkdomain_vpc_mltraining_to_vpc_mldataset.yaml"
# apiVersion: awi.cisco.awi/v1
#kind: InterNetworkDomain
#metadata:
#  name: mltraining-to-mldataset
#spec:
#  name: 'ML Training VPC to ML Dataset VPC'
#  source:
#    metadata:
#      name: 'ML Training VPC'
#    type: 'vpc'
#    network_id: 'vpc-089e44ee0ab01c252'
#    provider: 'aws'
#  destination:
#    metadata:
#      name: 'ML Dataset VPC'
#    type: 'vpc'
#    provider: 'aws'
#    network_id: 'vpc-0acb0f4c655cd5dd9'
#    default_access_control: "deny"
read -p ""
read -p "Now let's use the YAML file to run kubectl to create connection..."
pe "kubectl apply -f config/samples/gcloud/internetworkdomain_vpc_mltraining_to_vpc_mldataset.yaml"
read -p ""
read -p "Establishing connection will take a couple of minutes during which recording will be paused."
# go to vManage and show connection
read -p "After successful connection status of created CRD should be updated."
pe "kubectl get internetworkdomains mltraining-to-mldataset -o yaml"
read -p ""
read -p "Because default_access_control is set to deny traffic, for now we still won't be able to access dataset."
#ml-training-cluster> kubectl exec -ti -n ml-training ml-training-deployment-5dc695d4dc-2mdcf -- /bin/sh
#/ $ curl internal-a07091c66834a4587bc73f7e5ddc8f38-417755801.us-west-2.elb.amazonaws.com/data
#
#^C
read -p ""
clear
read -p "To enable pod to service connectivity we will create AppConnection CRD object, let's see how it looks."
pe "cat config/samples/gcloud/appconnection_mltraining_to_mldataset.yaml"
#$ cat config/samples/gcloud/appconnection_mltraining_to_mldataset.yaml
 #apiVersion: awi.cisco.awi/v1
 #kind: AppConnection
 #metadata:
 #  name: ml-training-app-to-ml-dataset
 #spec:
 #  name: 'Connection from ML training app pods to ML dataset service'
 #  cluster_connection_reference: 'vpc-089e44ee0ab01c252:vpc-0acb0f4c655cd5dd9'
 #  source:
 #    kind:
 #      endpoint:
 #        metadata:
 #          name: "ML Training Cluster App"
 #          labels:
 #            app: "ml-training-app"
 #  destination:
 #    kind:
 #      service:
 #        metadata:
 #          # name of service in format 'cluster_name:namespace:name'
 #          name: "ml-dataset-cluster:ml-dataset:ml-dataset-service"
read -p ""
read -p "The configuration we showed will allow connectivity from only specific labels. Pods with no label or other labels won’t have access to the destination service."
read -p "Now let's apply this configuration."
pe "kubectl apply -f config/samples/gcloud/appconnection_mltraining_to_mldataset.yaml"
read -p ""
read -p "After successful connection status of created CRD should be updated."
pe "kubectl get appconnections ml-training-app-to-ml-dataset-lb-service -o yaml"
  #apiVersion: awi.cisco.awi/v1
  #kind: AppConnection
  #metadata:
  #  annotations:
  #    kubectl.kubernetes.io/last-applied-configuration: |
  #      {"apiVersion":"awi.cisco.awi/v1","kind":"AppConnection","metadata":{"annotations":{},"name":"ml-training-app-to-ml-dataset","namespace":"default"},"spec":{"cluster_connection_reference":"vpc-089e44ee0ab01c252:vpc-0acb0f4c655cd5dd9","destination":{"kind":{"service":{"metadata":{"name":"ml-dataset-cluster:ml-dataset:ml-dataset-service"}}}},"name":"Connection from ML training app pods to ML dataset service","source":{"kind":{"endpoint":{"metadata":{"labels":{"app":"ml-training-app"},"name":"ML Training Cluster App"}}}}}}
  #  creationTimestamp: "2023-02-24T12:47:58Z"
  #  finalizers:
  #  - appconnection.awi.cisco.awi/finalizer
  #  generation: 1
  #  name: ml-training-app-to-ml-dataset
  #  namespace: default
  #  resourceVersion: "4720825"
  #  uid: d41f8b08-609d-40cb-a909-a1f1968cd5f9
  #spec:
  #  cluster_connection_reference: vpc-089e44ee0ab01c252:vpc-0acb0f4c655cd5dd9
  #  destination:
  #    kind:
  #      service:
  #        metadata:
  #          name: ml-dataset-cluster:ml-dataset:ml-dataset-service
  #  name: Connection from ML training app pods to ML dataset service
  #  source:
  #    kind:
  #      endpoint:
  #        metadata:
  #          labels:
  #            app: ml-training-app
  #          name: ML Training Cluster App
  #status: SUCCESS
read -p ""
read -p "Now we should be able to connect from ml-training-app pods to ml-dataset service."

# show the service here where we want to connect to???
# kubectl exec -ti ...
# / $ curl internal-a07091c66834a4587bc73f7e5ddc8f38-417755801.us-west-2.elb.amazonaws.com/data
#name,MDVP:Fo(Hz),MDVP:Fhi(Hz),MDVP:Flo(Hz),MDVP:Jitter(%),MDVP:Jitter(Abs),MDVP:RAP,MDVP:PPQ,Jitter:DDP,MDVP:Shimmer,MDVP:Shimmer(dB),Shimmer:APQ3,Shimmer:APQ5,MDVP:APQ,Shimmer:DDA,NHR,HNR,status,RPDE,DFA,spread1,spread2,D2,PPE
#phon_R01_S01_1,119.99200,157.30200,74.99700,0.00784,0.00007,0.00370,0.00554,0.01109,0.04374,0.42600,0.02182,0.03130,0.02971,0.06545,0.02211,21.03300,1,0.414783,0.815285,-4.813031,0.266482,2.301442,0.284654
#phon_R01_S01_2,122.40000,148.65000,113.81900,0.00968,0.00008,0.00465,0.00696,0.01394,0.06134,0.62600,0.03134,0.04518,0.04368,0.09403,0.01929,19.08500,1,0.458359,0.819521,-4.075192,0.335590,2.486855,0.368674
#phon_R01_S01_3,116.68200,131.11100,111.55500,0.01050,0.00009,0.00544,0.00781,0.01633,0.05233,0.48200,0.02757,0.03858,0.03590,0.08270,0.01309,20.65100,1,0.429895,0.825288,-4.443179,0.311173,2.342259,0.332634
#phon_R01_S01_4,116.67600,137.87100,111.36600,0.00997,0.00009,0.00502,0.00698,0.01505,0.05492,0.51700,0.02924,0.04005,0.03772,0.08771,0.01353,20.64400,1,0.434969,0.819235,-4.117501,0.334147,2.405554,0.368975
#phon_R01_S01_5,116.01400,141.78100,110.65500,0.01284,0.00011,0.00655,0.00908,0.01966,0.06425,0.58400,0.03490,0.04825,0.04465,0.10470,0.01767,19.64900,1,0.417356,0.823484,-3.747787,0.234513,2.332180,0.410335
#phon_R01_S01_6,120.55200,131.16200,113.78700,0.00968,0.00008,0.00463,0.00750,0.01388,0.04701,0.45600,0.02328,0.03526,0.03243,0.06985,0.01222,21.37800,1,0.415564,0.825069,-4.242867,0.299111,2.187560,0.357775
#phon_R01_S02_1,120.26700,137.24400,114.82000,0.00333,0.00003,0.00155,0.00202,0.00466,0.01608,0.14000,0.00779,0.00937,0.01351,0.02337,0.00607,24.88600,1,0.596040,0.764112,-5.634322,0.257682,1.854785,0.211756
#phon_R01_S02_2,107.33200,113.84000,104.31500,0.00290,0.00003,0.00144,0.00182,0.00431,0.01567,0.13400,0.00829,0.00946,0.01256,0.02487,0.00344,26.89200,1,0.637420,0.763262,-6.167603,0.183721,2.064693,0.163755
#phon_R01_S02_3,95.73000,132.06800,91.75400,0.00551,0.00006,0.00293,0.00332,0.00880,0.02093,0.19100,0.01073,0.01277,0.01717,0.03218,0.01070,21.81200,1,0.615551,0.773587,-5.498678,0.327769,2.322511,0.231571
#$ kubectl exec -ti -n ml-training ml-front-end -- /bin/sh
#/ $ curl internal-a07091c66834a4587bc73f7e5ddc8f38-417755801.us-west-2.elb.amazonaws.com/data
#
#^C
read -p ""
read -p "Connection from different pods or from withing nodes should still be rejected."
#$ kubectl exec -ti -n ml-training ml-front-end -- /bin/sh
#/ $ curl internal-a07091c66834a4587bc73f7e5ddc8f38-417755801.us-west-2.elb.amazonaws.com/data
#
#^C
# kubectl get nodes -A
# ssh from nodes
clear
read -p "What happens if new pods get created with the same label?
By design they should have access to the destination service automatically without any additional provisioning from the admin.
Let’s see what happens ..."
# chaning replicas number from 2 to 3
# ml-training-cluster> kubectl edit deploy -n ml-training       ml-training-deployment
#deployment.apps/ml-training-deployment edited

# ml-training-cluster> kubectl get pods -n ml-training -o wide
  #NAME                                      READY   STATUS    RESTARTS   AGE
  #ml-front-end                              1/1     Running   0          8m5s
  #ml-training-deployment-5dc695d4dc-2mdcf   1/1     Running   0          23h
  #ml-training-deployment-5dc695d4dc-9tnqt   1/1     Running   0          23h
  #ml-training-deployment-5dc695d4dc-nsd9l   1/1     Running   0          30s
# ml-training-cluster> kubectl exec -ti -n ml-training       ml-training-deployment-5dc695d4dc-nsd9l -- /bin/sh
  #/ $ curl internal-a07091c66834a4587bc73f7e5ddc8f38-417755801.us-west-2.elb.amazonaws.com/data
  #name,MDVP:Fo(Hz),MDVP:Fhi(Hz),MDVP:Flo(Hz),MDVP:Jitter(%),MDVP:Jitter(Abs),MDVP:RAP,MDVP:PPQ,Jitter:DDP,MDVP:Shimmer,MDVP:Shimmer(dB),Shimmer:APQ3,Shimmer:APQ5,MDVP:APQ,Shimmer:DDA,NHR,HNR,status,RPDE,DFA,spread1,spread2,D2,PPE
  #phon_R01_S01_1,119.99200,157.30200,74.99700,0.00784,0.00007,0.00370,0.00554,0.01109,0.04374,0.42600,0.02182,0.03130,0.02971,0.06545,0.02211,21.03300,1,0.414783,0.815285,-4.813031,0.266482,2.301442,0.284654
  #phon_R01_S01_2,122.40000,148.65000,113.81900,0.00968,0.00008,0.00465,0.00696,0.01394,0.06134,0.62600,0.03134,0.04518,0.04368,0.09403,0.01929,19.08500,1,0.458359,0.819521,-4.075192,0.335590,2.486855,0.368674
clear
read -p "To disable connection we just need to remove CRD objects."
pe "kubectl delete -f config/samples/gcloud/appconnection_mltraining_to_mldataset.yaml"
# $ kubectl delete -f config/samples/gcloud/appconnection_mltraining_to_mldataset.yaml
  #appconnection.awi.cisco.awi "ml-training-app-to-ml-dataset" deleted

# ml-training-cluster> kubectl exec -ti -n ml-training       ml-training-deployment-5dc695d4dc-nsd9l -- /bin/sh
  #/ $ curl internal-a07091c66834a4587bc73f7e5ddc8f38-417755801.us-west-2.elb.amazonaws.com/data
  #
  #^C
read -p ""
clear
read -p "In this demo we showed how to dynamically create connection from pods in one cluster to service in the other cluster
using Application WAN Interface system.

Thank you for watching"


