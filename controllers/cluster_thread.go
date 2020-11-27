package controllers

import (
	gbase8sv1 "Gbase8sCluster/api/v1"
	"Gbase8sCluster/util"
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"sync"
	"time"
)

type ContainerRole struct {
	Name         string
	Status       string
	Role         string
	HasSecondary bool
}

type QueueMsg struct {
	Name      string
	Namespace string
}

type ClusterThread struct {
	client.Client
	exeClient *util.ExecInPod
	queueMap  map[string]chan QueueMsg
	lock      sync.Mutex
}

func NewClusterThread(c client.Client, ec *util.ExecInPod) *ClusterThread {
	return &ClusterThread{
		Client:    c,
		exeClient: ec,
		queueMap:  make(map[string]chan QueueMsg),
	}
}

func (c *ClusterThread) AddMsg(msg *QueueMsg) {
	clusterName := msg.Name + msg.Namespace
	defer c.lock.Unlock()
	c.lock.Lock()
	if v, ok := c.queueMap[clusterName]; ok {
		select {
		case v <- *msg:
			log.Infof("write msg to %s %s", msg.Name, msg.Namespace)
		default:
			log.Infof("write msg to %s %s block, drop msg", msg.Name, msg.Namespace)
		}
	} else {
		q := make(chan QueueMsg, 1)
		q <- *msg
		c.queueMap[clusterName] = q
		go c.procQueueMsg(clusterName, q)
	}
}

func (c *ClusterThread) procQueueMsg(key string, queue chan QueueMsg) {
	i := 1
	for {
		select {
		case msg := <-queue:
			c.updateCluster(&msg)
		default:
			c.lock.Lock()
			if len(queue) == 0 {
				close(queue)
				delete(c.queueMap, key)
			}
			c.lock.Unlock()
			i = 0
		}
		if i == 0 {
			log.Infof("destroy msg queue %s", key)
			break
		}
	}
}

