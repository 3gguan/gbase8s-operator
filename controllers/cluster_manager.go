package controllers

import (
	gbase8sv1 "Gbase8sCluster/api/v1"
	"Gbase8sCluster/util"
	"context"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"sync"
	"time"
)

type detecting struct {
	//探测次数
	detectingCount int

	//探测时间间隔
	detectingInterval int

	//探测超时时间
	timeout int
}

type CfgQueueMsg struct {
	MsgType int
	Data    interface{}
}

type clusterManager struct {
	clusterName      string
	clusterNamespace string

	detecting *detecting

	//当前探测失败次数
	currFailedCount int

	//探测标志
	detectingFlag bool

	//探测主节点
	primaryPod *NodeInfo

	//所有节点
	podList *[]*NodeBasicInfo

	//detecting http client
	detectingHttpClient *util.HttpClient

	//general http client
	generalHttpClient *util.HttpClient

	//探测定时器
	detectingTicker *time.Ticker

	//配置消息队列
	cfgQueue chan *CfgQueueMsg

	////停止探测信号
	//stopDetectingChannel chan int

	//调整gbase8s集群定时器
	updateGbase8sClusterTicker *time.Ticker

	//调整cm集群定时器
	updateCmClusterTicker *time.Ticker

	//激活调整集群信号
	activeUpdateCluster chan int

	//销毁集群管理信号
	destroyClusterManager chan int
}

type ClusterManager struct {
	client.Client
	execClient *util.ExecInPod
	clusterMap map[string]*clusterManager
	rwLock     sync.RWMutex
}

func InitClusterManager(client client.Client, execClient *util.ExecInPod) *ClusterManager {
	return &ClusterManager{
		Client:     client,
		execClient: execClient,
		clusterMap: make(map[string]*clusterManager),
	}
}

func (c *ClusterManager) procModifyParam(param *clusterManager, queueMsg *CfgQueueMsg) {
	switch queueMsg.MsgType {
	//case FAILOVER_MSGTYPE_PRIMARYPOD:
	//	if v, ok := queueMsg.Data.(*NodeBasicInfo); ok {
	//		param.primaryPod = v
	//	} else {
	//		log.Error("Modify primary pod failed")
	//	}
	case FAILOVER_MSGTYPE_DETECTING:
		if v, ok := queueMsg.Data.(*detecting); ok {
			if param.detecting.detectingCount != v.detectingCount {
				log.Infof("modify detecting count for cluster %s %s", param.clusterName, param.clusterNamespace)
				param.detecting.detectingCount = v.detectingCount
				param.currFailedCount = 0
			}
			if param.detecting.detectingInterval != v.detectingInterval {
				log.Infof("modify detecting interval for cluster %s %s", param.clusterName, param.clusterNamespace)
				param.detecting.detectingInterval = v.detectingInterval
				param.currFailedCount = 0
				param.detectingTicker.Reset(time.Duration(v.detectingInterval) * time.Second)
			}
			if param.detecting.timeout != v.timeout {
				param.detecting.timeout = v.timeout
				param.detectingHttpClient = util.NewHttpClient().SetTimeout(time.Duration(v.timeout))
			}
		} else {
			log.Error("Modify detecting param failed")
		}
	case FAILOVER_MSGTYPE_PODLIST:
		if v, ok := queueMsg.Data.(*[]*NodeBasicInfo); ok {
			param.podList = v
		} else {
			log.Error("Modify pod list failed")
		}
	default:
		log.Errorf("Modify param failed, msg type error: %d", queueMsg.MsgType)
	}
}

func getPort(podName string) string {
	aa := strings.Split(podName, "-")
	return "3111" + aa[len(aa)-1]
}

