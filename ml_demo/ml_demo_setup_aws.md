### Machine learning demo setup

Setup for demo in k8s-ml-demo.sh script.

- Create source training cluster:
```
eksctl create cluster --name ml-training-cluster --vpc-cidr 10.101.0.0/16 --ssh-access=true --ssh-public-key=~/.ssh/dev_rsa.pub
```

- Apply k8s setup in this cluster:
```
kubectl apply -f k8s-ml-demo-src.yaml
```
- Disable SNAT for source pods:
```
kubectl set env daemonset -n kube-system aws-node AWS_VPC_K8S_CNI_EXCLUDE_SNAT_CIDRS=10.0.0.0/8
```
- Create destination dataset cluster:
```
eksctl create cluster --name ml-dataset-cluster --region us-west-2 --version 1.24 --vpc-cidr 10.201.0.0/16 --ssh-access=true --ssh-public-key=~/.ssh/dev_rsa.pub
```
- Apply k8s setup in this cluster:
```
kubectl apply -f k8s-ml-demo-dest-aws.yaml
```

- In load balancer created for destination service `ml-dataset-service` remove rules allowing access to port 80 in AWS console.

- Add rules towards Transit Gateway for `10.0.0.0/8` in `PublicRouteTable` in source and destination clusters
(if TGW is not available as an option in routing table create transit gateway attachment towards this VPC first).  

