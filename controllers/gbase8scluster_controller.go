/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	gbase8sv1 "Gbase8sCluster/api/v1"
	"context"
	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	_ "k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"Gbase8sCluster/util"
)

var log = logrus.New()

// Gbase8sClusterReconciler reconciles a Gbase8sCluster object
type Gbase8sClusterReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	ExecInPod *util.ExecInPod
	Event     record.EventRecorder
}

// +kubebuilder:rbac:groups=gbase8s.gbase.cn,resources=gbase8sclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gbase8s.gbase.cn,resources=gbase8sclusters/status,verbs=get;update;patch

func (r *Gbase8sClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	//log := r.Log.WithValues("gbase8scluster", req.NamespacedName)

	log.Infof("------ %s %s ------", req.Name, req.Namespace)

	//cmd := []string{"bash", "-c", "source env.sh && onstat -g rss"}
	//
	//retOut, retErr, err := r.ExecInPod.Exec(cmd, "gbase8s", "gbase8s-cluster-0", "default", nil)
	//if err != nil {
	//	log.Error(err, "failed to exec command")
	//
	//} else {
	//	log.Info("retOut: " + retOut)
	//	log.Info("retErr: " + retErr)
	//}

	// your logic here
	//获取Gbase8sCluster资源
	var gbase8sExpectReplicas int32
	var gbase8sCluster gbase8sv1.Gbase8sCluster
	if err := r.Get(ctx, req.NamespacedName, &gbase8sCluster); err != nil {
		log.Errorf("Unable to get gbase8s cluster resource, error: %s", err.Error())
		return ctrl.Result{}, err
	} else {
		log.Infof("Get gbase8s cluster resource success, gbase8s replicas: %d, cm replicas: %d", gbase8sCluster.Spec.Gbase8sCfg.Replicas, gbase8sCluster.Spec.CmCfg.Replicas)
		gbase8sExpectReplicas = gbase8sCluster.Spec.Gbase8sCfg.Replicas
	}

	//获取gbase8s statefulset资源
	var gbase8sReplicas int32
	var statefulset appsv1.StatefulSet
	reqTemp := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "gbase8s-cluster",
			Namespace: req.Namespace,
		},
	}
	if err := r.Get(ctx, reqTemp.NamespacedName, &statefulset); err != nil {
		log.Infof("Unable to get gbase8s statefulset resource, error: %s", err.Error())
		gbase8sReplicas = 0
	} else {
		log.Infof("Get gbase8s statefulset resource success, gbase8s replicas: %d", statefulset.Spec.Replicas)
		gbase8sReplicas = *statefulset.Spec.Replicas
	}

	//获取service资源
	var service corev1.Service
	reqSvc := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      GBASE8S_SERVICE_DEFAULT_NAME,
			Namespace: req.Namespace,
		},
	}
	if err := r.Get(ctx, reqSvc.NamespacedName, &service); err != nil {
		log.Infof("Unable to get gbase8s service resource, error: %s", err.Error())

		//创建service
		gsvc := NewGbase8sService()
		if err := r.Create(ctx, gsvc.svc); err != nil {
			log.Errorf("Create gbase8s service failed, err: %s", err.Error())
			return ctrl.Result{}, err
		}
	} else {
		log.Infof("Get gbase8s service resource success")
	}

	if gbase8sReplicas == 0 {
		//创建statefulset
		gsfs := NewGbase8sStatefulset(&gbase8sCluster)
		if err := r.Create(ctx, gsfs.sfs); err != nil {
			log.Errorf("Create gbase8s statefulset failed, err: %s", err.Error())
			return ctrl.Result{}, err
		}
	} else if gbase8sReplicas != gbase8sExpectReplicas {
		//更新statefulset
		gsfs := NewGbase8sStatefulset(&gbase8sCluster)
		gsfs.sfs.Spec.Replicas = &gbase8sExpectReplicas
		if err := r.Update(ctx, gsfs.sfs); err != nil {
			log.Errorf("Update gbase8s statefulset failed, err: %s", err.Error())
			return ctrl.Result{}, err
		} else {
			log.Info("Update gbase8s statefulset success")
		}
	}

	//var pods corev1.Pod
	//req.NamespacedName.Name = "gbase8s-cluster-0"
	//if err := r.Get(ctx, req.NamespacedName, &pods); err != nil {
	//	log.Error(err, "====unable to get gbase8s-cluster-0====")
	//} else {
	//	fmt.Println("===Get pod info success, ", pods.Spec.Hostname, pods.Status.Phase)
	//}

	//gbase8sCluster.Status.Status = "Running"
	//if err := r.Status().Update(ctx, &gbase8sCluster); err != nil {
	//	log.Error(err, "====unable to update gbase8s cluster status====")
	//}

	return ctrl.Result{}, nil
}

func (r *Gbase8sClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gbase8sv1.Gbase8sCluster{}).
		//For(&corev1.Pod{}).
		//Watches(&source.Kind{Type: &gbase8sv1.Gbase8sCluster{}}, &handler.EnqueueRequestForObject{}).
		//Owns(&corev1.Pod{}).
		Owns(&appsv1.StatefulSet{}).
		WithEventFilter(&PodStatusChangedPredicate{}).
		Complete(r)
}
