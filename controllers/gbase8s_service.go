package controllers

import (
	gbase8sv1 "Gbase8sCluster/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type gbase8sService struct {
	svc *corev1.Service
}

func NewGbase8sService(cluster *gbase8sv1.Gbase8sCluster) *gbase8sService {
	gsvc := gbase8sService{}
	trueVar := true
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: GBASE8S_SERVICE_DEFAULT_NAME,
			Labels: map[string]string{
				GBASE8S_SERVICE_LABEL_KEY: GBASE8S_SERVICE_LABEL_VALUE,
			},
			Namespace: cluster.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: cluster.APIVersion,
					Kind:       cluster.Kind,
					Name:       cluster.Name,
					UID:        cluster.UID,
					Controller: &trueVar,
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 9088,
					Name: "onsoctcp",
				},
			},
			ClusterIP: "None",
			Selector: map[string]string{
				GBASE8S_STATEFULSET_LABEL_KEY: GBASE8S_STATEFULSET_LABEL_VALUE,
			},
		},
	}

	gsvc.svc = &svc

	return &gsvc
}
