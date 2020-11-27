package controllers

import (
	gbase8sv1 "Gbase8sCluster/api/v1"
	"fmt"
	corev1 "k8s.io/api/core/v1"
)

type gbase8sPV struct {
	PVs  []*corev1.PersistentVolume
	PVCs []*corev1.PersistentVolumeClaim
}

//func newPV(volInfo *v1.Gbase8sStorage, nodeName, pvName, namespace string) (*corev1.PersistentVolume, error) {
//	quantity, err := resource.ParseQuantity(volInfo.Size)
//	if err != nil {
//		return nil, err
//	}
//	volMode := corev1.PersistentVolumeMode(volInfo.VolumeMode)
//	pv := corev1.PersistentVolume{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      pvName,
//			//Namespace: namespace,
//			Labels: map[string]string{
//				GBASE8S_PV_LABEL_KEY: pvName,
//			},
//		},
//		Spec: corev1.PersistentVolumeSpec{
//			Capacity: corev1.ResourceList{
//				corev1.ResourceStorage: quantity,
//			},
//			VolumeMode: &volMode,
//			AccessModes: []corev1.PersistentVolumeAccessMode{
//				corev1.ReadWriteOnce,
//			},
//			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
//			StorageClassName:              GBASE8S_STORAGE_CLASS_NAME,
//			PersistentVolumeSource: corev1.PersistentVolumeSource{
//				Local: &corev1.LocalVolumeSource{
//					Path: volInfo.Path,
//				},
//			},
//			NodeAffinity: &corev1.VolumeNodeAffinity{
//				Required: &corev1.NodeSelector{
//					NodeSelectorTerms: []corev1.NodeSelectorTerm{
//						{
//							MatchExpressions: []corev1.NodeSelectorRequirement{
//								{
//									Key:      "kubernetes.io/hostname",
//									Operator: corev1.NodeSelectorOpIn,
//									Values: []string{
//										nodeName,
//									},
//								},
//							},
//						},
//					},
//				},
//			},
//		},
//	}
//
//	return &pv, nil
//}
//
//func newPVC(pvcName, pvName, size, namespace string) (*corev1.PersistentVolumeClaim, error) {
//	quantity, err := resource.ParseQuantity(size)
//	if err != nil {
//		return nil, err
//	}
//	storageClassName := GBASE8S_STORAGE_CLASS_NAME
//	pvc := corev1.PersistentVolumeClaim{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      pvcName,
//			Namespace: namespace,
//		},
//		Spec: corev1.PersistentVolumeClaimSpec{
//			AccessModes: []corev1.PersistentVolumeAccessMode{
//				corev1.ReadWriteOnce,
//			},
//			Resources: corev1.ResourceRequirements{
//				Requests: corev1.ResourceList{
//					corev1.ResourceStorage: quantity,
//				},
//			},
//			StorageClassName: &storageClassName,
//			Selector: &metav1.LabelSelector{
//				MatchLabels: map[string]string{
//					GBASE8S_PV_LABEL_KEY: pvName,
//				},
//			},
//		},
//	}
//
//	return &pvc, nil
//}
//
//func setOwnerReference(pv *gbase8sPV, cluster *gbase8sv1.Gbase8sCluster) {
//	trueVar := true
//	if pv.PVs != nil && len(pv.PVs) != 0 {
//		for _, v := range pv.PVs {
//			v.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
//				{
//					APIVersion: cluster.APIVersion,
//					Kind:       cluster.Kind,
//					Name:       cluster.Name,
//					UID:        cluster.UID,
//					Controller: &trueVar,
//				},
//			}
//		}
//	}
//	if pv.PVCs != nil && len(pv.PVCs) != 0 {
//		for _, v := range pv.PVs {
//			v.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
//				{
//					APIVersion: cluster.APIVersion,
//					Kind:       cluster.Kind,
//					Name:       cluster.Name,
//					UID:        cluster.UID,
//					Controller: &trueVar,
//				},
//			}
//		}
//	}
//}

func NewGbase8sPV(cluster *gbase8sv1.Gbase8sCluster) (*gbase8sPV, error) {
	gbase8sPV := gbase8sPV{}
	for i, v := range cluster.Spec.Gbase8sCfg.Nodes {
		//创建storage pv pvc
		if v.Storage != nil {
			pvName := fmt.Sprintf("%s%s-%d", GBASE8S_PV_STORAGE_PREFIX, cluster.Name, i)
			if pv, err := newPV(v.Storage, v.Name, pvName); err != nil {
				return nil, err
			} else {
				gbase8sPV.PVs = append(gbase8sPV.PVs, pv)
			}

			pvcName := fmt.Sprintf("%s-%s%s-%d", GBASE8S_PVC_STORAGE_TEMPLATE_NAME, GBASE8S_STATEFULSET_NAME_PREFIX, cluster.Name, i)
			if pvc, err := newPVC(pvcName, pvName, v.Storage.Size, cluster.Namespace); err != nil {
				return nil, err
			} else {
				gbase8sPV.PVCs = append(gbase8sPV.PVCs, pvc)
			}
		}

		//创建log pv pvc
		if v.Log != nil {
			pvName := fmt.Sprintf("%s%s-%d", GBASE8S_PV_LOG_PREFIX, cluster.Name, i)
			if pv, err := newPV(v.Log, v.Name, pvName); err != nil {
				return nil, err
			} else {
				gbase8sPV.PVs = append(gbase8sPV.PVs, pv)
			}

			pvcName := fmt.Sprintf("%s-%s%s-%d", GBASE8S_PVC_LOG_TEMPLATE_NAME, GBASE8S_STATEFULSET_NAME_PREFIX, cluster.Name, i)
			if pvc, err := newPVC(pvcName, pvName, v.Log.Size, cluster.Namespace); err != nil {
				return nil, err
			} else {
				gbase8sPV.PVCs = append(gbase8sPV.PVCs, pvc)
			}
		}
	}

	setPVOwnerReference(gbase8sPV.PVs, cluster)
	setPVCOwnerReference(gbase8sPV.PVCs, cluster)

	return &gbase8sPV, nil
}
