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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	webappv1 "liasawesomeapp.kubebuilder.io/project/api/v1"
)

// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=webapp.liasawesomeapp.kubebuilder.io,resources=redis,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webapp.liasawesomeapp.kubebuilder.io,resources=redis/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=webapp.liasawesomeapp.kubebuilder.io,resources=redis/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Redis object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *RedisReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Log.V(0).Info("reconciling redis")

	var redis webappv1.Redis
	if err := r.Get(ctx, req.NamespacedName, &redis); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	redisDeployment, err := r.createRedisDeployment(redis)
	if err != nil {
		return ctrl.Result{}, err
	}
	reidsSvc, err := r.createRedisService(redis)
	if err != nil {
		return ctrl.Result{}, err
	}

	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("redis-controller")}

	err = r.Patch(ctx, &redisDeployment, client.Apply, applyOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.Patch(ctx, &reidsSvc, client.Apply, applyOpts...)
	if err != nil {
		return ctrl.Result{}, err
	}
	redis.Status.RedisServiceName = reidsSvc.Name
	log.Log.V(0).Info("HK :###: RedisServiceName=" + redis.Status.RedisServiceName)

	if err := r.Status().Update(ctx, &redis); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RedisReconciler) createRedisDeployment(redis webappv1.Redis) (appsv1.Deployment, error) {
	defOne := int32(1)
	ci := redis.Spec.ContainerImage
	if ci == "" {
		ci = "k8s.gcr.io/redis:e2e"
	}

	depl := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      redis.Name + "-deployment",
			Namespace: redis.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &defOne,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"redis": redis.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"redis": redis.Name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "redis",
							Image: ci,
							Ports: []corev1.ContainerPort{
								{ContainerPort: 6379, Name: "redis", Protocol: "TCP"},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
									corev1.ResourceMemory: *resource.NewMilliQuantity(100000, resource.BinarySI),
								},
							},
						},
					},
				},
			},
		},
	}

	// to be able to clean-up/ gc
	if err := ctrl.SetControllerReference(&redis, &depl, r.Scheme); err != nil {
		return depl, err
	}

	return depl, nil

}

func (r *RedisReconciler) createRedisService(redis webappv1.Redis) (corev1.Service, error) {

	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: corev1.SchemeGroupVersion.String(), Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      redis.Name + "-sillywebapp-db",
			Namespace: redis.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "redis", Port: 6379, Protocol: "TCP", TargetPort: intstr.FromString("redis")},
			},
			Selector: map[string]string{"redis": redis.Name},
		},
	}

	if err := ctrl.SetControllerReference(&redis, &svc, r.Scheme); err != nil {
		return svc, err
	}

	return svc, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.Redis{}).
		Complete(r)
}
