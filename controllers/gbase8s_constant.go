package controllers

const (
	GBASE8S_STATUS_NONE          = "None"
	GBASE8S_STATUS_INIT          = "Initialization"
	GBASE8S_STATUS_FAST_RECOVERY = "Fast Recovery"
	GBASE8S_STATUS_ONLINE        = "On-Line"
	GBASE8S_STATUS_RSS           = "Read-Only (RSS)"

	GBASE8S_ROLE_STANDARD = "Standard"
	GBASE8S_ROLE_PRIMARY  = "Primary"
	GBASE8S_ROLE_RSS      = "RSS"

	//gbase8s cm公用
	GBASE8S_PV_LABEL_KEY              = "gbase8sPVName"
	GBASE8S_STORAGE_CLASS_NAME        = "gbase8s-cluster-local-volume"
	GBASE8S_PVC_STORAGE_TEMPLATE_NAME = "gbase8s-storage"
	GBASE8S_PVC_LOG_TEMPLATE_NAME     = "gbase8s-log"

	//gbase8s相关
	GBASE8S_STATEFULSET_NAME_PREFIX = "gc-"

	GBASE8S_SERVICE_NAME_PREFIX        = "gbase8s-svc-"
	GBASE8S_SERVICE_LABEL_KEY          = "gbase8ssvc"
	GBASE8S_SERVICE_LABEL_VALUE_PREFIX = "gbase8ssvc-"

	GBASE8S_POD_LABEL_KEY          = "gbase8s"
	GBASE8S_POD_LABEL_VALUE_PREFIX = "gbase8s-"

	GBASE8S_CONTAINER_NAME = "gbase8s"

	GBASE8S_PV_STORAGE_PREFIX = "gbase8s-storage-"
	GBASE8S_PV_LOG_PREFIX     = "gbase8s-log-"

	GBASE8S_MOUNT_STORAGE_PATH = "/opt/gbase8s/storage"
	GBASE8S_MOUNT_LOG_PATH     = "/opt/gbase8s/logs"

	GBASE8S_ONSOCTCP_PORT = 9088
	GBASE8S_DRSOCTCP_PORT = 19088

	GBASE8S_CONF_VOLUME            = "conf"
	GBASE8S_ONCONFIG_MOUNT_PATH    = "/conf/onconfig.ol_gbasedbt_1"
	GBASE8S_ALLOWED_MOUNT_PATH     = "/conf/allowed.surrogates"
	GBASE8S_GBASEDBT_PASSWORD_NAME = "GBASEDBT_PASSWORD"
	GBASE8S_GBASEDBT_PASSWORD_KEY  = "gbase8s-password"
	GBASE8S_ENV_ONCONFIG_FILENAME  = "ONCONFIG_FILE_NAME"

	//cm相关
	CM_STATEFULSET_NAME_PREFIX = "cm-"

	CM_SERVICE_NAME_PREFIX        = "cm-svc-"
	CM_SERVICE_LABEL_KEY          = "cmsvc"
	CM_SERVICE_LABEL_VALUE_PREFIX = "cmsvc-"

	CM_POD_LABEL_KEY          = "cm"
	CM_POD_LABEL_VALUE_PREFIX = "cm-"

	CM_CONTAINER_NAME = "cm"

	CM_SLA_REDIRECT_PORT = 10000
	CM_SLA_PROXY_PORT    = 10001

	CM_MOUNT_LOG_PATH = "/opt/gbase8s/logs"
	//CM_STORAGE_CLASS_NAME = "cm-local-volume"
	CM_PV_LOG_PREFIX               = "cm-log-"
	CM_REDIRECT_GROUP_DEFAULT_NAME = "cm_redirect"
	CM_PROXY_GROUP_DEFAULT_NAME    = "cm_proxy"

	CM_CONF_VOLUME_NAME       = "conf"
	CM_CONF_MOUNT_PATH        = "/conf/cfg.cm"
	CM_GBASEDBT_PASSWORD_NAME = "GBASEDBT_PASSWORD"
	CM_GBASEDBT_PASSWORD_KEY  = "cm-password"
	CM_ENV_CONF_FILENAME      = "CONF_FILE_NAME"
)
