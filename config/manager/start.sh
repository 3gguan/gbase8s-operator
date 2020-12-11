#!/bin/bash

kubectl create configmap gbase8s-operator-conf --from-file=config -n gbase8s-operator-conf
