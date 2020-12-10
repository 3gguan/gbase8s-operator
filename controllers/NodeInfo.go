package controllers

import "strings"

type SubStatus struct {
	ServerName string
	Connected  bool
}

type NodeBasicInfo struct {
	PodName    string
	Namespace  string
	HostName   string
	Domain     string
	ServerName string
}

type NodeInfo struct {
	NodeBasicInfo

	ServerStatus     string
	ServerType       string
	SourceServerName string
	Connected        bool
	SubStatus        []SubStatus
}

func ParseNodeInfo(podName, hostname, domain, nodeInfoStr string) (*NodeInfo, error) {
	nodeInfo := &NodeInfo{
		NodeBasicInfo: NodeBasicInfo{
			PodName:    podName,
			HostName:   hostname,
			Domain:     domain,
			ServerName: strings.Replace(podName, "-", "_", -1),
		},
		ServerStatus: GBASE8S_STATUS_NONE,
		Connected:    false,
	}
	list1 := strings.Split(nodeInfoStr, "\n")
	for _, v := range list1 {
		if strings.Contains(v, "GBase Database Server Version") {
			list2 := strings.Split(v, "--")
			if len(list2) == 4 {
				nodeInfo.ServerStatus = strings.TrimSpace(list2[1])
			}
		} else if strings.Contains(v, "Local server type") {
			list2 := strings.Split(v, ":")
			if len(list2) == 2 {
				nodeInfo.ServerType = strings.TrimSpace(list2[1])
			}
		} else if strings.Contains(v, "RSS server name") {
			list2 := strings.Split(v, ":")
			if len(list2) == 2 {
				tempName := strings.TrimSpace(list2[1])
				nodeInfo.SubStatus = append(nodeInfo.SubStatus, SubStatus{
					ServerName: tempName,
					Connected:  false,
				})
			}
		} else if strings.Contains(v, "RSS connection status") {
			list2 := strings.Split(v, ":")
			if len(list2) == 2 {
				if strings.TrimSpace(list2[1]) == "Connected" {
					nodeInfo.SubStatus[len(nodeInfo.SubStatus)-1].Connected = true
				}
			}
		} else if strings.Contains(v, "Source server name") {
			list2 := strings.Split(v, ":")
			if len(list2) == 2 {
				nodeInfo.SourceServerName = strings.TrimSpace(list2[1])
			}
		}
	}

	if strings.Contains(nodeInfoStr, "Connected") {
		nodeInfo.Connected = true
	}

	return nodeInfo, nil
}