func getStatByHost(host, domain string, param *clusterManager) (string, error) {
	log.Infof("get status by host: %s.%s", host, domain)
	url := fmt.Sprintf("http://%s.%s:%s/hac/getStatus", host, domain, GBASE8S_CONFIG_PORT)
	//url := fmt.Sprintf("http://192.168.70.2:%s/hac/getStatus", getPort(host))
	resp, err := param.detectingHttpClient.Get(url)
	if err != nil {
		return "", errors.New(fmt.Sprintf("get status by host failed, err: %s", err.Error()))
	}

	if resp.Code != "0" {
		return "", errors.New(fmt.Sprintf("get status by host failed, resp code: %s, message: %s", resp.Code, resp.Message))
	}

	if v, ok := resp.Data.(string); ok {
		return v, nil
	} else {
		return "", errors.New("get status by host failed, response data type error")
	}
}

func execByHost(host, domain, cmd string, param *clusterManager) (string, string, error) {
	url := fmt.Sprintf("http://%s.%s:%s/hac/exec", host, domain, GBASE8S_CONFIG_PORT)
	//url := fmt.Sprintf("http://192.168.70.2:%s/hac/exec", getPort(host))
	resp, err := param.generalHttpClient.Post(url, map[string]string{
		"cmd": cmd,
	})

	var stdout, stderr string
	if err == nil {
		if v, ok := resp.Data.(map[string]interface{}); ok {
			out := v["stdout"]
			err := v["stderr"]
			if vo, ok := out.(string); ok {
				stdout = vo
			}
			if eo, ok := err.(string); ok {
				stderr = eo
			}
		}
	}

	return stdout, stderr, err
}

