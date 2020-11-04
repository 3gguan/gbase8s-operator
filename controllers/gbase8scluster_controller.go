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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gbase8sv1 "Gbase8sCluster/api/v1"
)

// Gbase8sClusterReconciler reconciles a Gbase8sCluster object
type Gbase8sClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=gbase8s.gbase.cn,resources=gbase8sclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gbase8s.gbase.cn,resources=gbase8sclusters/status,verbs=get;update;patch

func (r *Gbase8sClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("gbase8scluster", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

func (r *Gbase8sClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gbase8sv1.Gbase8sCluster{}).
		Complete(r)
}
