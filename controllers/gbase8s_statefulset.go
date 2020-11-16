package controllers

import (
	gbase8sv1 "Gbase8sCluster/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type gbase8sStatefulset struct {
	sfs *appsv1.StatefulSet
}

func NewGbase8sStatefulset(cluster *gbase8sv1.Gbase8sCluster) *gbase8sStatefulset {
	gsfs := gbase8sStatefulset{}

	trueVar := true
	createStatefulset := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: GBASE8S_STATEFULSET_DEFAULT_NAME,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: cluster.APIVersion,
					Kind:       cluster.Kind,
					Name:       cluster.Name,
					UID:        cluster.UID,
					Controller: &trueVar,
				},
			},
			Namespace: "default",
		},

		Spec: appsv1.StatefulSetSpec{
			ServiceName: GBASE8S_SERVICE_DEFAULT_NAME,
			Replicas:    &cluster.Spec.Gbase8sCfg.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "gbase8s-cluster",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "gbase8s-cluster",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "gbase8s",
							Image: "gbase8s:8.8",
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"SYS_ADMIN",
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9088,
									Name:          "onsoctcp",
								},
							},
						},
					},
				},
			},
		},
	}

	gsfs.sfs = &createStatefulset

	return &gsfs
}