func (c *ClusterManager) procFailover(param *clusterManager) {
	if !param.detectingFlag {
		return
	}
	primaryPod := param.primaryPod
	if primaryPod == nil {
		return
	}

	//发送探测请求
	detectingFailed := false
	var tmpErr error
	if resp, err := getStatByHost(primaryPod.HostName, primaryPod.Domain, param); err != nil {
		detectingFailed = true
		tmpErr = err
	} else {
		if strings.Contains(resp, "shared memory not initialized") {
			detectingFailed = true
		}
	}
	if detectingFailed {
		if tmpErr != nil {
			log.Infof("detecting %s failed, err: %s", param.primaryPod.PodName, tmpErr.Error())
		} else {
			log.Infof("detecting %s failed", param.primaryPod.PodName)
		}

		log.Infof("failed count: %d, total count: %d", param.currFailedCount, param.detecting.detectingCount)
		param.currFailedCount++
	} else {
		log.Infof("detecting %s success", param.primaryPod.PodName)
		param.currFailedCount = 0
	}

	//进行故障转移
	if param.currFailedCount >= param.detecting.detectingCount {
		log.Infof("cluster %s %s failover start", param.clusterName, param.clusterNamespace)
		//获取secondary节点信息
		var nodeList []*NodeInfo
		for _, v := range *param.podList {
			if v.PodName != primaryPod.PodName {
				if resp, err := getStatByHost(v.HostName, v.Domain, param); err == nil {
					nodeInfo, _ := ParseNodeInfo(v.PodName, v.HostName, v.Domain, resp)
					nodeList = append(nodeList, nodeInfo)
				} else {
					log.Errorf("err: %s", err.Error())
				}
			}
		}

		logStr := ""
		for _, v := range nodeList {
			logStr += v.PodName
			logStr += " "
		}
		log.Infof("all node: %s", logStr)

		//找一个secondary节点，切成主
		var secondaryNodeList []*NodeInfo
		for i := 0; i < len(nodeList); i++ {
			if nodeList[i].SourceServerName == primaryPod.ServerName {
				secondaryNodeList = append(secondaryNodeList, nodeList[i])
			}
		}

		logStr = ""
		for _, v := range nodeList {
			logStr += v.PodName
			logStr += " "
		}
		log.Infof("secondary node: %s", logStr)

		//切主
		if len(secondaryNodeList) != 0 {
			for _, secondaryNode := range secondaryNodeList {
				//secondaryNode := v
				cmdBuilder := strings.Builder{}
				var addNodesTemp []string
				for _, v := range *param.podList {
					if v.PodName != secondaryNode.PodName {
						cmdBuilder.WriteString(" && onmode -d add RSS ")
						cmdBuilder.WriteString(v.ServerName)
						addNodesTemp = append(addNodesTemp, v.ServerName)
					}
				}

				if cmdBuilder.Len() != 0 {
					log.Infof("cluster %s %s failover to %s", param.clusterName, param.clusterNamespace, secondaryNode.PodName)
					_, stderr, err := execByHost(secondaryNode.HostName, secondaryNode.Domain, "source /env.sh && onmode -d standard"+cmdBuilder.String(), param)
					if err != nil {
						log.Errorf("failover failed, err: %s", err)
						continue
					} else {
						param.currFailedCount = 0
						param.primaryPod = secondaryNode
						for _, v := range addNodesTemp {
							param.primaryPod.SubStatus = append(param.primaryPod.SubStatus, SubStatus{ServerName: v, Connected: false})
						}

						if stderr != "" {
							log.Errorf("cluster %s %s failover failed, message: %s", param.clusterName, param.clusterNamespace, stderr)
						} else {
							for _, node := range nodeList {
								if (node.PodName != secondaryNode.PodName) && (node.ServerType == GBASE8S_ROLE_RSS) {
									rssCmd := "source /env.sh && curl -o tape http://" +
										secondaryNode.PodName +
										"." +
										secondaryNode.Domain +
										":8000/hac/getTape && sh /recover.sh tape && onmode -d RSS " +
										secondaryNode.ServerName
									log.Infof("secondary: %s, add to cluster", node.PodName)
									_, stderr, err := execByHost(node.HostName, node.Domain, rssCmd, param)
									if err != nil {
										log.Errorf("secondary %s add to cluster failed, err: %s stderr: %s", node.PodName, err.Error(), stderr)
									} else {
										for _, v := range param.primaryPod.SubStatus {
											if v.ServerName == node.ServerName {
												v.Connected = true
												break
											}
										}
									}
								}
							}
							//for _, v := range nodeList {
							//	if (v.PodName != secondaryNode.PodName) && (v.ServerType == GBASE8S_ROLE_RSS) {
							//		_, stderr1, err := execByHost(v.HostName, v.Domain,
							//			"source /env.sh && echo \"0\" > check.conf && onmode -ky && oninit -PHY && onmode -d RSS "+secondaryNode.ServerName,
							//			param)
							//		_, _, _ = execByHost(v.HostName, v.Domain, "echo \"1\" > /check.conf", param)
							//		if err != nil {
							//			log.Errorf("failover failed, err: %s", err)
							//		} else {
							//			if stderr != "" {
							//				log.Errorf("failover failed, message: %s", stderr1)
							//			}
							//		}
							//	}
							//}
						}

						//激活更新集群
						select {
						case param.activeUpdateCluster <- 1:
						default:
						}

						break
					}
				} else {
					log.Infof("cluster %s %s failover failed, secondary node in primary %s", param.clusterName, param.clusterNamespace, secondaryNode.PodName)
				}
			}
		} else {
			log.Errorf("cannot find failover node, cluster: %s, namespace: %s", param.clusterName, param.clusterNamespace)
		}

		log.Infof("cluster %s %s failover end", param.clusterName, param.clusterNamespace)
	}
}

