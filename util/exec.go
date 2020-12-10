package util

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
	"time"

	// "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

var log = logrus.New()

type ExecInPod struct {
	Scheme    *runtime.Scheme
	Clientset *kubernetes.Clientset
	Config    *rest.Config
}

func (r *ExecInPod) GetClientConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Info("can not get in cluster client config. " + err.Error())

		kubeConfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			log.Error("can not get client config. " + err.Error())
			return nil, err
		}
	}

	return config, nil
}

func (r *ExecInPod) GetClientsetFromConfig(config *rest.Config) (*kubernetes.Clientset, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error("can not get clientset from config. " + err.Error())
		return nil, err
	}

	return clientset, nil
}

func (r *ExecInPod) GetClientset() (*kubernetes.Clientset, error) {
	config, err := r.GetClientConfig()
	if err != nil {
		return nil, err
	}

	return r.GetClientsetFromConfig(config)
}

func (r *ExecInPod) GetRESTClient() (*rest.RESTClient, error) {
	config, err := r.GetClientConfig()
	if err != nil {
		return &rest.RESTClient{}, err
	}

	return rest.RESTClientFor(config)
}

func NewExecClient() (*ExecInPod, error) {
	execInPod := ExecInPod{}

	config, err := execInPod.GetClientConfig()
	if err != nil {
		return nil, err
	}
	execInPod.Config = config

	clientset, err := execInPod.GetClientsetFromConfig(config)
	if err != nil {
		return nil, err
	}
	execInPod.Clientset = clientset

	return &execInPod, err
}

//command: []string{"bash", "-c", "source env.sh && onstat -g rss"}
func (r *ExecInPod) Exec(command []string, containerName, podName, namespace string, stdin io.Reader) (string, string, error) {
	clientset := r.Clientset
	config := r.Config

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		Timeout(time.Minute * 10).
		SubResource("exec")
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return "", "", err
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&corev1.PodExecOptions{
		Command:   command,
		Container: containerName,
		Stdin:     stdin != nil,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	//log.Info("Request URL:", req.URL().String())

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", "", err
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", "", err
	}

	return stdout.String(), stderr.String(), nil
}

//func main() {
//	//var namespace, containerName, podName, command string
//	//fmt.Print("Enter namespace: ")
//	//fmt.Scanln(&namespace)
//	//fmt.Print("Enter name of the pod: ")
//	//fmt.Scanln(&podName)
//	//fmt.Print("Enter name of the container [leave empty if there is only one container]: ")
//	//fmt.Scanln(&containerName)
//	//fmt.Print("Enter the commmand to execute: ")
//	//fmt.Scanln(&command)
//
//	// For now I am assuming stdin for the command to be nil
//	//output, stderr, err := ExecToPodThroughAPI(command, containerName, podName, namespace, nil)
//
//	output, stderr, err := ExecToPodThroughAPI("sh /1.sh",
//		"gbase8s",
//		"gbase8s-cluster-0",
//		"default",
//		nil)
//	if len(stderr) != 0 {
//		fmt.Println("STDERR:", stderr)
//	}
//	if err != nil {
//		//fmt.Printf("Error occured while `exec`ing to the Pod %q, namespace %q, command %q. Error: %+v\n", podName, namespace, command, err)
//		fmt.Printf(err.Error())
//	} else {
//		fmt.Println("Output:")
//		fmt.Println(output)
//	}
//}
