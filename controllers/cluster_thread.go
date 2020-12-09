package controllers

//import (
//	gbase8sv1 "Gbase8sCluster/api/v1"
//	"Gbase8sCluster/util"
//	"context"
//	"errors"
//	"fmt"
//	corev1 "k8s.io/api/core/v1"
//	"k8s.io/apimachinery/pkg/labels"
//	"k8s.io/apimachinery/pkg/types"
//	"sigs.k8s.io/controller-runtime/pkg/client"
//	"strings"
//	"sync"
//	"time"
//)
//
//type QueueMsg struct {
//	Name      string
//	Namespace string
//}
//
//type ClusterThread struct {
//	client.Client
//	exeClient *util.ExecInPod
//	queueMap  map[string]chan QueueMsg
//	lock      sync.Mutex
//}
//
//func NewClusterThread(c client.Client, ec *util.ExecInPod) *ClusterThread {
//	return &ClusterThread{
//		Client:    c,
//		exeClient: ec,
//		queueMap:  make(map[string]chan QueueMsg),
//	}
//}
//
//func (c *ClusterThread) AddMsg(msg *QueueMsg) {
//	clusterName := msg.Name + msg.Namespace
//	defer c.lock.Unlock()
//	c.lock.Lock()
//	if v, ok := c.queueMap[clusterName]; ok {
//		select {
//		case v <- *msg:
//			log.Infof("write msg to %s %s", msg.Name, msg.Namespace)
//		default:
//			log.Infof("write msg to %s %s block, drop msg", msg.Name, msg.Namespace)
//		}
//	} else {
//		q := make(chan QueueMsg, 1)
//		q <- *msg
//		c.queueMap[clusterName] = q
//		go c.procQueueMsg(clusterName, q)
//	}
//}
//
//func (c *ClusterThread) procQueueMsg(key string, queue chan QueueMsg) {
//	log.Infof("thread %s start", key)
//	i := 1
//	for {
//		select {
//		case msg := <-queue:
//			c.updateCluster(&msg)
//		default:
//			c.lock.Lock()
//			if len(queue) == 0 {
//				close(queue)
//				delete(c.queueMap, key)
//			}
//			c.lock.Unlock()
//			i = 0
//		}
//		if i == 0 {
//			log.Infof("destroy msg queue %s", key)
//			break
//		}
//	}
//	log.Infof("thread %s end", key)
//}

///*
//以4个节点为例
//
//初始化：
//4标准。---返回数组中第一个节点作为主
//
//正常：
//1主 3备。 ---无动作
//
//新增：
//1主 3备 2标准。 ---标准变备
//
//异常：
//1废主 3备；2废主 2备；3废主 1备。 ---等待切主
//1废主 1主 2备。 ---已经切主，废主变备
//4废主。 ---不知所措
//*/
//func findRealGbase8sPrimary(nodes *[]NodeInfo) (*NodeInfo, error) {
//	var primaryNum, secondaryNum, standardNum int
//	for _, v := range *nodes {
//		if v.ServerType == GBASE8S_ROLE_PRIMARY {
//			primaryNum++
//			if v.Connected {
//				return &v, nil
//			}
//		} else if v.ServerType == GBASE8S_ROLE_RSS {
//			secondaryNum++
//		} else {
//			standardNum++
//		}
//	}
//	if standardNum == len(*nodes) {
//		return &((*nodes)[0]), nil
//	}
//	if secondaryNum != 0 {
//		return nil, errors.New("wait")
//	} else if primaryNum == len(*nodes) {
//		return nil, errors.New("all nodes damaged")
//	} else {
//		return nil, errors.New("internal error")
//	}
//}
//
//func isGbase8sClusterNormal(nodes *[]NodeInfo) bool {
//	for _, v := range *nodes {
//		if !v.Connected {
//			return false
//		}
//	}
//	return true
//}
//
//func (c *ClusterThread) getNodesRssStatus(pods *corev1.PodList) (*[]NodeInfo, error) {
//	var nodes []NodeInfo
//	onstatCmd := []string{"bash", "-c", "source /env.sh && onstat -g rss verbose"}
//	//needWait := false
//	for _, v := range pods.Items {
//		stdout, stderr, err := c.exeClient.Exec(onstatCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
//		if err != nil {
//			if !strings.Contains(stderr, "shared memory not initialized") {
//				return nil, errors.New("get rss status failed, error: " + err.Error() + " " + stderr)
//			}
//		}
//
//		nodeInfo, _ := ParseNodeInfo(v.Name, v.Spec.Hostname, "", stdout)
//		nodes = append(nodes, *nodeInfo)
//	}
//
//	return &nodes, nil
//}

