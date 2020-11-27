package controllers

import (
	gbase8sv1 "Gbase8sCluster/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CmStatefulset struct {
	sfs *appsv1.StatefulSet
}

func NewCmStatefulset(cluster *gbase8sv1.Gbase8sCluster) *gbase8sStatefulset {
	gsfs := gbase8sStatefulset{}

	trueVar := true
	createStatefulset := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: CM_STATEFULSET_NAME_PREFIX + cluster.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: cluster.APIVersion,
					Kind:       cluster.Kind,
					Name:       cluster.Name,
					UID:        cluster.UID,
					Controller: &trueVar,
				},
			},
			Namespace: cluster.Namespace,
		},

		Spec: appsv1.StatefulSetSpec{
			ServiceName: CM_SERVICE_NAME_PREFIX + cluster.Name,
			Replicas:    &cluster.Spec.CmCfg.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					CM_POD_LABEL_KEY: CM_POD_LABEL_VALUE_PREFIX + cluster.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						CM_POD_LABEL_KEY: CM_POD_LABEL_VALUE_PREFIX + cluster.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  CM_CONTAINER_NAME,
							Image: cluster.Spec.CmCfg.Image,
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"SYS_ADMIN",
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: CM_SLA_REDIRECT_PORT,
									Name:          "redirect",
								},
								{
									ContainerPort: CM_SLA_PROXY_PORT,
									Name:          "proxy",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "START_MANUAL",
									Value: "1",
								},
							},
						},
					},
				},
			},
		},
	}

	if cluster.Spec.CmCfg.Nodes != nil && len(cluster.Spec.CmCfg.Nodes) != 0 {
		createStatefulset.Spec.Template.Spec.Containers[0].VolumeMounts =
			[]corev1.VolumeMount{
				{
					Name:      GBASE8S_PVC_LOG_TEMPLATE_NAME,
					MountPath: CM_MOUNT_LOG_PATH,
				},
			}

		storageClassName := CM_STORAGE_CLASS_NAME
		createStatefulset.Spec.VolumeClaimTemplates =
			[]corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: GBASE8S_PVC_LOG_TEMPLATE_NAME,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: &storageClassName,
					},
				},
			}
	}

	gsfs.sfs = &createStatefulset

	return &gsfs
}
