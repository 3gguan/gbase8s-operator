package controllers

import (
	"Gbase8sCluster/api/v1"
	gbase8sv1 "Gbase8sCluster/api/v1"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type gbase8sPV struct {
	PVs  []*corev1.PersistentVolume
	PVCs []*corev1.PersistentVolumeClaim
}

func newPV(volInfo *v1.Gbase8sStorage, nodeName, pvName string) (*corev1.PersistentVolume, error) {
	quantity, err := resource.ParseQuantity(volInfo.Size)
	if err != nil {
		return nil, err
	}
	volMode := corev1.PersistentVolumeMode(volInfo.VolumeMode)
	pv := corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvName,
			Namespace: "default",
			Labels: map[string]string{
				GBASE8S_PV_LABEL_KEY: pvName,
			},
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: corev1.ResourceList{
				corev1.ResourceStorage: quantity,
			},
			VolumeMode: &volMode,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			StorageClassName:              GBASE8S_STORAGE_CLASS_NAME,
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				Local: &corev1.LocalVolumeSource{
					Path: volInfo.Path,
				},
			},
			NodeAffinity: &corev1.VolumeNodeAffinity{
				Required: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "kubernetes.io/hostname",
									Operator: corev1.NodeSelectorOpIn,
									Values: []string{
										nodeName,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return &pv, nil
}

func newPVC(pvcName, pvName, size string) (*corev1.PersistentVolumeClaim, error) {
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, err
	}
	storageClassName := GBASE8S_STORAGE_CLASS_NAME
	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: "default",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: quantity,
				},
			},
			StorageClassName: &storageClassName,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					GBASE8S_PV_LABEL_KEY: pvName,
				},
			},
		},
	}

	return &pvc, nil
}

func NewGbase8sPV(cluster *gbase8sv1.Gbase8sCluster) (*gbase8sPV, error) {
	gbase8sPV := gbase8sPV{}
	for i, v := range cluster.Spec.Gbase8sCfg.Nodes {
		//创建storage pv pvc
		if v.Storage != nil {
			pvName := fmt.Sprintf("%s-pv-storage-%d", GBASE8S_SERVICE_DEFAULT_NAME, i)
			if pv, err := newPV(v.Storage, v.Name, pvName); err != nil {
				return nil, err
			} else {
				gbase8sPV.PVs = append(gbase8sPV.PVs, pv)
			}

			pvcName := fmt.Sprintf("%s-%s-%d", GBASE8S_PVC_STORAGE_TEMPLATE_NAME, GBASE8S_STATEFULSET_DEFAULT_NAME, i)
			if pvc, err := newPVC(pvcName, pvName, v.Storage.Size); err != nil {
				return nil, err
			} else {
				gbase8sPV.PVCs = append(gbase8sPV.PVCs, pvc)
			}
		}

		//创建log pv pvc
		if v.Log != nil {
			pvName := fmt.Sprintf("%s-pv-log-%d", GBASE8S_SERVICE_DEFAULT_NAME, i)
			if pv, err := newPV(v.Log, v.Name, pvName); err != nil {
				return nil, err
			} else {
				gbase8sPV.PVs = append(gbase8sPV.PVs, pv)
			}

			pvcName := fmt.Sprintf("%s-%s-%d", GBASE8S_PVC_LOG_TEMPLATE_NAME, GBASE8S_STATEFULSET_DEFAULT_NAME, i)
			if pvc, err := newPVC(pvcName, pvName, v.Storage.Size); err != nil {
				return nil, err
			} else {
				gbase8sPV.PVCs = append(gbase8sPV.PVCs, pvc)
			}
		}
	}

	return &gbase8sPV, nil
}
