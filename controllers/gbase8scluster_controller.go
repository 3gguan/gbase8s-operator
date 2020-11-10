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
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"

	gbase8sv1 "Gbase8sCluster/api/v1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"Gbase8sCluster/util"
)

// Gbase8sClusterReconciler reconciles a Gbase8sCluster object
type Gbase8sClusterReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	ExecInPod *util.ExecInPod
}

// +kubebuilder:rbac:groups=gbase8s.gbase.cn,resources=gbase8sclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gbase8s.gbase.cn,resources=gbase8sclusters/status,verbs=get;update;patch

func (r *Gbase8sClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("gbase8scluster", req.NamespacedName)

	cmd := []string{"bash", "-c", "source env.sh && onstat -g rss"}

	retOut, retErr, err := r.ExecInPod.Exec(cmd, "gbase8s", "gbase8s-cluster-0", "default", nil)
	if err != nil {
		log.Error(err, "failed to exec command")

	} else {
		log.Info("retOut: " + retOut)
		log.Info("retErr: " + retErr)
	}

	// your logic here
	var gbase8sCluster gbase8sv1.Gbase8sCluster
	if err := r.Get(ctx, req.NamespacedName, &gbase8sCluster); err != nil {
		log.Error(err, "====unable to get gbase8s cluster====")
	} else {
		fmt.Println("===Get gbase8s cluster spec info success, ", gbase8sCluster.Spec.Gbase8sCfg.Replicas, gbase8sCluster.Spec.CmCfg.Replicas)
	}

	var pods corev1.Pod
	req.NamespacedName.Name = "gbase8s-cluster-0"
	if err := r.Get(ctx, req.NamespacedName, &pods); err != nil {
		log.Error(err, "====unable to get gbase8s cluster====")
	} else {
		fmt.Println("===Get gbase8s cluster spec info success, ", gbase8sCluster.Spec.Gbase8sCfg.Replicas, gbase8sCluster.Spec.CmCfg.Replicas)
	}

	gbase8sCluster.Status.Status = "Running"
	if err := r.Status().Update(ctx, &gbase8sCluster); err != nil {
		log.Error(err, "====unable to update gbase8s cluster status====")
	}

	return ctrl.Result{}, nil
}

func (r *Gbase8sClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gbase8sv1.Gbase8sCluster{}).
		Complete(r)
}
