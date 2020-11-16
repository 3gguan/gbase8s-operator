package controllers

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type gbase8sService struct {
	svc *corev1.Service
}

func NewGbase8sService() *gbase8sService {
	gsvc := gbase8sService{}

	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: GBASE8S_SERVICE_DEFAULT_NAME,
			Labels: map[string]string{
				"app": "gbase8s-cluster",
			},
			Namespace: "default",
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
				"app": "gbase8s-cluster",
			},
		},
	}

	gsvc.svc = &svc

	return &gsvc
}
