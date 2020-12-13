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
			Name: GBASE8S_STATEFULSET_NAME_PREFIX + cluster.Name,
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
			ServiceName: GBASE8S_SERVICE_NAME_PREFIX + cluster.Name,
			Replicas:    &cluster.Spec.Gbase8sCfg.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					GBASE8S_POD_LABEL_KEY: GBASE8S_POD_LABEL_VALUE_PREFIX + cluster.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						GBASE8S_POD_LABEL_KEY: GBASE8S_POD_LABEL_VALUE_PREFIX + cluster.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  GBASE8S_CONTAINER_NAME,
							Image: cluster.Spec.Gbase8sCfg.Image,
							SecurityContext: &corev1.SecurityContext{
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{
										"SYS_ADMIN",
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: GBASE8S_ONSOCTCP_PORT,
									Name:          "onsoctcp",
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "AUTO_SERVER_NAME",
									Value: "1",
								},
							},
						},
					},
				},
			},
		},
	}

	if cluster.Spec.Gbase8sCfg.Nodes != nil && len(cluster.Spec.Gbase8sCfg.Nodes) != 0 {
		createStatefulset.Spec.Template.Spec.Containers[0].VolumeMounts =
			[]corev1.VolumeMount{
				{
					Name:      GBASE8S_PVC_STORAGE_TEMPLATE_NAME,
					MountPath: GBASE8S_MOUNT_STORAGE_PATH,
				},
				{
					Name:      GBASE8S_PVC_LOG_TEMPLATE_NAME,
					MountPath: GBASE8S_MOUNT_LOG_PATH,
				},
			}

		storageClassName := GBASE8S_STORAGE_CLASS_NAME
		createStatefulset.Spec.VolumeClaimTemplates =
			[]corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: GBASE8S_PVC_STORAGE_TEMPLATE_NAME,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: &storageClassName,
					},
				},
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

	if cluster.Spec.Gbase8sCfg.ConfigMap.Name != "" {
		createStatefulset.Spec.Template.Spec.Volumes =
			[]corev1.Volume{
				{
					Name: GBASE8S_CONF_VOLUME,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cluster.Spec.Gbase8sCfg.ConfigMap.Name,
							},
							Items: []corev1.KeyToPath{
								{
									Key:  cluster.Spec.Gbase8sCfg.ConfigMap.OnconfigKey,
									Path: cluster.Spec.Gbase8sCfg.ConfigMap.OnconfigKey,
								},
								{
									Key:  cluster.Spec.Gbase8sCfg.ConfigMap.AllowedKey,
									Path: cluster.Spec.Gbase8sCfg.ConfigMap.AllowedKey,
								},
							},
						},
					},
				},
			}

		if cluster.Spec.Gbase8sCfg.ConfigMap.OnconfigKey != "" {
			createStatefulset.Spec.Template.Spec.Containers[0].VolumeMounts =
				append(createStatefulset.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
					Name:      GBASE8S_CONF_VOLUME,
					MountPath: GBASE8S_ONCONFIG_MOUNT_PATH,
					SubPath:   cluster.Spec.Gbase8sCfg.ConfigMap.OnconfigKey,
				})

			createStatefulset.Spec.Template.Spec.Containers[0].Env =
				append(createStatefulset.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
					Name:  GBASE8S_ENV_ONCONFIG_FILENAME,
					Value: GBASE8S_ONCONFIG_MOUNT_PATH,
				})
		}

		if cluster.Spec.Gbase8sCfg.ConfigMap.AllowedKey != "" {
			createStatefulset.Spec.Template.Spec.Containers[0].VolumeMounts =
				append(createStatefulset.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
					Name:      GBASE8S_CONF_VOLUME,
					MountPath: GBASE8S_ALLOWED_MOUNT_PATH,
					SubPath:   cluster.Spec.Gbase8sCfg.ConfigMap.AllowedKey,
				})
		}
	}

	if cluster.Spec.Gbase8sCfg.SecretName != "" {
		createStatefulset.Spec.Template.Spec.Containers[0].Env =
			append(createStatefulset.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{
				Name: GBASE8S_GBASEDBT_PASSWORD_NAME,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: cluster.Spec.Gbase8sCfg.SecretName,
						},
						Key: GBASE8S_GBASEDBT_PASSWORD_KEY,
					},
				},
			})
	}

	for _, v := range cluster.Spec.Gbase8sCfg.Env {
		createStatefulset.Spec.Template.Spec.Containers[0].Env = append(createStatefulset.Spec.Template.Spec.Containers[0].Env, v)
	}

	for k, v := range cluster.Spec.Gbase8sCfg.Labels {
		if _, ok := createStatefulset.Spec.Template.Labels[k]; !ok {
			createStatefulset.Spec.Template.Labels[k] = v
		}
	}

	gsfs.sfs = &createStatefulset

	return &gsfs
}
