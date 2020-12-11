package controllers

import (
	gbase8sv1 "Gbase8sCluster/api/v1"
	"Gbase8sCluster/entity"
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

	//探测主节点
	primaryPod *NodeBasicInfo

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
	case FAILOVER_MSGTYPE_PRIMARYPOD:
		if v, ok := queueMsg.Data.(*NodeBasicInfo); ok {
			param.primaryPod = v
		} else {
			log.Error("Modify primary pod failed")
		}
	case FAILOVER_MSGTYPE_DETECTING:
		if v, ok := queueMsg.Data.(*detecting); ok {
			if param.detecting.detectingCount != v.detectingCount {
				param.detecting.detectingCount = v.detectingCount
				param.currFailedCount = 0
			}
			if param.detecting.detectingInterval != v.detectingInterval {
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

func getStatByHost(host, domain string, param *clusterManager) (*entity.ResponseData, error) {
	//url := fmt.Sprintf("http://%s.%s:%s/hac/getStatus", host, domain, getPort(host))
	url := fmt.Sprintf("http://192.168.70.2:%s/hac/getStatus", getPort(host))
	return param.detectingHttpClient.Get(url)
}

func execByHost(host, domain, cmd string, param *clusterManager) (string, string, error) {
	//url := fmt.Sprintf("http://%s.%s:%s/hac/exec", host, domain, getPort(host))
	url := fmt.Sprintf("http://192.168.70.2:%s/hac/exec", getPort(host))
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
	//发送探测请求
	primaryPod := param.primaryPod
	if primaryPod != nil {
		log.Infof("detecting %s", param.primaryPod.PodName)
		detectingFailed := false
		if resp, err := getStatByHost(primaryPod.HostName, primaryPod.Domain, param); err != nil {
			if strings.Contains(err.Error(), "timeout") ||
				strings.Contains(err.Error(), "Timeout") {
				detectingFailed = true
			}
		} else {
			if resp.Code != "0" {
				detectingFailed = true
			} else {
				if v, ok := resp.Data.(string); ok {
					if strings.Contains(v, "shared memory not initialized") {
						detectingFailed = true
					}
				}
			}
		}
		if detectingFailed {
			param.currFailedCount++
		} else {
			param.currFailedCount = 0
		}

		//进行故障转移
		if param.currFailedCount >= param.detecting.detectingCount {

			//获取secondary节点信息
			var nodeList []*NodeInfo
			for _, v := range *param.podList {
				if v.PodName != primaryPod.PodName {
					if resp, err := getStatByHost(v.HostName, v.Domain, param); err == nil {
						if resp.Code == "0" {
							if val, ok := resp.Data.(string); ok {
								nodeInfo, _ := ParseNodeInfo(v.PodName, v.HostName, v.Domain, val)
								nodeList = append(nodeList, nodeInfo)
							}
						}
					} else {
						log.Errorf("err: %s", err.Error())
					}
				}
			}

			//找一个secondary节点，切成主
			var secondaryNode *NodeInfo
			for _, v := range nodeList {
				if v.SourceServerName == primaryPod.ServerName {
					secondaryNode = v
					break
				}
			}

			//切主
			if secondaryNode != nil {
				cmdBuilder := strings.Builder{}
				for _, v := range *param.podList {
					if v.PodName != secondaryNode.PodName {
						cmdBuilder.WriteString(" && onmode -d add RSS ")
						cmdBuilder.WriteString(v.ServerName)
					}
				}

				if cmdBuilder.Len() != 0 {
					log.Infof("cluster %s %s failover to %s", param.clusterName, param.clusterNamespace, secondaryNode.PodName)
					_, stderr, err := execByHost(secondaryNode.HostName, secondaryNode.Domain, "source /env.sh && onmode -d standard"+cmdBuilder.String(), param)
					if err != nil {
						log.Errorf("failover failed, err: %s", err)
					} else {
						param.currFailedCount = 0
						param.primaryPod = &secondaryNode.NodeBasicInfo
						param.activeUpdateCluster <- 1
						if stderr != "" {
							log.Errorf("failover failed, message: %s", stderr)
						} else {
							for _, v := range nodeList {
								if (v.PodName != secondaryNode.PodName) && (v.ServerType == GBASE8S_ROLE_RSS) {
									_, stderr1, err := execByHost(v.HostName, v.Domain,
										"source /env.sh && echo \"0\" > check.conf && onmode -ky && oninit -PHY && onmode -d RSS "+secondaryNode.ServerName,
										param)
									_, _, _ = execByHost(v.HostName, v.Domain, "echo \"1\" > /check.conf", param)
									if err != nil {
										log.Errorf("failover failed, err: %s", err)
									} else {
										if stderr != "" {
											log.Errorf("failover failed, message: %s", stderr1)
										}
									}
								}
							}
						}
					}
				}
			} else {
				log.Errorf("cannot find failover node, cluster: %s, namespace: %s", param.clusterName, param.clusterNamespace)
			}
		}
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
func findRealGbase8sPrimary(nodes *[]NodeInfo) (*NodeInfo, error) {
	var primaryNode, secondaryNode, standardNode []*NodeInfo
	for _, v := range *nodes {
		if v.ServerType == GBASE8S_ROLE_PRIMARY {
			if v.Connected {
				return &v, nil
			}
			primaryNode = append(primaryNode, &v)
		} else if v.ServerType == GBASE8S_ROLE_RSS {
			secondaryNode = append(secondaryNode, &v)
		} else {
			standardNode = append(standardNode, &v)
		}
	}

	if (len(standardNode) != 0) && (len(standardNode) == len(*nodes)) {
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

func isGbase8sClusterNormal(nodes *[]NodeInfo) bool {
	for _, v := range *nodes {
		if !v.Connected {
			return false
		}
	}
	return true
}

func (c *ClusterManager) getNodesRssStatus(pods *corev1.PodList, domain string) (*[]NodeInfo, error) {
	var nodes []NodeInfo
	onstatCmd := []string{"bash", "-c", "source /env.sh && onstat -g rss verbose"}
	for _, v := range pods.Items {
		stdout, stderr, err := c.execClient.Exec(onstatCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
		if err != nil {
			if !strings.Contains(stderr, "shared memory not initialized") {
				return nil, errors.New("get rss status failed, error: " + err.Error() + " " + stderr)
			}
		}

		nodeInfo, _ := ParseNodeInfo(v.Name, v.Spec.Hostname, domain, stdout)
		nodes = append(nodes, *nodeInfo)
	}

	return &nodes, nil
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

	//gPodLabels := map[string]string{
	//	GBASE8S_POD_LABEL_KEY: GBASE8S_POD_LABEL_VALUE_PREFIX + gbase8sCluster.Name,
	//}
	//gPods := &corev1.PodList{}
	//opts := &client.ListOptions{
	//	Namespace:     param.clusterNamespace,
	//	LabelSelector: labels.SelectorFromSet(gPodLabels),
	//}
	//if err := c.List(ctx, gPods, opts); err != nil {
	//	return errors.New("get gbase8s pods failed, err: " + err.Error())
	//}

	if gbase8sExpectReplicas != int32(len(gPods.Items)) {
		return errors.New("wait")
	}

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

	//获取所有容器的rss状态
	nodes, err := c.getNodesRssStatus(gPods, (*param.podList)[0].Domain)
	if err != nil {
		return err
	}

	needWait := false
	for _, v := range *nodes {
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

	p, err := findRealGbase8sPrimary(nodes)
	if err != nil {
		return err
	}

	//向主节点添加辅节点
	var addNodes strings.Builder
	for _, v := range *nodes {
		if v.PodName != p.PodName && !v.Connected {
			bfind := false
			for _, vs := range p.SubStatus {
				if vs.ServerName == v.ServerName {
					bfind = true
					break
				}
			}
			if !bfind {
				addNodes.WriteString("&& onmode -d add RSS ")
				addNodes.WriteString(v.ServerName)
			}
		}
	}
	if addNodes.Len() != 0 {
		addCmd := []string{"bash", "-c", "source /env.sh" + addNodes.String()}
		log.Infof("primary: %s, add cmd: %s", p.PodName, addNodes.String())
		_, stderr, err := c.execClient.Exec(addCmd, GBASE8S_CONTAINER_NAME, p.PodName, param.clusterNamespace, nil)
		if err != nil {
			return errors.New(fmt.Sprintf("add rss failed, exec pod: %s, err: %s, %s", p.PodName, err.Error(), stderr))
		}
	}

	//辅节点加入集群
	for _, v := range *nodes {
		if v.PodName != p.PodName && !v.Connected {
			rssCmd := []string{"bash", "-c", "source /env.sh && curl -o tape http://" +
				p.PodName +
				"." +
				(*param.podList)[0].Domain +
				":8000/hac/getTape && sh /recover.sh tape && onmode -d RSS " +
				p.ServerName}
			log.Infof("secondary: %s, add to cluster", v.PodName)
			_, stderr, err := c.execClient.Exec(rssCmd, GBASE8S_CONTAINER_NAME, v.PodName, param.clusterNamespace, nil)
			if err != nil {
				return errors.New(fmt.Sprintf("exec rss failed, exec pod: %s, err: %s, %s", v.PodName, err.Error(), stderr))
			}
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

	//cmPodLabels := map[string]string{
	//	CM_POD_LABEL_KEY: CM_POD_LABEL_VALUE_PREFIX + gbase8sCluster.Name,
	//}
	//cmPods := &corev1.PodList{}
	//opts := &client.ListOptions{
	//	Namespace:     param.clusterName,
	//	LabelSelector: labels.SelectorFromSet(cmPodLabels),
	//}
	//if err := c.List(ctx, cmPods, opts); err != nil {
	//	return errors.New("get cm pods failed, err: " + err.Error())
	//}

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
		if stdout == "1\n" {
			//if needReloadCm {
			//	log.Infof("reload cm %s config", v.Name)
			//	stdout, stderr, err := c.execClient.Exec(reloadCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
			//	if err != nil {
			//		return errors.New(fmt.Sprintf("reload cm failed, err: %s %s %s", err.Error(), stderr, stdout))
			//	}
			//}
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

	var pNode *NodeInfo
	for _, v := range *nodes {
		if v.ServerType == GBASE8S_ROLE_PRIMARY {
			pNode = &v
			break
		}
	}

	//更新主节点
	if pNode != nil {
		param.primaryPod = &NodeBasicInfo{
			PodName:    pNode.PodName,
			Namespace:  pNode.Namespace,
			HostName:   pNode.HostName,
			Domain:     pNode.Domain,
			ServerName: pNode.ServerName,
		}
		return nil
	} else {
		return errors.New("find primary node failed")
	}
}

func (c *ClusterManager) stopDetecting(param *clusterManager) {
	param.primaryPod = nil
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
					param.updateGbase8sClusterTicker.Stop()
					log.Errorf("process update gbase8s cluster failed, err: %s", err.Error())
				}
			} else {
				param.updateGbase8sClusterTicker.Stop()
				param.updateCmClusterTicker.Reset(2 * time.Second)
				if err := c.startDetecting(param); err != nil {
					log.Errorf("start detecting %s %s failed, err: %s", param.clusterName, param.clusterNamespace, err.Error())
				}
			}
		case <-param.updateCmClusterTicker.C:
			if err := c.procUpdateCmCluster(param); err != nil {
				if err.Error() != "wait" {
					param.updateCmClusterTicker.Stop()
					log.Errorf("process update cm cluster failed, err: %s", err.Error())
				}
			} else {
				param.updateCmClusterTicker.Stop()
			}
		case <-param.activeUpdateCluster:
			param.updateGbase8sClusterTicker.Reset(3 * time.Second)
		case <-param.destroyClusterManager:
			stop = true
			break
		}

		if stop {
			param.updateCmClusterTicker.Stop()
			param.updateGbase8sClusterTicker.Stop()
			param.detectingTicker.Stop()
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