- Add some ml data in destination pod:
```
kubectl exec -ti -n ml-dataset    deploy/ml-data-deployment -- /bin/sh
# cd /usr/share/nginx/html
# echo "name,MDVP:Fo(Hz),MDVP:Fhi(Hz),MDVP:Flo(Hz),MDVP:Jitter(%),MDVP:Jitter(Abs),MDVP:RAP,MDVP:PPQ,Jitter:DDP,MDVP:Shimmer,MDVP:Shimmer(dB),Shimmer:APQ3,Shimmer:APQ5,MDVP:APQ,Shimmer:DDA,NHR,HNR,status,RPDE,DFA,spread1,spread2,D2,PPE
phon_R01_> S01_1,119.99200,157.30200,74.99700,0.00784,0.00007,0.00370,0.00554,0.01109,0.04374,0.42600,0.02182,0.03130,0.02971,0.06545,0.02211,21.03300,1,0.414783,0.815285,-4.813031,0.266482,2.301442,0.284654
> phon_R01_S01_2,122.40000,148.65000,113.81900,0.00968,0.00008,0.00465,0.00696,0.01394,0.06134,0.62600,0.03134,0.04518,0.04368,0.09403,0.01929,19.08500,1,0.458359,0.819521,-4.075192,0.335590,2.486855,0.368674
phon_R01_S01_3,> 116.68200,131.11100,111.55500,0.01050,0.00009,0.00544,0.00781,0.01633,0.05233,0.48200,0.02757,0.03858,0.03590,0.08270,0.01309,20.65100,1,0.429895,0.825288,-4.443179,0.311173,2.342259,0.332634
phon_R01_S01_4,116.67600,137.87100,111.36600,0.00997,0.00009,0.00502,0.00698,0.01505,0.05492,0.51700,0.02924,0.04005,0.03772,0.08771,0.01353,20.64400,1,0.434969,0.819235,-4.117501,0.334147,2.405554,0.368975
phon_R01_S01_5,116.01400,141.78100,110.65500,0.01284,0.00011,0.00655,0.00908,0.01966,0.06425,0.58400,0.03490,0.04825,0.04465,0.10470,0.01767,19.64900,1,0.417356,0.823484,-3.747787,0.234513,2.332180,0.410335
phon_R01_S01_6,120.55200,131.16200,113.78700,0.00968,0.00008,0.00463,0.00750,0.0138> > > 8,0.04701,0.45600,0.02328,0.03526,0.03243,0.06985,0.01222,21.37800,1,0.415564,0.825069,-4.242867,0.299111,2.187560,0.357775
phon_R01_S02_1,120.26700,137.24400,114.82000,0.00333,0.00003,0.00155,0.00202,0.00466,0.01608,0.14000,0.00779,0.00937,0.01351,0.02337,0.00607,24.88600,1,0.596040,0.764112,-5.634322,0.257682,1.854785,0.211756
phon_R01_S02_2,107.33200,113.84000,104.31500,0.00290,0.00003,0.00144,0.00182,0.00431,0.01567,0.13400,0.00829,0.00946,0.01256,0.02487,0.00344,26.89200,1,0.637420,0.763262,-6.167603,0.183721,2.064693,0.163755
phon_R01_S02_3,95.73000,132.06800,91.75400,0.00551,0.00006,0.00293,0.00332,0.00880,0.02093,0.19100,0.01073,0.01277,0.01717,0.03218,0.01070,21.81200,1,0.615551,0.773587,-5.498678,0.327769,2.322511,0.231571
phon_R01_S02_4,95.05600,120.10300,91.22600,0.00532,0.00006,0.00268,0.00332,0.00803,0.02838,0.> > > > 25500,0.01441,0.01725,0.02444,0.04324,0.01022,21.86200,1,0.547037,0.798463,-5.011879,0.325996,2.432792,0.271362
phon_R01_S02_5,88.33300,112.24000,84.07200,0.00505,0.00006,0.00254,0.00330,0.00763,0.02143,0.19700,0.01079,0.01342,0.01892,0.03237,0.01166,21.11800,1,0.611137,0.776156,-5.249770,0.391002,2.407313,0.249740
phon_R01_S02_6,91.90400,115.87100,86.29200,0.00540,0.00006,0.00281,0.00336,0.00844,0.02752,0.24900,0.01424,0.01641,0.02214,0.04272,0.01141,21.41400,1,0.583390,0.792520,-4.960234,0.363566,2.642476,0.275931
phon> > > _R01_S04_1,136.92600,159.86600,131.27600,0.00293,0.00002,0.00118,0.00153,0.00355,0.01259,0.11200,0.00656,0.00717,0.01140,0.01968,0.00581,25.70300,1,0.460600,0.646846,-6.547148,0.152813,2.041277,0.138512
phon_R01_S04_2,139.17300,179.13900,76.55600,0.00390,0.00003,0.00165,0.00208,0.00496,0.01642,0.15400,0.00728,0.00932,0.01797,0.02184,0.01041,24.88900,1,0.430166,0.665833,-5.660217,0.254989,2.519422,0.199889
phon_R01_S04_3,152.84500,163.30500,75.83600,0.00294,0.00002,0.00121,0.00149,0.00> > 364,0.01828,0.15800,0.01064,0.00972,0.01246,0.03191,0.00609,24.92200,1,0.474791,0.654027,-6.105098,0.203653,2.125618,0.170100
phon_R01_S04_4,142.16> 700,217.45500,83.15900,0.00369,0.00003,0.00157,0.00203,0.00471,0.01503,0.12600,0.00772,0.00888,0.01359,0.02316,0.00839,25.17500,1,0.565924,0.658245,-5.340115,0.210185,2.205546,0.234589
phon_R01_S04_5,144.18800,349.25900,82.76400,0.00544,0.00004,0.00211,0.00292,0.00632,0.02047,0.19200,0.00969,0.01200,0.02074,0.02908,0.01859,22.33300,1,0.567380,0.644692,-5.440040,0.239764,2.264501,0.218> 164
phon_R01_S04_6,168.77800,232.18100,75.60300,0.00718,0.00004,0.00284,0.00387,0.00853,0.03327,0.34800,0.01441,0.01893,0.0> 3430,0.04322,0.02919,20.37600,1,0.631099,0.605417,-2.931070,0.434326,3.007463,0.430788
p> hon_R01_S05_1,153.04600,175.82900,68.62300,0.00742,0.00005,0.00364,0.00432,0.01092,0.05517,0.54200,0.02471,0.03572,0.05767,0.07413,0.03160,17.28000,1,0.665318,0.719467,-3.949079,0.357870,3.109010,0.377429
phon_> R01_S05_2,156.40500,189.39800,142.82200,0.00768,0.00005,0.00372,0.00399,0.01116,0.03995,0.34800,0.01721,0.02374,0.04310,0.05164,0.03365,17.15300,1,0.649554,0.686080,-4.554466,0.340176,2.856676,0.322111" > data
```

