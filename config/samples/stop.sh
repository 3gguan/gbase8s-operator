#!/bin/bash

kubectl delete configmap gbase8s-conf
kubectl delete -f secrets.yaml
kubectl delete -f gbase8s_v1_gbase8scluster.yaml