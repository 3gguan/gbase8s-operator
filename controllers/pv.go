package controllers

import (
	"Gbase8sCluster/api/v1"
	gbase8sv1 "Gbase8sCluster/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newPV(volInfo *v1.Gbase8sStorage, nodeName, pvName string) (*corev1.PersistentVolume, error) {
	quantity, err := resource.ParseQuantity(volInfo.Size)
	if err != nil {
		return nil, err
	}
	volMode := corev1.PersistentVolumeMode(volInfo.VolumeMode)
	pv := corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
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

func newPVC(pvcName, pvName, size, namespace string) (*corev1.PersistentVolumeClaim, error) {
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, err
	}
	storageClassName := GBASE8S_STORAGE_CLASS_NAME
	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
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

func setPVOwnerReference(pvs []*corev1.PersistentVolume, cluster *gbase8sv1.Gbase8sCluster) {
	trueVar := true
	if pvs != nil && len(pvs) != 0 {
		for _, v := range pvs {
			v.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: cluster.APIVersion,
					Kind:       cluster.Kind,
					Name:       cluster.Name,
					UID:        cluster.UID,
					Controller: &trueVar,
				},
			}
		}
	}
}

func setPVCOwnerReference(pvcs []*corev1.PersistentVolumeClaim, cluster *gbase8sv1.Gbase8sCluster) {
	trueVar := true
	if pvcs != nil && len(pvcs) != 0 {
		for _, v := range pvcs {
			v.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: cluster.APIVersion,
					Kind:       cluster.Kind,
					Name:       cluster.Name,
					UID:        cluster.UID,
					Controller: &trueVar,
				},
			}
		}
	}
}
