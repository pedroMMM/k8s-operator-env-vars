/*
Copyright 2023.

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
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "k8s.io/api/apps/v1"       // Required for Watching
	corev1 "k8s.io/api/core/v1"       // Required for Watching
	"k8s.io/apimachinery/pkg/runtime" // Required for Watching
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder" // Required for Watching
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"   // Required for Watching
	"sigs.k8s.io/controller-runtime/pkg/predicate" // Required for Watching

	// Required for Watching
	"sigs.k8s.io/controller-runtime/pkg/source" // Required for Watching
)

const (
	configMapField  = ".spec.configMap"
	targetConfigMap = "environment"
)

// EnvVarReconciler reconciles a EnvVar object
type EnvVarReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=custom.pedro,resources=envvars,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=custom.pedro,resources=envvars/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=custom.pedro,resources=envvars/finalizers,verbs=update
//+kubebuilder:rbac:groups=custom.pedro,resources=deployments,verbs=get;update;list;watch
//+kubebuilder:rbac:groups=custom.pedro,resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EnvVar object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *EnvVarReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("reconciling", "objectName", req.Name)

	deployments, err := r.getValidDeployments(ctx, req)
	if err != nil {
		log.Error(err, "failed to compile Deployment Environment Variable collection")
		return ctrl.Result{}, err
	}

	envVars, ok := deployments[req.Name]
	if !ok {
		log.Info(fmt.Sprintf("the current Deployment %q isn't on the target list", req.Name))
		return ctrl.Result{}, nil
	}

	var deployment appsv1.Deployment

	if err := r.Get(ctx, req.NamespacedName, &deployment); err != nil {
		log.Error(err, "unable to pull Deployment")
		return ctrl.Result{}, err
	}

	toUpdate := deployment.DeepCopy()

	for _, container := range toUpdate.Spec.Template.Spec.Containers {
		for _, overwrite := range envVars {
			found := false

			for _, existing := range container.Env {
				if existing.Name == overwrite.Name {
					found = true

					if existing.Value != overwrite.Value {

						log.Info(fmt.Sprintf(
							"updating Deployment %q - Container %q - Environment Variable %q - from %q to %q",
							toUpdate.Name, container.Name, overwrite.Name, existing.Value, overwrite.Value))

						existing.Value = overwrite.Value
					}

				}
			}

			if !found {
				log.Info(fmt.Sprintf(
					"adding Deployment %q - Container %q - Environment Variable %q - as %q",
					toUpdate.Name, container.Name, overwrite.Name, overwrite.Value))

				container.Env = append(container.Env, corev1.EnvVar{Name: overwrite.Name, Value: overwrite.Value})
			}
		}
	}

	toUpdate.ManagedFields = nil
	if err := r.Patch(ctx, toUpdate, client.Merge, &client.PatchOptions{FieldManager: "env-var-operator"}); err != nil {
		log.Error(err, fmt.Sprintf("failed to update the Deployment %q", toUpdate.Name))
		return ctrl.Result{}, err
	}

	// if err := r.Update(ctx, toUpdate, &client.UpdateOptions{}); err != nil {
	// 	log.Error(err, fmt.Sprintf("failed to update the Deployment %q", toUpdate.Name))
	// 	return ctrl.Result{}, err
	// }

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnvVarReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&appsv1.Deployment{}).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForConfigMap),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *EnvVarReconciler) findObjectsForConfigMap(configMap client.Object) []reconcile.Request {
	// attachedConfigDeployments := &appsv1.ConfigDeploymentList{}
	if configMap.GetName() == "test" {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      configMap.GetName(),
					Namespace: configMap.GetNamespace(),
				},
			},
		}
	}

	return []reconcile.Request{}
}

func (r *EnvVarReconciler) getValidDeployments(ctx context.Context, req ctrl.Request) (map[string][]EnvVar, error) {
	log := log.FromContext(ctx)

	var cm corev1.ConfigMap
	if err := client.IgnoreNotFound(r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: targetConfigMap}, &cm)); err != nil {
		log.Error(err, "unable to pull ConfigMap")
		return nil, err
	}

	data := map[string][]EnvVar{}

	for k, v := range cm.Data {
		var envVars []EnvVar

		if err := json.Unmarshal([]byte(v), &envVars); err != nil {
			log.Error(err, "unable to unmarshal ConfigMap data")
			return nil, err
		}

		data[k] = envVars
	}

	return data, nil
}

type EnvVar struct {
	Name  string
	Value string
}
