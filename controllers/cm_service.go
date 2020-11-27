package controllers

import (
	gbase8sv1 "Gbase8sCluster/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CmService struct {
	svc *corev1.Service
}

func NewCmService(cluster *gbase8sv1.Gbase8sCluster) *CmService {
	gsvc := CmService{}
	trueVar := true
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: CM_SERVICE_NAME_PREFIX + cluster.Name,
			Labels: map[string]string{
				CM_SERVICE_LABEL_KEY: CM_SERVICE_LABEL_VALUE_PREFIX + cluster.Name,
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
					Port: CM_SLA_REDIRECT_PORT,
					Name: "redirect",
				},
				{
					Port: CM_SLA_PROXY_PORT,
					Name: "proxy",
				},
			},
			ClusterIP: "None",
			Selector: map[string]string{
				CM_POD_LABEL_KEY: CM_POD_LABEL_VALUE_PREFIX + cluster.Name,
			},
		},
	}

	gsvc.svc = &svc

	return &gsvc
}
