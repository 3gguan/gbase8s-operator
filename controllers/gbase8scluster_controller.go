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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	_ "k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
)

// Gbase8sClusterReconciler reconciles a Gbase8sCluster object
type Gbase8sClusterReconciler struct {
	client.Client
	//Log       logr.Logger
	Scheme *runtime.Scheme
	Event  record.EventRecorder
	*Gbase8sClusterBuilder
	ClusterManager *ClusterManager
}

func (r *Gbase8sClusterReconciler) createPVs(pvs []*corev1.PersistentVolume, ctx context.Context) error {
	if len(pvs) != 0 {
		for _, v := range pvs {
			reqTemp := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name: v.Name,
				},
			}
			pvTemp := corev1.PersistentVolume{}
			if err := r.Get(ctx, reqTemp.NamespacedName, &pvTemp); err != nil {
				if errors.IsNotFound(err) {
					if err := r.Create(ctx, v); err != nil {
						log.Errorf("Create gbase8s pv %s failed, err: %s", v.Name, err.Error())
						return err
					} else {
						log.Infof("Create gbase8s pv %s success", v.Name)
					}
				} else {
					log.Errorf("Get gbase8s pv %s failed, error: %s", v.Name, err.Error())
					return err
				}
			} else {
				//log.Infof("Get gbase8s pv %s success", v.Name)
			}
		}
	}

	return nil
}

func (r *Gbase8sClusterReconciler) createPVCs(pvcs []*corev1.PersistentVolumeClaim, ctx context.Context) error {
	if len(pvcs) != 0 {
		for _, v := range pvcs {
			reqTemp := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      v.Name,
					Namespace: v.Namespace,
				},
			}
			pvcTemp := corev1.PersistentVolumeClaim{}
			if err := r.Get(ctx, reqTemp.NamespacedName, &pvcTemp); err != nil {
				if errors.IsNotFound(err) {
					if err := r.Create(ctx, v); err != nil {
						log.Errorf("Create gbase8s pvc %s failed, err: %s", v.Name, err.Error())
						return err
					} else {
						log.Infof("Create gbase8s pvc %s success", v.Name)
					}
				} else {
					log.Errorf("Get gbase8s pvc %s failed, error: %s", v.Name, err.Error())
					return err
				}

			} else {
				//log.Infof("Get gbase8s pvc %s success", v.Name)
			}
			//else {
			//	if err := r.Update(ctx, v); err != nil {
			//		log.Errorf("Update gbase8s pvc %s failed, error: %s", v.Name, err.Error())
			//	} else {
			//		log.Infof("Update gbase8s pvc %s success", v.Name)
			//	}
			//}
		}
	}

	return nil
}

// +kubebuilder:rbac:groups=gbase8s.gbase.cn,resources=gbase8sclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gbase8s.gbase.cn,resources=gbase8sclusters/status,verbs=get;update;patch

