package controllers

import (
	gbase8sv1 "Gbase8sCluster/api/v1"
	"fmt"
	corev1 "k8s.io/api/core/v1"
)

type CmPV struct {
	PVs  []*corev1.PersistentVolume
	PVCs []*corev1.PersistentVolumeClaim
}

func NewCmPV(cluster *gbase8sv1.Gbase8sCluster) (*CmPV, error) {
	cmPV := CmPV{}
	for i, v := range cluster.Spec.CmCfg.Nodes {
		//创建log pv pvc
		if v.Storage != nil {
			pvName := fmt.Sprintf("%s%s-%d", CM_PV_LOG_PREFIX, cluster.Name, i)
			if pv, err := newPV(v.Storage, v.Name, pvName); err != nil {
				return nil, err
			} else {
				cmPV.PVs = append(cmPV.PVs, pv)
			}

			pvcName := fmt.Sprintf("%s-%s%s-%d", GBASE8S_PVC_LOG_TEMPLATE_NAME, CM_STATEFULSET_NAME_PREFIX, cluster.Name, i)
			if pvc, err := newPVC(pvcName, pvName, v.Storage.Size, cluster.Namespace); err != nil {
				return nil, err
			} else {
				cmPV.PVCs = append(cmPV.PVCs, pvc)
			}
		}
	}

	setPVOwnerReference(cmPV.PVs, cluster)
	setPVCOwnerReference(cmPV.PVCs, cluster)

	return &cmPV, nil
}