/*
以4个节点为例

初始化：
4标准。---返回数组中第一个节点作为主

正常：
1主 3备。 ---无动作

新增：
1主 3备 2标准。 ---标准变备

异常：
1废主 3备；2废主 2备；3废主 1备。 ---等待切主
1废主 1主 2备。 ---已经切主，废主变备
4废主。 ---不知所措
*/
func findRealGbase8sPrimary(nodes []*NodeInfo, param *clusterManager) (*NodeInfo, error) {
	if param.primaryPod != nil {
		return param.primaryPod, nil
		//for _, v := range nodes {
		//	if param.primaryPod.PodName == v.PodName {
		//		return v, nil
		//	}
		//}
		//
		//return errors.New("cannot find primary node, it ca")
		//for i := 0; i < len(nodes); i++ {
		//	v := &(nodes)[i]
		//	if param.primaryPod.PodName == v.PodName {
		//		return v, nil
		//	}
		//}
	}

	var primaryNode, secondaryNode, standardNode []*NodeInfo
	for _, v := range nodes {
		if v.ServerType == GBASE8S_ROLE_PRIMARY {
			if v.Connected {
				return v, nil
			}
			primaryNode = append(primaryNode, v)
		} else if v.ServerType == GBASE8S_ROLE_RSS {
			secondaryNode = append(secondaryNode, v)
		} else {
			standardNode = append(standardNode, v)
		}
	}
	//for i := 0; i < len(nodes); i++ {
	//	v := &(nodes)[i]
	//	if v.ServerType == GBASE8S_ROLE_PRIMARY {
	//		if v.Connected {
	//			return v, nil
	//		}
	//		primaryNode = append(primaryNode, v)
	//	} else if v.ServerType == GBASE8S_ROLE_RSS {
	//		secondaryNode = append(secondaryNode, v)
	//	} else {
	//		standardNode = append(standardNode, v)
	//	}
	//}

	if (len(standardNode) != 0) && (len(standardNode) == len(nodes)) {
		pNode := standardNode[0]
		for _, v := range standardNode {
			if v.PodName < pNode.PodName {
				pNode = v
			}
		}
		return pNode, nil
	}

	if len(primaryNode) != 0 {
		return primaryNode[0], nil
	}

	return nil, errors.New("cannot find primary node")

	//var primaryNum, secondaryNum, standardNum int
	//for _, v := range *nodes {
	//	if v.ServerType == GBASE8S_ROLE_PRIMARY {
	//		primaryNum++
	//		if v.Connected {
	//			return &v, nil
	//		}
	//	} else if v.ServerType == GBASE8S_ROLE_RSS {
	//		secondaryNum++
	//	} else {
	//		standardNum++
	//	}
	//}
	//if standardNum == len(*nodes) {
	//	return &((*nodes)[0]), nil
	//}
	//if secondaryNum != 0 {
	//	return nil, errors.New("wait")
	//} else if primaryNum == len(*nodes) {
	//	return nil, errors.New("all nodes damaged")
	//} else {
	//	return nil, errors.New("internal error")
	//}
}

func isGbase8sClusterNormal(nodes []*NodeInfo) bool {
	for _, v := range nodes {
		if !v.Connected {
			return false
		}
	}
	return true
}