func (c *ClusterThread) GetHostTemplate(pods *corev1.PodList) *[]string {
	getHostCmd := []string{"bash", "-c", "hostname && dnsdomainname"}

	//获取hostname和dnsdomainname
	hostnameStr := ""
	for _, v := range pods.Items {
		if len(v.Status.ContainerStatuses) != 0 {
			if v.Status.ContainerStatuses[0].State.Running != nil {
				stdout, _, err := c.exeClient.Exec(getHostCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
				if err != nil {
					log.Errorf("get hostname failed, error: %s", err.Error())
				} else {
					if stdout != "" {
						hostnameStr = stdout
						break
					}
				}
			}
		}
	}

	if hostnameStr == "" {
		return nil
	}

	hostname := strings.Split(hostnameStr, "\n")
	//var hostnameTemplate string
	for i := len(hostname[0]); i > 0; i-- {
		if hostname[0][i-1] == '-' {
			hostname[0] = hostname[0][0 : i-1]
			break
		}
	}

	return &hostname
}

func (c *ClusterThread) updateCluster(msg *QueueMsg) {
	for {
		time.Sleep(time.Second * 3)

		//获取期望pod个数
		ctx := context.Background()
		fmt.Println("===========")
		var gbase8sExpectReplicas int32
		var gbase8sCluster gbase8sv1.Gbase8sCluster
		reqTemp := ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      msg.Name,
				Namespace: msg.Namespace,
			},
		}
		if err := c.Get(ctx, reqTemp.NamespacedName, &gbase8sCluster); err != nil {
			log.Errorf("Update cluster failed, cannot get gbase8s cluster, error: %s", err.Error())
			return
		}
		gbase8sExpectReplicas = gbase8sCluster.Spec.Gbase8sCfg.Replicas
		if gbase8sExpectReplicas <= 1 {
			return
		}

		//获取所有pod
		statefulset := &appsv1.StatefulSet{}
		reqTemp = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "gbase8s-cluster",
				Namespace: msg.Namespace,
			},
		}
		if err := c.Get(ctx, reqTemp.NamespacedName, statefulset); err != nil {
			log.Errorf("Update cluster failed, cannot get gbase8s statefulset, error: %s", err.Error())
			return
		}

		pods := &corev1.PodList{}
		opts := &client.ListOptions{
			Namespace:     msg.Namespace,
			LabelSelector: labels.SelectorFromSet(statefulset.Spec.Template.Labels),
		}
		if err := c.List(ctx, pods, opts); err != nil {
			continue
		}

		if gbase8sExpectReplicas != int32(len(pods.Items)) {
			continue
		}

		//如果有pod没在运行状态，等待
		flag := 1
		for _, v := range pods.Items {
			if len(v.Status.ContainerStatuses) != 0 {
				if v.Status.ContainerStatuses[0].State.Running == nil {
					flag = 0
					break
				}
			}
		}
		if flag == 0 {
			continue
		}

		//获取所有容器的rss状态
		var roles []ContainerRole
		onstatCmd := []string{"bash", "-c", "source /env.sh && onstat -g rss"}
		for _, v := range pods.Items {
			tempRole := ContainerRole{
				Name:         v.Name,
				HasSecondary: false,
			}
			stdout, _, err := c.exeClient.Exec(onstatCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
			if err != nil {
				log.Errorf("get rss status failed, error: %s", err.Error())
			}

			//log.Info(stdout)
			list1 := strings.Split(stdout, "\n")
			for _, v := range list1 {
				if strings.Contains(v, "GBase Database Server Version") {
					list2 := strings.Split(v, "--")
					if len(list2) == 4 {
						tempRole.Status = strings.TrimSpace(list2[1])
					}
				} else if strings.Contains(v, "Local server type") {
					list2 := strings.Split(v, ":")
					if len(list2) == 2 {
						tempRole.Role = strings.TrimSpace(list2[1])
					}
				} else if strings.Contains(v, "Connected") {
					tempRole.HasSecondary = true
				}
			}

			roles = append(roles, tempRole)
		}
		//fmt.Println(roles)
		//有没启动完成的等待下次
		flag = 1
		for _, v := range roles {
			if v.Status != GBASE8S_STATUS_ONLINE {
				flag = 0
				break
			}
		}
		if flag == 0 {
			continue
		}

		hostTemplate := c.GetHostTemplate(pods)

		//标准节点个数
		standardCount := 0
		//主节点个数
		primaryCount := 0
		//辅节点个数
		secondaryCount := 0
		//真正主节点个数
		hasSendaryCount := 0
		for _, v := range roles {
			if v.Role == GBASE8S_ROLE_STANDARD {
				standardCount++
			} else if v.Role == GBASE8S_ROLE_PRIMARY {
				primaryCount++
			} else if v.Role == GBASE8S_ROLE_RSS {
				secondaryCount++
			}
			if v.HasSecondary == true {
				hasSendaryCount++
			}
		}

		if int32(standardCount+primaryCount+secondaryCount) < gbase8sExpectReplicas {
			continue
		}

		if int32(standardCount) == gbase8sExpectReplicas {
			//集群没建立过
			log.Info("gbase8s cluster init")
			addCmd := ""
			for i, v := range pods.Items {
				if i == 0 {
					continue
				}
				addCmd += " && onmode -d add RSS " + strings.Replace(v.Name, "-", "_", -1)
			}
			addRssCmd := []string{"bash", "-c", "source /env.sh" + addCmd}
			_, _, err := c.exeClient.Exec(addRssCmd, pods.Items[0].Spec.Containers[0].Name, pods.Items[0].Name, pods.Items[0].Namespace, nil)
			if err != nil {
				log.Errorf("add rss failed, exec pod: %s, err: %s", pods.Items[0].Name, err.Error())
				return
			}

			//rssCmd := []string{"curl", "-o", "http://" + pods.Items[0].Name + "." + (*msg.hostTemplate)[1] + ":8000/hac/getTape"}
			//recoverCmd := []string{"sh", "/recover.sh", ""}
			rssCmd := []string{"bash", "-c", "source /env.sh && curl -o tape http://" +
				pods.Items[0].Name +
				"." +
				(*hostTemplate)[1] +
				":8000/hac/getTape && sh /recover.sh tape && onmode -d RSS " +
				strings.Replace(pods.Items[0].Name, "-", "_", -1)}
			for i, v := range pods.Items {
				if i != 0 {
					_, _, err := c.exeClient.Exec(rssCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
					if err != nil {
						log.Errorf("exec rss failed, exec pod: %s, err: %s", v.Name, err.Error())
						return
					}
				}
			}
		} else if (hasSendaryCount == 1) && (int32(secondaryCount) == (gbase8sExpectReplicas - 1)) {
			//集群正常（controller重启可能会是这种情况）
			log.Info("gbase8s cluster normal")
		} else {
			//集群坏掉了，重建集群
			if hasSendaryCount == 0 {
				//主节点down，还没切换
				continue
			}
			//else if (primaryCount == 1) && (int32(secondaryCount) < (gbase8sExpectReplicas - 1)) {
			//	//主节点down，已经切换，需要重建集群
			//} else if primaryCount
		}

		log.Info("update cluster success")
		break
	}
}
