package controllers

import (
	gbase8sv1 "Gbase8sCluster/api/v1"
	"Gbase8sCluster/util"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type Gbase8sClusterBuilder struct {
	client.Client
	ExecInPod *util.ExecInPod
}

func NewGbase8sClusterBuilder(c client.Client, execClient *util.ExecInPod) *Gbase8sClusterBuilder {
	return &Gbase8sClusterBuilder{
		Client:    c,
		ExecInPod: execClient,
	}
}

//func (r *Gbase8sClusterBuilder) GetHostTemplate(pods *corev1.PodList) *[]string {
//	getHostCmd := []string{"bash", "-c", "hostname && dnsdomainname"}
//
//	//获取hostname和dnsdomainname
//	hostnameStr := ""
//	for _, v := range pods.Items {
//		if len(v.Status.ContainerStatuses) != 0 {
//			if v.Status.ContainerStatuses[0].State.Running != nil {
//				stdout, stderr, err := r.ExecInPod.Exec(getHostCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
//				if err != nil {
//					log.Errorf("get hostname failed, error: %s %s", err.Error(), stderr)
//				} else {
//					if stdout != "" {
//						hostnameStr = stdout
//						break
//					}
//				}
//			}
//		}
//	}
//
//	if hostnameStr == "" {
//		return nil
//	}
//
//	hostname := strings.Split(hostnameStr, "\n")
//	//var hostnameTemplate string
//	for i := len(hostname[0]); i > 0; i-- {
//		if hostname[0][i-1] == '-' {
//			hostname[0] = hostname[0][0 : i-1]
//			break
//		}
//	}
//
//	return &hostname
//}

func (r *Gbase8sClusterBuilder) GenerateTrustString(podNum int, host, domain string) string {
	//准备互信字符串
	var hostfileStr strings.Builder
	for i := 0; i < podNum; i++ {
		tmpStr := fmt.Sprintf("%s-%d", host, i)
		hostfileStr.WriteString(tmpStr)
		hostfileStr.WriteString(" gbasedbt\n")
		hostfileStr.WriteString(tmpStr + "." + domain)
		hostfileStr.WriteString(" gbasedbt\n")
	}

	return hostfileStr.String()
}

func (r *Gbase8sClusterBuilder) BuildTrust(pods *corev1.PodList, trustStr string) {
	//向容器内写入互信字符串
	setHostfileCmd := []string{"bash", "-c", "echo -e " + "'" + trustStr + "'" + " > /opt/gbase8s/etc/hostfile"}
	for _, v := range pods.Items {
		//log.Infof("pod name: %s", v.Name)
		if len(v.Status.ContainerStatuses) != 0 {
			if v.Status.ContainerStatuses[0].State.Running != nil {
				_, _, err := r.ExecInPod.Exec(setHostfileCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
				if err != nil {
					log.Errorf("set hostfile failed, error: %s", err.Error())
				}
			}
		}
	}
}

func (r *Gbase8sClusterBuilder) GenerateGbase8sSqlhostString(podNum int, host, domain string) string {
	var sqlhostStr strings.Builder
	serverNameTemplate := strings.Replace(host, "-", "_", -1)
	for i := 0; i < podNum; i++ {
		serverName := fmt.Sprintf("%s_%d", serverNameTemplate, i)
		hostName := fmt.Sprintf("%s-%d.%s", host, i, domain)
		sqlhostStr.WriteString(serverName)
		sqlhostStr.WriteString(" onsoctcp ")
		sqlhostStr.WriteString(hostName)
		sqlhostStr.WriteString(fmt.Sprintf(" %d\n", GBASE8S_ONSOCTCP_PORT))

		sqlhostStr.WriteString("dr_" + serverName)
		sqlhostStr.WriteString(" drsoctcp ")
		sqlhostStr.WriteString(hostName)
		sqlhostStr.WriteString(fmt.Sprintf(" %d\n", GBASE8S_DRSOCTCP_PORT))
	}

	return sqlhostStr.String()
}

func (r *Gbase8sClusterBuilder) BuildGbase8sSqlhost(pods *corev1.PodList, host, domain string) {
	//准备sqlhost字符串
	str := r.GenerateGbase8sSqlhostString(len(pods.Items), host, domain)

	//向容器内写入sqlhost字符串
	setSqlhostCmd := []string{"bash", "-c", "echo -e " + "'" + str + "'" + " > /opt/gbase8s/etc/sqlhosts.ol_gbasedbt_1"}
	for _, v := range pods.Items {
		//log.Infof("pod name: %s", v.Name)
		if len(v.Status.ContainerStatuses) != 0 {
			if v.Status.ContainerStatuses[0].State.Running != nil {
				_, _, err := r.ExecInPod.Exec(setSqlhostCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
				if err != nil {
					log.Errorf("set sqlhost failed, error: %s", err.Error())
				}
			}
		}
	}
}

func (r *Gbase8sClusterBuilder) GenerateCmSqlhostString(
	gbase8sPodNum, cmPodNum int,
	gbase8sHost, gbase8sDomain string,
	cmHost, cmDomain string,
	redirectGroupName, proxyGroupName string) string {

	var rName, pName string
	if len(redirectGroupName) != 0 {
		rName = redirectGroupName
	} else {
		rName = CM_REDIRECT_GROUP_DEFAULT_NAME
	}

	if len(proxyGroupName) != 0 {
		pName = proxyGroupName
	} else {
		pName = CM_PROXY_GROUP_DEFAULT_NAME
	}

	//gbase8s sqlhost
	var sqlhostStr strings.Builder
	sqlhostStr.WriteString("g_cluster group - - i=10\n")
	serverNameTemplate := strings.Replace(gbase8sHost, "-", "_", -1)
	for i := 0; i < gbase8sPodNum; i++ {
		serverName := fmt.Sprintf("%s_%d", serverNameTemplate, i)
		hostName := fmt.Sprintf("%s-%d.%s", gbase8sHost, i, gbase8sDomain)
		sqlhostStr.WriteString(serverName)
		sqlhostStr.WriteString(" onsoctcp ")
		sqlhostStr.WriteString(hostName)
		sqlhostStr.WriteString(fmt.Sprintf(" %d g=g_cluster\n", GBASE8S_ONSOCTCP_PORT))
	}

	sqlhostStr.WriteString(rName)
	sqlhostStr.WriteString(" group - - c=1\n")
	cmRNameTemplate := "redirect_" + strings.Replace(cmHost, "-", "_", -1)
	for i := 0; i < cmPodNum; i++ {
		serverName := fmt.Sprintf("%s_%d", cmRNameTemplate, i)
		hostName := fmt.Sprintf("%s-%d.%s", cmHost, i, cmDomain)
		sqlhostStr.WriteString(serverName)
		sqlhostStr.WriteString(" onsoctcp ")
		sqlhostStr.WriteString(hostName)
		sqlhostStr.WriteString(fmt.Sprintf(" %d g=%s\n", CM_SLA_REDIRECT_PORT, rName))
	}

	sqlhostStr.WriteString(pName)
	sqlhostStr.WriteString(" group - - c=1\n")
	cmPNameTemplate := "proxy_" + strings.Replace(cmHost, "-", "_", -1)
	for i := 0; i < cmPodNum; i++ {
		serverName := fmt.Sprintf("%s_%d", cmPNameTemplate, i)
		hostName := fmt.Sprintf("%s-%d.%s", cmHost, i, cmDomain)
		sqlhostStr.WriteString(serverName)
		sqlhostStr.WriteString(" onsoctcp ")
		sqlhostStr.WriteString(hostName)
		sqlhostStr.WriteString(fmt.Sprintf(" %d g=%s\n", CM_SLA_PROXY_PORT, pName))
	}

	return sqlhostStr.String()
}

func (r *Gbase8sClusterBuilder) BuildCmSqlhost(pods *corev1.PodList, str string) {
	//向容器内写入sqlhost字符串
	setSqlhostCmd := []string{"bash", "-c", "echo -e " + "'" + str + "'" + " > /opt/gbase8s/etc/sqlhosts.cm"}
	for _, v := range pods.Items {
		//log.Infof("pod name: %s", v.Name)
		if len(v.Status.ContainerStatuses) != 0 {
			if v.Status.ContainerStatuses[0].State.Running != nil {
				_, _, err := r.ExecInPod.Exec(setSqlhostCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
				if err != nil {
					log.Errorf("set sqlhost failed, error: %s", err.Error())
				}
			}
		}
	}
}

func (r *Gbase8sClusterBuilder) BuildCluster(cluster *gbase8sv1.Gbase8sCluster) error {
	expectGbase8sPodNum := cluster.Spec.Gbase8sCfg.Replicas
	gbase8sPodLabels := map[string]string{
		GBASE8S_POD_LABEL_KEY: GBASE8S_POD_LABEL_VALUE_PREFIX + cluster.Name,
	}
	gbase8sPods, err := GetAllPods(cluster.Namespace, &gbase8sPodLabels)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}

	//log.Infof("get gbase8s pods success, pods num: %d", len(gbase8sPods.Items))
	if expectGbase8sPodNum != int32(len(gbase8sPods.Items)) {
		//log.Info("gbase8s not build")
		return nil
	}

	expectCmPodNum := cluster.Spec.CmCfg.Replicas
	cmPodLabels := map[string]string{
		CM_POD_LABEL_KEY: CM_POD_LABEL_VALUE_PREFIX + cluster.Name,
	}
	cmPods, err := GetAllPods(cluster.Namespace, &cmPodLabels)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return nil
		} else {
			return err
		}
	}

	//log.Infof("get cm pods success, pods num: %d", len(cmPods.Items))
	if expectCmPodNum != int32(len(cmPods.Items)) {
		//log.Infof("cm not build")
		return nil
	}

	//获取hostname模版和dnsdomain
	gbase8sHost, gbase8sDomain, err := GetHostTemplate(gbase8sPods)
	if err != nil {
		return err
	}

	cmHost, cmDomain, err := GetHostTemplate(cmPods)
	if err != nil {
		return err
	}

	trustStr := r.GenerateTrustString(len(gbase8sPods.Items), gbase8sHost, gbase8sDomain)
	trustStr += r.GenerateTrustString(len(cmPods.Items), cmHost, cmDomain)
	r.BuildTrust(gbase8sPods, trustStr)
	r.BuildGbase8sSqlhost(gbase8sPods, gbase8sHost, gbase8sDomain)
	cmSqlhostStr := r.GenerateCmSqlhostString(
		len(gbase8sPods.Items),
		len(cmPods.Items),
		gbase8sHost,
		gbase8sDomain,
		cmHost,
		cmDomain,
		cluster.Spec.CmCfg.RedirectGroupName,
		cluster.Spec.CmCfg.ProxyGroupName)
	r.BuildCmSqlhost(cmPods, cmSqlhostStr)

	return nil
}