func (c *ClusterManager) getNodesRssStatus(pods *corev1.PodList, domain string) ([]*NodeInfo, error) {
	var nodes []*NodeInfo
	onstatCmd := []string{"bash", "-c", "source /env.sh && onstat -g rss verbose"}
	for _, v := range pods.Items {
		stdout, stderr, err := c.execClient.Exec(onstatCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
		if err != nil {
			if !strings.Contains(stderr, "shared memory not initialized") {
				return nil, errors.New(fmt.Sprintf("get %s %s rss status failed, error: %s %s", v.Name, v.Namespace, err.Error(), stderr))
			}
		}

		nodeInfo, _ := ParseNodeInfo(v.Name, v.Spec.Hostname, domain, stdout)
		nodes = append(nodes, nodeInfo)
	}

	return nodes, nil
}

func (c *ClusterManager) getOnlineNodesRssStatus(pods *corev1.PodList, domain string) []*NodeInfo {
	var nodes []*NodeInfo
	onstatCmd := []string{"bash", "-c", "source /env.sh && onstat -g rss verbose"}
	for _, v := range pods.Items {
		stdout, _, err := c.execClient.Exec(onstatCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
		if err == nil {
			nodeInfo, _ := ParseNodeInfo(v.Name, v.Spec.Hostname, domain, stdout)
			nodes = append(nodes, nodeInfo)
		}
	}

	return nodes
}

func (c *ClusterManager) isAllNodesServiceRunning(pods *corev1.PodList) bool {
	isAllRunning := true
	isRunningCmd := []string{"bash", "-c", "ps -ef | grep runserver | grep -v grep | wc -l"}
	for _, v := range pods.Items {
		stdout, _, err := c.execClient.Exec(isRunningCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
		if err != nil {
			log.Infof("pod %s %s is not running, err: %s", v.Name, v.Namespace, err.Error())
			isAllRunning = false
			break
		} else {
			if stdout != "2\n" && stdout != "2\r\n" && stdout != "2" {
				log.Infof("pod %s %s is not running", v.Name, v.Namespace)
				isAllRunning = false
				break
			}
		}
	}

	return isAllRunning
}

//添加辅节点
func (c *ClusterManager) addSecondary(primary *NodeInfo, onlineNodes []*NodeInfo, namespace string) error {
	var addNodes strings.Builder
	var addNodesTemp []*NodeInfo
	for _, v := range onlineNodes {
		if v.PodName != primary.PodName {
			bfind := false
			for _, vs := range primary.SubStatus {
				if vs.ServerName == v.ServerName {
					bfind = true
					break
				}
			}
			if !bfind {
				addNodes.WriteString("&& onmode -d add RSS ")
				addNodes.WriteString(v.ServerName)
				addNodesTemp = append(addNodesTemp, v)
			}
		}
	}
	if addNodes.Len() != 0 {
		addCmd := []string{"bash", "-c", "source /env.sh" + addNodes.String()}
		log.Infof("primary: %s, add cmd: %s", primary.PodName, addNodes.String())
		_, stderr, err := c.execClient.Exec(addCmd, GBASE8S_CONTAINER_NAME, primary.PodName, namespace, nil)
		if err != nil {
			return errors.New(fmt.Sprintf("add rss failed, exec pod: %s, err: %s, %s", primary.PodName, err.Error(), stderr))
		} else {
			for _, v := range addNodesTemp {
				primary.SubStatus = append(primary.SubStatus, SubStatus{ServerName: v.ServerName, Connected: false})
			}
		}
	}

	return nil
}

//设置为辅节点
func (c *ClusterManager) setRssServerType(primary *NodeInfo, onlineNodes []*NodeInfo, namespace string) error {
	hasError := false
	for _, v := range onlineNodes {
		if (v.PodName != primary.PodName) && (v.SourceServerName != primary.ServerName) {
			bFind := false
			for _, status := range primary.SubStatus {
				if status.ServerName == v.ServerName {
					bFind = true
				}
			}
			if !bFind {
				hasError = true
				continue
			}

			rssCmd := []string{"bash", "-c", "source /env.sh && curl -o tape http://" +
				primary.PodName +
				"." +
				primary.Domain +
				":8000/hac/getTape && sh /recover.sh tape && onmode -d RSS " +
				primary.ServerName}
			log.Infof("secondary: %s, add to cluster", v.PodName)
			_, stderr, err := c.execClient.Exec(rssCmd, GBASE8S_CONTAINER_NAME, v.PodName, namespace, nil)
			if err != nil {
				log.Error(fmt.Sprintf("exec rss failed, exec pod: %s, err: %s, %s", v.PodName, err.Error(), stderr))
				hasError = true
			}
		}
	}

	if hasError {
		return errors.New("")
	}
	return nil
}

func (c *ClusterManager) procUpdateGbase8sCluster(param *clusterManager) error {
	log.Infof("process update gbase8s cluster %s %s", param.clusterName, param.clusterNamespace)

	//获取期望pod个数
	ctx := context.Background()
	var gbase8sExpectReplicas int32
	var gbase8sCluster gbase8sv1.Gbase8sCluster
	reqTemp := types.NamespacedName{
		Name:      param.clusterName,
		Namespace: param.clusterNamespace,
	}
	if err := c.Get(ctx, reqTemp, &gbase8sCluster); err != nil {
		return errors.New("update cluster failed, cannot get gbase8s cluster, error: " + err.Error())
	}
	gbase8sExpectReplicas = gbase8sCluster.Spec.Gbase8sCfg.Replicas
	if gbase8sExpectReplicas <= 1 {
		return errors.New("at least 2 gbase8s nodes needed")
	}

	//获取所有gbase8s pod
	gPods, err := GetAllPods(param.clusterNamespace, &map[string]string{
		GBASE8S_POD_LABEL_KEY: GBASE8S_POD_LABEL_VALUE_PREFIX + gbase8sCluster.Name,
	})
	if err != nil {
		return err
	}

	if gbase8sExpectReplicas != int32(len(gPods.Items)) {
		return errors.New("wait")
	}

	if param.primaryPod != nil {
		nodes := c.getOnlineNodesRssStatus(gPods, (*param.podList)[0].Domain)
		if len(nodes) == 0 {
			return errors.New("wait")
		}
		if err := c.addSecondary(param.primaryPod, nodes, param.clusterNamespace); err != nil {
			return errors.New("wait")
		}
		if err := c.setRssServerType(param.primaryPod, nodes, param.clusterNamespace); err != nil {
			return errors.New("wait")
		}
		if len(nodes) != len(gPods.Items) {
			return errors.New("wait")
		}
	} else {
		//如果有pod没在运行状态，等待
		flag := 1
		for _, v := range gPods.Items {
			if len(v.Status.ContainerStatuses) != 0 {
				if v.Status.ContainerStatuses[0].State.Running == nil {
					flag = 0
					break
				}
			}
		}
		if flag == 0 {
			return errors.New("wait")
		}

		//判断容器内服务是否已经启动，没启动就等待
		if !c.isAllNodesServiceRunning(gPods) {
			return errors.New("wait")
		}

		//获取所有容器的rss状态
		nodes, err := c.getNodesRssStatus(gPods, (*param.podList)[0].Domain)
		if err != nil {
			log.Errorf("update gbase8s cluster failed, %s", err.Error())
			return errors.New("wait")
		}

		needWait := false
		for _, v := range nodes {
			if v.ServerStatus == GBASE8S_STATUS_NONE ||
				v.ServerStatus == GBASE8S_STATUS_INIT ||
				v.ServerStatus == GBASE8S_STATUS_FAST_RECOVERY {
				needWait = true
				break
			}
		}
		if needWait {
			return errors.New("wait")
		}

		if isGbase8sClusterNormal(nodes) {
			return nil
		}

		p, err := findRealGbase8sPrimary(nodes, param)
		if err != nil {
			return err
		}

		if err := c.addSecondary(p, nodes, param.clusterNamespace); err != nil {
			return errors.New("wait")
		}
		if err := c.setRssServerType(p, nodes, param.clusterNamespace); err != nil {
			return errors.New("wait")
		}

	}

	return nil
}

func (c *ClusterManager) procUpdateCmCluster(param *clusterManager) error {
	log.Infof("process update cm cluster %s %s", param.clusterName, param.clusterNamespace)

	ctx := context.Background()
	var cmExpectReplicas int32
	var gbase8sCluster gbase8sv1.Gbase8sCluster
	reqTemp := types.NamespacedName{
		Name:      param.clusterName,
		Namespace: param.clusterNamespace,
	}
	if err := c.Get(ctx, reqTemp, &gbase8sCluster); err != nil {
		return errors.New("update cluster failed, cannot get gbase8s cluster, error: %s" + err.Error())
	}

	cmExpectReplicas = gbase8sCluster.Spec.CmCfg.Replicas
	if cmExpectReplicas < 1 {
		return errors.New("at least 1 cm nodes needed")
	}

	//获取cm pods
	cmPods, err := GetAllPods(param.clusterNamespace, &map[string]string{
		CM_POD_LABEL_KEY: CM_POD_LABEL_VALUE_PREFIX + gbase8sCluster.Name,
	})
	if err != nil {
		return err
	}

	if cmExpectReplicas != int32(len(cmPods.Items)) {
		return errors.New("wait")
	}

	//如果有pod没在运行状态，等待
	flag := 1
	for _, v := range cmPods.Items {
		if len(v.Status.ContainerStatuses) != 0 {
			if v.Status.ContainerStatuses[0].State.Running == nil {
				flag = 0
				break
			}
		}
	}
	if flag == 0 {
		return errors.New("wait")
	}

	//启动cm或重新加载cm配置
	statCmd := []string{"bash", "-c", "ps -ef|grep oncmsm|grep -v grep|wc -l"}
	startCmd := []string{"bash", "-c", "source /env.sh && sh start_manual.sh"}
	//reloadCmd := []string{"bash", "-c", "source /env.sh && oncmsm -r -c /opt/gbase8s/etc/cfg.cm"}
	for _, v := range cmPods.Items {
		stdout, stderr, err := c.execClient.Exec(statCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
		if err != nil {
			return errors.New(fmt.Sprintf("get cm status failed, err: %s %s", err.Error(), stderr))
		}
		if stdout == "1\n" || stdout == "1\r\n" || stdout == "1" {
			//if needReloadCm {
			//	log.Infof("reload cm %s config", v.Name)
			//	stdout, stderr, err := c.execClient.Exec(reloadCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
			//	if err != nil {
			//		return errors.New(fmt.Sprintf("reload cm failed, err: %s %s %s", err.Error(), stderr, stdout))
			//	}
			//}
			log.Infof("cm %s is normal", v.Name)
		} else {
			log.Infof("start cm %s", v.Name)
			_, stderr, err := c.execClient.Exec(startCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
			if err != nil {
				return errors.New(fmt.Sprintf("start cm failed, err: %s %s", err.Error(), stderr))
			}
		}
	}

	return nil
}

func (c *ClusterManager) startDetecting(param *clusterManager) error {
	param.detectingFlag = true

	if param.primaryPod != nil {
		return nil
	}

	//获取主节点
	pods, err := GetAllPods(param.clusterNamespace, &map[string]string{
		GBASE8S_POD_LABEL_KEY: GBASE8S_POD_LABEL_VALUE_PREFIX + param.clusterName,
	})
	if err != nil {
		return err
	}
	nodes, err := c.getNodesRssStatus(pods, (*param.podList)[0].Domain)
	if err != nil {
		return err
	}

	for _, v := range nodes {
		if v.ServerType == GBASE8S_ROLE_PRIMARY {
			param.primaryPod = v
			break
		}
	}

	if param.primaryPod == nil {
		return errors.New("find primary node failed")
	}

	return nil
}

func (c *ClusterManager) stopDetecting(param *clusterManager) {
	param.detectingFlag = false
}

func startTimer(ticker *time.Ticker, d time.Duration) {
	ticker.Reset(d)
}

func stopTimer(ticker *time.Ticker) {
	ticker.Stop()
	isOver := false
	for {
		select {
		case <-ticker.C:
		default:
			isOver = true
			break
		}
		if isOver {
			break
		}
	}
}

func (c *ClusterManager) clusterThread(param *clusterManager) {
	log.Infof("cluster thread %s %s start", param.clusterName, param.clusterNamespace)
	stop := false
	for {
		select {
		case queueMsg := <-param.cfgQueue:
			c.procModifyParam(param, queueMsg)
		case <-param.detectingTicker.C:
			c.procFailover(param)
		case <-param.updateGbase8sClusterTicker.C:

			if err := c.procUpdateGbase8sCluster(param); err != nil {
				if err.Error() != "wait" {
					stopTimer(param.updateGbase8sClusterTicker)
					log.Errorf("process update gbase8s cluster failed, err: %s", err.Error())
				}
			} else {
				stopTimer(param.updateGbase8sClusterTicker)
				startTimer(param.updateCmClusterTicker, 2*time.Second)
				if err := c.startDetecting(param); err != nil {
					log.Errorf("start detecting %s %s failed, err: %s", param.clusterName, param.clusterNamespace, err.Error())
				}
			}
		case <-param.updateCmClusterTicker.C:
			if err := c.procUpdateCmCluster(param); err != nil {
				if err.Error() != "wait" {
					stopTimer(param.updateCmClusterTicker)
					log.Errorf("process update cm cluster failed, err: %s", err.Error())
				}
			} else {
				stopTimer(param.updateCmClusterTicker)
			}
		case <-param.activeUpdateCluster:
			startTimer(param.updateGbase8sClusterTicker, 3*time.Second)
		case <-param.destroyClusterManager:
			stop = true
			break
		}

		if stop {
			stopTimer(param.updateCmClusterTicker)
			stopTimer(param.updateGbase8sClusterTicker)
			stopTimer(param.detectingTicker)
			close(param.activeUpdateCluster)
			close(param.destroyClusterManager)

			break
		}
	}
	log.Infof("cluster thread %s %s end", param.clusterName, param.clusterNamespace)
}

func (c *ClusterManager) modifyParam(param *clusterManager, msgType int, data interface{}) {
	msg := CfgQueueMsg{
		MsgType: msgType,
		Data:    data,
	}
	param.cfgQueue <- &msg
}

func (c *ClusterManager) UpdateCluster(clusterName, clusterNamespace string) {
	name := clusterName + "|" + clusterNamespace
	c.rwLock.Lock()
	if v, ok := c.clusterMap[name]; ok {
		select {
		case v.activeUpdateCluster <- 1:
			log.Infof("write msg to %s %s", clusterName, clusterNamespace)
		default:
			log.Infof("write msg to %s %s block, drop msg", clusterName, clusterNamespace)
		}
	}
	c.rwLock.Unlock()
}

func (c *ClusterManager) AddCluster(clusterName, clusterNamespace string, podList *[]*NodeBasicInfo, detectingCount, detectingInterval, timeout int) {
	name := clusterName + "|" + clusterNamespace
	c.rwLock.Lock()
	if v, ok := c.clusterMap[name]; ok {
		c.modifyParam(v, FAILOVER_MSGTYPE_PODLIST, podList)
		detecting := &detecting{
			detectingCount:    detectingCount,
			detectingInterval: detectingInterval,
			timeout:           timeout,
		}
		c.modifyParam(v, FAILOVER_MSGTYPE_DETECTING, detecting)
		c.rwLock.Unlock()
	} else {
		cm := clusterManager{
			clusterName:      clusterName,
			clusterNamespace: clusterNamespace,
			detecting: &detecting{
				detectingCount:    detectingCount,
				detectingInterval: detectingInterval,
				timeout:           timeout,
			},
			currFailedCount:            0,
			detectingFlag:              false,
			primaryPod:                 nil,
			podList:                    podList,
			detectingHttpClient:        util.NewHttpClient().SetTimeout(time.Duration(timeout)),
			detectingTicker:            time.NewTicker(time.Duration(detectingInterval) * time.Second),
			generalHttpClient:          util.NewHttpClient().SetTimeout(10 * 60),
			cfgQueue:                   make(chan *CfgQueueMsg),
			updateGbase8sClusterTicker: time.NewTicker(2 * time.Second),
			updateCmClusterTicker:      time.NewTicker(2 * time.Second),
			activeUpdateCluster:        make(chan int, 1),
			destroyClusterManager:      make(chan int),
		}

		//cm.detectingTicker.Stop()
		cm.updateGbase8sClusterTicker.Stop()
		cm.updateCmClusterTicker.Stop()

		c.clusterMap[name] = &cm
		c.rwLock.Unlock()
		go c.clusterThread(&cm)
	}
}

func (c *ClusterManager) DelCluster(clusterName, clusterNamespace string) {
	name := clusterName + "|" + clusterNamespace
	c.rwLock.Lock()
	if v, ok := c.clusterMap[name]; ok {
		v.destroyClusterManager <- 1
		delete(c.clusterMap, name)
	}
	c.rwLock.Unlock()
}