func (r *Gbase8sClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	//log := r.Log.WithValues("gbase8scluster", req.NamespacedName)

	log.Infof("------ %s %s ------", req.Name, req.Namespace)

	// your logic here
	//获取Gbase8sCluster资源
	var gbase8sExpectReplicas int32
	var cmExpectReplicas int32
	var gbase8sCluster gbase8sv1.Gbase8sCluster
	if err := r.Get(ctx, req.NamespacedName, &gbase8sCluster); err != nil {
		log.Errorf("Unable to get gbase8s cluster resource, error: %s", err.Error())
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			r.ClusterManager.DelCluster(req.Name, req.Namespace)
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	} else {
		//log.Infof("Get gbase8s cluster resource success, gbase8s replicas: %d, cm replicas: %d", gbase8sCluster.Spec.Gbase8sCfg.Replicas, gbase8sCluster.Spec.CmCfg.Replicas)
		gbase8sExpectReplicas = gbase8sCluster.Spec.Gbase8sCfg.Replicas
		cmExpectReplicas = gbase8sCluster.Spec.CmCfg.Replicas
	}

	//创建pv,pvc
	nodes := gbase8sCluster.Spec.Gbase8sCfg.Nodes
	if nodes != nil && len(nodes) != 0 {
		if pv, err := NewGbase8sPV(&gbase8sCluster); err != nil {
			log.Errorf("Create gbase8s pvs failed, err: %s", err.Error())
			return ctrl.Result{}, err
		} else {
			err := r.createPVs(pv.PVs, ctx)
			if err != nil {
				return ctrl.Result{}, err
			}
			err = r.createPVCs(pv.PVCs, ctx)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	cmNodes := gbase8sCluster.Spec.CmCfg.Nodes
	if cmNodes != nil && len(cmNodes) != 0 {
		if pv, err := NewCmPV(&gbase8sCluster); err != nil {
			log.Errorf("Create cm pvs failed, err: %s", err.Error())
			return ctrl.Result{}, err
		} else {
			err := r.createPVs(pv.PVs, ctx)
			if err != nil {
				return ctrl.Result{}, err
			}
			err = r.createPVCs(pv.PVCs, ctx)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	//获取gbase8s statefulset资源
	var gbase8sReplicas int32
	gstatefulset := &appsv1.StatefulSet{}
	gsfsReq := types.NamespacedName{
		Name:      GBASE8S_STATEFULSET_NAME_PREFIX + req.Name,
		Namespace: req.Namespace,
	}
	if err := r.Get(ctx, gsfsReq, gstatefulset); err != nil {
		//log.Infof("Unable to get gbase8s statefulset resource, error: %s", err.Error())
		if errors.IsNotFound(err) {
			gbase8sReplicas = 0
		} else {
			return ctrl.Result{}, err
		}
	} else {
		//log.Infof("Get gbase8s statefulset resource success, gbase8s replicas: %d", gstatefulset.Spec.Replicas)
		gbase8sReplicas = *gstatefulset.Spec.Replicas
	}

	//获取cm statefulset资源
	var cmReplicas int32
	cmStatefulset := &appsv1.StatefulSet{}
	cmSfsReq := types.NamespacedName{
		Name:      CM_STATEFULSET_NAME_PREFIX + req.Name,
		Namespace: req.Namespace,
	}
	if err := r.Get(ctx, cmSfsReq, cmStatefulset); err != nil {
		//log.Infof("Unable to get cm statefulset resource, error: %s", err.Error())
		if errors.IsNotFound(err) {
			cmReplicas = 0
		} else {
			return ctrl.Result{}, err
		}
	} else {
		//log.Infof("Get cm statefulset resource success, cm replicas: %d", cmStatefulset.Spec.Replicas)
		cmReplicas = *cmStatefulset.Spec.Replicas
	}

	//获取并创建gbase8s service资源
	var gservice corev1.Service
	gsvcReq := types.NamespacedName{
		Name:      GBASE8S_SERVICE_NAME_PREFIX + req.Name,
		Namespace: req.Namespace,
	}
	if err := r.Get(ctx, gsvcReq, &gservice); err != nil {
		//log.Infof("Unable to get gbase8s service resource, error: %s", err.Error())

		if errors.IsNotFound(err) {
			//创建service
			gsvc := NewGbase8sService(&gbase8sCluster)
			if err := r.Create(ctx, gsvc.svc); err != nil {
				log.Errorf("Create gbase8s service failed, err: %s", err.Error())
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	} else {
		//log.Infof("Get gbase8s service resource success")
	}

	//获取cm service资源
	var cmservice corev1.Service
	cmsvcReq := types.NamespacedName{
		Name:      CM_SERVICE_NAME_PREFIX + req.Name,
		Namespace: req.Namespace,
	}
	if err := r.Get(ctx, cmsvcReq, &cmservice); err != nil {
		//log.Infof("Unable to get cm service resource, error: %s", err.Error())

		if errors.IsNotFound(err) {
			//创建service
			gsvc := NewCmService(&gbase8sCluster)
			if err := r.Create(ctx, gsvc.svc); err != nil {
				log.Errorf("Create cm service failed, err: %s", err.Error())
				return ctrl.Result{}, err
			}
		} else {
			return ctrl.Result{}, err
		}
	} else {
		//log.Infof("Get cm service resource success")
	}

	//gbase8s statefulset 处理
	if gbase8sReplicas == 0 {
		//创建statefulset
		gsfs := NewGbase8sStatefulset(&gbase8sCluster)
		if err := r.Create(ctx, gsfs.sfs); err != nil {
			log.Errorf("Create gbase8s statefulset failed, err: %s", err.Error())
			return ctrl.Result{}, err
		}
		gstatefulset = gsfs.sfs
	} else if gbase8sReplicas != gbase8sExpectReplicas {
		//更新statefulset
		gsfs := NewGbase8sStatefulset(&gbase8sCluster)
		gsfs.sfs.Spec.Replicas = &gbase8sExpectReplicas
		if err := r.Update(ctx, gsfs.sfs); err != nil {
			log.Errorf("Update gbase8s statefulset failed, err: %s", err.Error())
			return ctrl.Result{}, err
		} else {
			//log.Info("Update gbase8s statefulset success")
		}
		gstatefulset = gsfs.sfs
	}

	//cm statefulset 处理
	if cmReplicas == 0 {
		//创建statefulset
		cmsfs := NewCmStatefulset(&gbase8sCluster)
		if err := r.Create(ctx, cmsfs.sfs); err != nil {
			log.Errorf("Create cm statefulset failed, err: %s", err.Error())
			return ctrl.Result{}, err
		}
		cmStatefulset = cmsfs.sfs
	} else if cmReplicas != cmExpectReplicas {
		//更新statefulset
		gsfs := NewCmStatefulset(&gbase8sCluster)
		gsfs.sfs.Spec.Replicas = &cmExpectReplicas
		if err := r.Update(ctx, gsfs.sfs); err != nil {
			log.Errorf("Update cm statefulset failed, err: %s", err.Error())
			return ctrl.Result{}, err
		} else {
			//log.Info("Update cm statefulset success")
		}
		cmStatefulset = gsfs.sfs
	}

	//log.Info("#################start######################")
	gbase8sPodLabels := map[string]string{
		GBASE8S_POD_LABEL_KEY: GBASE8S_POD_LABEL_VALUE_PREFIX + gbase8sCluster.Name,
	}
	gbase8sPods, err := GetAllPods(gbase8sCluster.Namespace, &gbase8sPodLabels)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		} else {
			return ctrl.Result{}, err
		}
	}
	_, gDomain, err := GetHostTemplate(gbase8sPods)
	if err != nil {
		return ctrl.Result{}, nil
	}

	var nodeList []*NodeBasicInfo
	for _, v := range gbase8sPods.Items {
		tempNode := &NodeBasicInfo{
			PodName:    v.Name,
			HostName:   v.Spec.Hostname,
			Namespace:  v.Namespace,
			ServerName: strings.Replace(v.Name, "-", "_", -1),
			Domain:     gDomain,
		}
		nodeList = append(nodeList, tempNode)
	}
	r.ClusterManager.AddCluster(gbase8sCluster.Name,
		gbase8sCluster.Namespace,
		&nodeList,
		gbase8sCluster.Spec.Gbase8sCfg.Failover.DetectingCount,
		gbase8sCluster.Spec.Gbase8sCfg.Failover.DetectingInterval,
		gbase8sCluster.Spec.Gbase8sCfg.Failover.Timeout)

	err = r.BuildCluster(&gbase8sCluster)
	if err != nil {
		log.Errorf("Unable to build gbase8s cluster, error: %s", err.Error())
	} else {
		r.ClusterManager.UpdateCluster(gbase8sCluster.Name, gbase8sCluster.Namespace)
	}

	//log.Info("#################end######################")

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
