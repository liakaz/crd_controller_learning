/*
Copyright 2022.

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
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	webappv1 "liasawesomeapp.kubebuilder.io/project/api/v1"
)

// SillyWebappReconciler reconciles a SillyWebapp object
type SillyWebappReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=webapp.liasawesomeapp.kubebuilder.io,resources=sillywebapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webapp.liasawesomeapp.kubebuilder.io,resources=sillywebapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;get;patch;create;update
//+kubebuilder:rbac:groups=core,resources=services,verbs=list;watch;get;patch;create;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SillyWebapp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *SillyWebappReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Log.V(0).Info("reconciling sillywebapp")

	var sillywebapp webappv1.SillyWebapp
	if err := r.Get(ctx, req.NamespacedName, &sillywebapp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var redis webappv1.Redis
	redisName := client.ObjectKey{Name: sillywebapp.Spec.RedisName, Namespace: req.Namespace}
	if err := r.Get(ctx, redisName, &redis); err != nil {
		log.Log.Error(err, "didn't get redis")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	deployment, err := r.createDeployment(sillywebapp, redis)
	if err != nil {
		return ctrl.Result{}, err
	}
	svc, err := r.createService(sillywebapp)
	if err != nil {
		return ctrl.Result{}, err
	}

	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("sillywebapp-controller")}

	err = r.Patch(ctx, &deployment, client.Apply, applyOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Patch(ctx, &svc, client.Apply, applyOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}

	sillywebapp.Status.URL = getServiceURL(svc, sillywebapp.Spec.Frontend.ServingPort)

	err = r.Status().Update(ctx, &sillywebapp)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Log.V(0).Info("reconciled sillywebapp")

	return ctrl.Result{}, nil
}

func getServiceURL(svc corev1.Service, port int32) string {
	if svc.Spec.ClusterIP == "" {
		log.Log.V(0).Info("service cluster IP is empty")
		return ""
	}
	return "http://" + svc.Spec.ClusterIP + ":" + strconv.Itoa(int(port))
}

func (r *SillyWebappReconciler) createService(sillywebapp webappv1.SillyWebapp) (corev1.Service, error) {
	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: corev1.SchemeGroupVersion.String(), Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      sillywebapp.Name,
			Namespace: sillywebapp.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 8080, Protocol: "TCP", TargetPort: intstr.FromString("http")},
			},
			Selector: map[string]string{"sillywebapp": sillywebapp.Name},
			Type:     corev1.ServiceTypeClusterIP,
		},
	}

	// set the controller reference to be able to cleanup during delete/gc
	if err := ctrl.SetControllerReference(&sillywebapp, &svc, r.Scheme); err != nil {
		return svc, err
	}

	return svc, nil
}

func (r *SillyWebappReconciler) createDeployment(sillywebapp webappv1.SillyWebapp, redis webappv1.Redis) (appsv1.Deployment, error) {

	log.Log.V(0).Info("HK - WR-createDeployment :###: RedisServiceName=" + redis.Status.RedisServiceName + "\n")

	depl := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      sillywebapp.Name,
			Namespace: sillywebapp.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: sillywebapp.Spec.Frontend.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"sillywebapp": sillywebapp.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"sillywebapp": sillywebapp.Name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "frontend",
							Image: "hmanikkothu/watchlist:v1",
							Env: []corev1.EnvVar{
								{Name: "REDIS_HOST", Value: redis.Status.RedisServiceName},
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 8080, Name: "http", Protocol: "TCP"},
							},
							Resources: *sillywebapp.Spec.Frontend.Resources.DeepCopy(),
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(&sillywebapp, &depl, r.Scheme); err != nil {
		return depl, err
	}

	return depl, nil
}

func (r *SillyWebappReconciler) sillyAppsUsingRedis(obj client.Object) []reconcile.Request {
	listOptions := []client.ListOption{
		client.InNamespace(obj.GetNamespace()),
		client.MatchingFields{".spec.redisName": obj.GetName()},
	}
	var list webappv1.SillyWebappList
	if err := r.List(context.Background(), &list, listOptions...); err != nil {
		log.Log.Error(err, "unable to list SillyWebapp")
		return nil
	}
	res := make([]reconcile.Request, len(list.Items))
	for i, sillywebapplist := range list.Items {
		res[i].Name = sillywebapplist.Name
		res[i].Namespace = sillywebapplist.Namespace
	}
	return res
}

// SetupWithManager sets up the controller with the Manager.
func (r *SillyWebappReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&webappv1.SillyWebapp{},
		".spec.redisName",
		func(rawObj client.Object) []string {
			redisName := rawObj.(*webappv1.SillyWebapp).Spec.RedisName
			if redisName == "" {
				return nil
			}
			return []string{redisName}
		})

	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.SillyWebapp{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Watches(
			&source.Kind{Type: &webappv1.Redis{}},
			handler.EnqueueRequestsFromMapFunc(r.sillyAppsUsingRedis),
		).
		Complete(r)
}