//func (c *ClusterThread) updateGbase8sCluster(clusterName, namespaceName string) error {
//	for {
//		time.Sleep(time.Second * 3)
//		log.Infof("update gbase8s cluster %s %s", clusterName, namespaceName)
//
//		//获取期望pod个数
//		ctx := context.Background()
//		var gbase8sExpectReplicas int32
//		var gbase8sCluster gbase8sv1.Gbase8sCluster
//		reqTemp := types.NamespacedName{
//			Name:      clusterName,
//			Namespace: namespaceName,
//		}
//		if err := c.Get(ctx, reqTemp, &gbase8sCluster); err != nil {
//			return errors.New("Update cluster failed, cannot get gbase8s cluster, error: " + err.Error())
//		}
//		gbase8sExpectReplicas = gbase8sCluster.Spec.Gbase8sCfg.Replicas
//		if gbase8sExpectReplicas <= 1 {
//			return errors.New("at least 2 gbase8s nodes needed")
//		}
//
//		//获取所有gbase8s pod
//		gPodLabels := map[string]string{
//			GBASE8S_POD_LABEL_KEY: GBASE8S_POD_LABEL_VALUE_PREFIX + gbase8sCluster.Name,
//		}
//		gPods := &corev1.PodList{}
//		opts := &client.ListOptions{
//			Namespace:     namespaceName,
//			LabelSelector: labels.SelectorFromSet(gPodLabels),
//		}
//		if err := c.List(ctx, gPods, opts); err != nil {
//			return errors.New("get gbase8s pods failed, err: " + err.Error())
//		}
//
//		if gbase8sExpectReplicas != int32(len(gPods.Items)) {
//			continue
//		}
//
//		//如果有pod没在运行状态，等待
//		flag := 1
//		for _, v := range gPods.Items {
//			if len(v.Status.ContainerStatuses) != 0 {
//				if v.Status.ContainerStatuses[0].State.Running == nil {
//					flag = 0
//					break
//				}
//			}
//		}
//		if flag == 0 {
//			continue
//		}
//
//		//获取所有容器的rss状态
//		nodes, err := c.getNodesRssStatus(gPods)
//		if err != nil {
//			return err
//		}
//
//		needWait := false
//		for _, v := range *nodes {
//			if v.ServerStatus == GBASE8S_STATUS_NONE ||
//				v.ServerStatus == GBASE8S_STATUS_INIT ||
//				v.ServerStatus == GBASE8S_STATUS_FAST_RECOVERY {
//				needWait = true
//				break
//			}
//		}
//		if needWait {
//			continue
//		}
//
//		if isGbase8sClusterNormal(nodes) {
//			return errors.New("success")
//		}
//
//		p, err := findRealGbase8sPrimary(nodes)
//		if err != nil {
//			if err.Error() == "wait" {
//				continue
//			} else {
//				return errors.New("find real gbase8s primary failed, err: " + err.Error())
//			}
//		}
//
//		_, domain, err := GetHostTemplate(gPods)
//		if err != nil {
//			return err
//		}
//
//		//向主节点添加辅节点
//		var addNodes strings.Builder
//		for _, v := range *nodes {
//			if v.PodName != p.PodName && !v.Connected {
//				bfind := false
//				for _, vs := range p.SubStatus {
//					if vs.ServerName == v.ServerName {
//						bfind = true
//						break
//					}
//				}
//				if !bfind {
//					addNodes.WriteString("&& onmode -d add RSS ")
//					addNodes.WriteString(v.ServerName)
//				}
//			}
//		}
//		if addNodes.Len() != 0 {
//			addCmd := []string{"bash", "-c", "source /env.sh" + addNodes.String()}
//			log.Infof("primary: %s, add cmd: %s", p.PodName, addNodes.String())
//			_, stderr, err := c.exeClient.Exec(addCmd, GBASE8S_CONTAINER_NAME, p.PodName, namespaceName, nil)
//			if err != nil {
//				return errors.New(fmt.Sprintf("add rss failed, exec pod: %s, err: %s, %s", p.PodName, err.Error(), stderr))
//			}
//		}
//
//		//辅节点加入集群
//		for _, v := range *nodes {
//			if v.PodName != p.PodName && !v.Connected {
//				rssCmd := []string{"bash", "-c", "source /env.sh && curl -o tape http://" +
//					p.PodName +
//					"." +
//					domain +
//					":8000/hac/getTape && sh /recover.sh tape && onmode -d RSS " +
//					p.ServerName}
//				log.Infof("secondary: %s, add to cluster", v.PodName)
//				_, stderr, err := c.exeClient.Exec(rssCmd, GBASE8S_CONTAINER_NAME, v.PodName, namespaceName, nil)
//				if err != nil {
//					return errors.New(fmt.Sprintf("exec rss failed, exec pod: %s, err: %s, %s", v.PodName, err.Error(), stderr))
//				}
//			}
//		}
//
//		return nil
//	}
//}
//
//func (c *ClusterThread) updateCmCluster(clusterName, namespaceName string, needReloadCm bool) error {
//	for {
//		time.Sleep(time.Second * 3)
//		log.Infof("update cm cluster %s %s", clusterName, namespaceName)
//
//		ctx := context.Background()
//		var cmExpectReplicas int32
//		var gbase8sCluster gbase8sv1.Gbase8sCluster
//		reqTemp := types.NamespacedName{
//			Name:      clusterName,
//			Namespace: namespaceName,
//		}
//		if err := c.Get(ctx, reqTemp, &gbase8sCluster); err != nil {
//			return errors.New("Update cluster failed, cannot get gbase8s cluster, error: %s" + err.Error())
//		}
//
//		cmExpectReplicas = gbase8sCluster.Spec.CmCfg.Replicas
//		if cmExpectReplicas < 1 {
//			return errors.New("at least 1 cm nodes needed")
//		}
//
//		//获取cm pods
//		cmPodLabels := map[string]string{
//			CM_POD_LABEL_KEY: CM_POD_LABEL_VALUE_PREFIX + gbase8sCluster.Name,
//		}
//		cmPods := &corev1.PodList{}
//		opts := &client.ListOptions{
//			Namespace:     namespaceName,
//			LabelSelector: labels.SelectorFromSet(cmPodLabels),
//		}
//		if err := c.List(ctx, cmPods, opts); err != nil {
//			return errors.New("get cm pods failed, err: " + err.Error())
//		}
//
//		if cmExpectReplicas != int32(len(cmPods.Items)) {
//			continue
//		}
//
//		//如果有pod没在运行状态，等待
//		flag := 1
//		for _, v := range cmPods.Items {
//			if len(v.Status.ContainerStatuses) != 0 {
//				if v.Status.ContainerStatuses[0].State.Running == nil {
//					flag = 0
//					break
//				}
//			}
//		}
//		if flag == 0 {
//			continue
//		}
//
//		//启动cm或重新加载cm配置
//		statCmd := []string{"bash", "-c", "ps -ef|grep oncmsm|grep -v grep|wc -l"}
//		startCmd := []string{"bash", "-c", "source /env.sh && sh start_manual.sh"}
//		reloadCmd := []string{"bash", "-c", "source /env.sh && oncmsm -r -c /opt/gbase8s/etc/cfg.cm"}
//		for _, v := range cmPods.Items {
//			stdout, stderr, err := c.exeClient.Exec(statCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
//			if err != nil {
//				return errors.New(fmt.Sprintf("get cm status failed, err: %s %s", err.Error(), stderr))
//			}
//			if stdout == "1\n" {
//				if needReloadCm {
//					log.Infof("reload cm %s config", v.Name)
//					stdout, stderr, err := c.exeClient.Exec(reloadCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
//					if err != nil {
//						return errors.New(fmt.Sprintf("reload cm failed, err: %s %s %s", err.Error(), stderr, stdout))
//					}
//				}
//			} else {
//				log.Infof("start cm %s", v.Name)
//				_, stderr, err := c.exeClient.Exec(startCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
//				if err != nil {
//					return errors.New(fmt.Sprintf("start cm failed, err: %s %s", err.Error(), stderr))
//				}
//			}
//		}
//
//		return nil
//	}
//}
//
//func (c *ClusterThread) updateCluster(msg *QueueMsg) {
//
//	needReloadCm := false
//	if err := c.updateGbase8sCluster(msg.Name, msg.Namespace); err != nil {
//		if err.Error() != "success" {
//			log.Errorf("update gbase8s cluster failed, err: %s", err.Error())
//			return
//		}
//	} else {
//		needReloadCm = true
//	}
//
//	needReloadCm = false
//	if err := c.updateCmCluster(msg.Name, msg.Namespace, needReloadCm); err != nil {
//		log.Errorf("update cm cluster failed, err: %s", err.Error())
//	}
//
//	return
//}
