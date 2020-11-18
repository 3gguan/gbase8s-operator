package controllers

const (
	GBASE8S_SERVICE_DEFAULT_NAME = "gbase8s-cluster"

	GBASE8S_STATEFULSET_DEFAULT_NAME = "gbase8s-cluster"

	GBASE8S_STORAGE_CLASS_NAME = "gbase8s-local-volume"

	GBASE8S_PV_LABEL_KEY = "gbase8sPVName"

	GBASE8S_PVC_STORAGE_TEMPLATE_NAME = "gbase8s-local-storage-volume"
	GBASE8S_PVC_LOG_TEMPLATE_NAME     = "gbase8s-local-log-volume"

	GBASE8S_CLUSTER_CONTAINER_NAME = "gbase8s"

	GBASE8S_MOUNT_STORAGE_PATH = "/opt/gbase8s/storage"
	GBASE8S_MOUNT_LOG_PATH     = "/opt/gbase8s/logs"

	GBASE8S_STATEFULSET_LABEL_KEY   = "gbase8sApp"
	GBASE8S_STATEFULSET_LABEL_VALUE = "gbase8s-cluster"

	GBASE8S_SERVICE_LABEL_KEY   = "gbase8sServiceName"
	GBASE8S_SERVICE_LABEL_VALUE = "gbase8s-cluster-service"
)
