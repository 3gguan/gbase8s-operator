package controllers

import (
	"Gbase8sCluster/util"
	"context"
	"errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type Pod struct {
	client.Client
	execClient *util.ExecInPod
}

var pod *Pod

func InitPod(client client.Client, execClient *util.ExecInPod) {
	pod = &Pod{
		Client:     client,
		execClient: execClient,
	}
}

func GetAllPods(namespace string, podLabels *map[string]string) (*corev1.PodList, error) {
	ctx := context.Background()
	pods := &corev1.PodList{}
	opts := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(*podLabels),
	}
	err := pod.List(ctx, pods, opts)

	return pods, err
}

func GetHostTemplate(pods *corev1.PodList) (string, string, error) {
	getHostCmd := []string{"bash", "-c", "hostname && dnsdomainname"}

	//获取hostname和dnsdomainname
	hostnameStr := ""
	errStr := ""
	for _, v := range pods.Items {
		if len(v.Status.ContainerStatuses) != 0 {
			if v.Status.ContainerStatuses[0].State.Running != nil {
				stdout, _, err := pod.execClient.Exec(getHostCmd, v.Spec.Containers[0].Name, v.Name, v.Namespace, nil)
				if err != nil {
					errStr = err.Error()
				}
				if stdout != "" {
					hostnameStr = stdout
					break
				}
			}
		}
	}

	if hostnameStr == "" {
		if errStr != "" {
			return "", "", errors.New("get host template failed, err: " + errStr)
		}
		return "", "", errors.New("get host template failed")
	}

	hostname := strings.Split(hostnameStr, "\n")
	//var hostnameTemplate string
	for i := len(hostname[0]); i > 0; i-- {
		if hostname[0][i-1] == '-' {
			hostname[0] = hostname[0][0 : i-1]
			break
		}
	}

	return hostname[0], hostname[1], nil
}