- create kubeconfig file for source and destination clusters using steps based on https://devopscube.com/kubernetes-kubeconfig-file/,
for dataset cluster (for training replace `awi-admin-dataset` with `awi-admin-training` for all steps):
  
```
kubectl -n kube-system create serviceaccount awi-admin-dataset-secret
```

```
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: awi-admin-dataset-secret
  namespace: kube-system
  annotations:
    kubernetes.io/service-account.name: awi-admin-dataset
type: kubernetes.io/service-account-token
EOF
```

--

```
cat << EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: awi-admin-dataset
rules:
- apiGroups: [""]
  resources:
  - nodes
  - nodes/proxy
  - services
  - endpoints
  - pods
  verbs: ["get", "list", "watch"]
- apiGroups:
  - extensions
  resources:
  - ingresses
  verbs: ["get", "list", "watch"]
EOF
```

```
cat << EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: awi-admin-dataset
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: awi-admin-dataset
subjects:
- kind: ServiceAccount
  name: awi-admin-dataset
  namespace: kube-system
EOF
```


```
export SA_SECRET_TOKEN=$(kubectl -n kube-system get secret/awi-admin-dataset-secret -o=go-template='{{.data.token}}' | base64 --decode)

export CLUSTER_NAME=$(kubectl config current-context)

export CURRENT_CLUSTER=$(kubectl config view --raw -o=go-template='{{range .contexts}}{{if eq .name "'''${CLUSTER_NAME}'''"}}{{ index .context "cluster" }}{{end}}{{end}}')

export CLUSTER_CA_CERT=$(kubectl config view --raw -o=go-template='{{range .clusters}}{{if eq .name "'''${CURRENT_CLUSTER}'''"}}"{{with index .cluster "certificate-authority-data" }}{{.}}{{end}}"{{ end }}{{ end }}')

export CLUSTER_ENDPOINT=$(kubectl config view --raw -o=go-template='{{range .clusters}}{{if eq .name "'''${CURRENT_CLUSTER}'''"}}{{ .cluster.server }}{{end}}{{ end }}')
```


```
cat << EOF > awi-admin-dataset-config
apiVersion: v1
kind: Config
current-context: ${CLUSTER_NAME}
contexts:
- name: ${CLUSTER_NAME}
  context:
    cluster: ${CLUSTER_NAME}
    user: awi-admin-dataset
clusters:
- name: ${CLUSTER_NAME}
  cluster:
    certificate-authority-data: ${CLUSTER_CA_CERT}
    server: ${CLUSTER_ENDPOINT}
users:
- name: awi-admin-dataset
  user:
    token: ${SA_SECRET_TOKEN}
EOF
```

- merge configs
```
KUBECONFIG=awi-admin-dataset-config:awi-admin-training-config kubectl config view --merge --flatten > kubeconfig
```

- move this config to `awi-grpc-catalyst-sdwan` repo
- deploy `awi-grpc-catalyst-sdwan` in source cluster
```
cd awi-grpc-catalyst-sdwan
VMANAGE_USERNAME=admin VMANAGE_PASSWORD=<pass> AWS_ACCESS_KEY_ID=<id> AWS_SECRET_ACCESS_KEY=<key> make deploy-kubernetes-config
```

- deploy kube-awi operator in source cluster
```
cd kube-awi
make deploy
```

