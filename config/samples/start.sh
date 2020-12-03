#!/bin/bash

kubectl create configmap gbase8s-conf --from-file=conf
kubectl apply -f secrets.yaml
kubectl apply -f gbase8s_v1_gbase8scluster.yaml