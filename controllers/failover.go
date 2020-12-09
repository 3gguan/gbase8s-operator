package controllers

//
//import (
//	"Gbase8sCluster/util"
//	"fmt"
//	"strings"
//	"sync"
//	"time"
//)
//
//type detecting struct {
//	//探测次数
//	detectingCount int
//
//	//探测时间间隔
//	detectingInterval int
//
//	//探测超时时间
//	timeout int
//}
//
//type CfgQueueMsg struct {
//	MsgType int
//	Data interface{}
//}
//
//type clusterManager struct {
//	clusterName string
//	clusterNamespace string
//
//	detecting *detecting
//
//	//当前探测失败次数
//	currFailedCount int
//
//	//探测主节点
//	primaryPod *NodeBasicInfo
//
//	//所有节点
//	podList *[]*NodeBasicInfo
//
//	//http client
//	httpClient *util.HttpClient
//
//	//定时器
//	detectingTicker *time.Ticker
//
//	//消息队列
//	queue chan *CfgQueueMsg
//
//	//停止信号
//	stopDetectingChannel chan int
//}
//
//type ClusterManager struct {
//	clusterMap  map[string]*clusterManager
//	rwLock      sync.RWMutex
//}
//
//var failover *ClusterManager
//
//func NewFailover() *ClusterManager {
//	return &ClusterManager{
//		clusterMap: make(map[string]*clusterManager),
//	}
//}
//
//func (f *ClusterManager) procModifyParam(param *clusterManager, queueMsg *CfgQueueMsg) {
//	switch queueMsg.MsgType {
//	case FAILOVER_MSGTYPE_PRIMARYPOD:
//		if v, ok := queueMsg.Data.(*NodeBasicInfo); ok {
//			param.primaryPod = v
//		} else {
//			log.Error("Modify primary pod failed")
//		}
//	case FAILOVER_MSGTYPE_DETECTING:
//		if v, ok := queueMsg.Data.(*detecting); ok {
//			if param.detecting.detectingCount != v.detectingCount {
//				param.detecting.detectingCount = v.detectingCount
//				param.currFailedCount = 0
//			}
//			if param.detecting.detectingInterval != v.detectingInterval {
//				param.detecting.detectingInterval = v.detectingInterval
//				param.currFailedCount = 0
//				param.detectingTicker.Reset(time.Duration(v.detectingInterval) * time.Second)
//			}
//			if param.detecting.timeout != v.timeout {
//				param.detecting.timeout = v.timeout
//				param.httpClient.SetTimeout(time.Duration(v.timeout))
//			}
//		} else {
//			log.Error("Modify detecting param failed")
//		}
//	case FAILOVER_MSGTYPE_PODLIST:
//		if v, ok := queueMsg.Data.(*[]*NodeBasicInfo); ok {
//			param.podList = v
//		} else {
//			log.Error("Modify pod list failed")
//		}
//	default:
//		log.Errorf("Modify param failed, msg type error: %d", queueMsg.MsgType)
//	}
//}
//
//func (f *ClusterManager) procFailover(param *clusterManager) {
//	//发送探测请求
//	primaryPod := param.primaryPod
//	if primaryPod != nil {
//		url := fmt.Sprintf("http://%s:%d/hac/getStatus", primaryPod.HostName, GBASE8S_CONFIG_PORT)
//		if resp, err := param.httpClient.Get(url); err != nil {
//			if strings.Contains(err.Error(), "timeout") ||
//				strings.Contains(err.Error(), "Timeout") {
//				param.currFailedCount++
//			}
//		} else {
//			if resp.Code != "0" {
//				param.currFailedCount++
//			}
//		}
//
//		//进行故障转移
//		if param.currFailedCount >= param.detecting.detectingCount {
//
//			//获取secondary节点信息
//			var nodeList []*NodeInfo
//			for _, v := range *param.podList {
//				if v.PodName != primaryPod.PodName {
//					url := fmt.Sprintf("http://%s:%d/hac/getStatus", v.HostName, GBASE8S_CONFIG_PORT)
//					if resp, err := param.httpClient.Get(url); err == nil {
//						if resp.Code == "0" {
//							if val, ok := resp.Data.(string); ok {
//								nodeInfo, _ := ParseNodeInfo(v.PodName, v.HostName, v.Domain, val)
//								nodeList = append(nodeList, nodeInfo)
//							}
//						}
//					}
//				}
//			}
//
//			//找一个secondary节点，切成主
//			var secondaryNode *NodeInfo
//			for _, v := range nodeList {
//				if v.SourceServerName == primaryPod.ServerName {
//					secondaryNode = v
//					break
//				}
//			}
//
//			//切主
//			if secondaryNode != nil {
//				url := fmt.Sprintf("http://%s.%s:%d/hac/exec", secondaryNode.HostName, secondaryNode.Domain, GBASE8S_CONFIG_PORT)
//				cmdBuilder := strings.Builder{}
//				for _, v := range *param.podList {
//					if v.PodName != secondaryNode.PodName {
//						cmdBuilder.WriteString(" && onmode -d add RSS ")
//						cmdBuilder.WriteString(primaryPod.ServerName)
//					}
//				}
//
//				if cmdBuilder.Len() != 0 {
//					body := map[string]interface{}{
//						"cmd": "source /env.sh && onmode -d standard" + cmdBuilder.String(),
//					}
//					resp, err := param.httpClient.Post(url, body)
//					if err != nil {
//						log.Errorf("failover failed, err: %s", err)
//					} else {
//						if resp.Code != "0" {
//							log.Errorf("failover failed, code: %s, message: %s", resp.Code, resp.Message)
//						} else {
//							//通知重建集群
//
//
//							param.primaryPod = nil
//						}
//					}
//				}
//
//
//			} else {
//				log.Errorf("cannot find failover node, cluster: %s, namespace: %s", param.clusterName, param.clusterNamespace)
//			}
//		}
//	}
//
//}
//
//func (f *ClusterManager) failoverThread(param *clusterManager) {
//	log.Infof("failover thread %s %s start", param.clusterName, param.clusterNamespace)
//	stop := false
//	for {
//		select {
//		case queueMsg := <-param.queue:
//			f.procModifyParam(param, queueMsg)
//		case <-param.detectingTicker.C:
//			f.procFailover(param)
//		case <-param.stopDetectingChannel:
//			stop = true
//			break
//		}
//
//		if stop {
//			break
//		}
//	}
//	log.Infof("failover thread %s %s end", param.clusterName, param.clusterNamespace)
//}
//
//func (f *ClusterManager) modifyParam(param *clusterManager, msgType int, data interface{}) {
//	msg := CfgQueueMsg{
//		MsgType: msgType,
//		Data: data,
//	}
//	param.queue <- &msg
//}
//
//func (f *ClusterManager) AddCluster(clusterName, clusterNamespace string, podList *[]*NodeBasicInfo, detectingCount, detectingInterval, timeout int) {
//	name := clusterName + "|" + clusterNamespace
//	f.rwLock.Lock()
//	if v, ok := f.clusterMap[name]; ok {
//		f.modifyParam(v, FAILOVER_MSGTYPE_PODLIST, podList)
//		detecting := &detecting{
//			detectingCount: detectingCount,
//			detectingInterval: detectingInterval,
//			timeout: timeout,
//		}
//		f.modifyParam(v, FAILOVER_MSGTYPE_DETECTING, detecting)
//		f.rwLock.Unlock()
//	} else {
//		clusterManager := clusterManager{
//			clusterName: clusterName,
//			clusterNamespace: clusterNamespace,
//			detecting: &detecting{
//				detectingCount: detectingCount,
//				detectingInterval: detectingInterval,
//				timeout: timeout,
//			},
//			currFailedCount: 0,
//			podList: podList,
//			httpClient: util.NewHttpClient().SetTimeout(time.Duration(timeout)),
//			detectingTicker: time.NewTicker(time.Duration(detectingInterval) * time.Second),
//			queue: make(chan *CfgQueueMsg),
//		}
//		f.clusterMap[name] = &clusterManager
//		f.rwLock.Unlock()
//		go f.failoverThread(&clusterManager)
//	}
//}
//
//func (f *ClusterManager) DelCluster(clusterName, clusterNamespace string) {
//	name := clusterName + "|" + clusterNamespace
//	f.rwLock.Lock()
//	if v, ok := f.clusterMap[name]; ok {
//		v.stopDetectingChannel <- 1
//		delete(f.clusterMap, name)
//	}
//	f.rwLock.Unlock()
//}
