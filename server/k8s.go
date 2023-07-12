package server

// this piece is the real "orchestration", which deploys a kubernetes cluster
// and a pod - effectively managing a piece of compute

// CREATE A CLUSTER
// will launch spot instances (one controller/api, one pod (for now))
// ignore VPC/networking....

// CONNECT TO A CLUSTER
// look at k8s
//
