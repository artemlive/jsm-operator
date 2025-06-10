/*
Copyright 2025.

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

package controller

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	jsmv1beta1 "github.com/artemlive/jsm-operator/api/v1beta1"
	jsmclient "github.com/artemlive/jsm-operator/internal/client"
)

// JSMTeamReconciler reconciles a JSMTeam object
type JSMTeamReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	JSMClient *jsmclient.JSMClient
}

// +kubebuilder:rbac:groups=jsm.macpaw.dev,resources=jsmteams,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jsm.macpaw.dev,resources=jsmteams/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jsm.macpaw.dev,resources=jsmteams/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the JSMTeam object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *JSMTeamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var team jsmv1beta1.JSMTeam
	if err := r.Get(ctx, req.NamespacedName, &team); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	teamName := team.Name
	if team.Spec.Name != "" {
		teamName = team.Spec.Name
	}

	var resolvedID string
	switch {
	case team.Spec.ID != "":
		// if Spec.ID is provided, prefer it
		resolvedID = team.Spec.ID
	case team.Status.ID != "":
		// if status already has ID, reuse it
		resolvedID = team.Status.ID
	default:
		var err error
		resolvedID, err = r.JSMClient.GetOpsgenieTeamIDByName(ctx, teamName)
		if err != nil {
			logger.Error(err, "unable to get team ID by name", "name", teamName)
			return ctrl.Result{}, err
		}
	}

	if team.Status.ID != resolvedID || team.Status.ObservedGeneration != team.ObjectMeta.Generation {
		team.Status.ID = resolvedID
		team.Status.ObservedGeneration = team.ObjectMeta.Generation

		if err := r.Status().Update(ctx, &team); err != nil {
			logger.Error(err, "unable to update JSMTeam status")
			return ctrl.Result{}, err
		}
	}

	logger.Info("successfully synced JSMTeam ID to status", "name", req.NamespacedName, "id", resolvedID, "teamName", teamName)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JSMTeamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			RateLimiter: workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](
				20*time.Second,
				5*time.Minute,
			),
		}).
		For(&jsmv1beta1.JSMTeam{}).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{})).
		Named("jsmteam").
		Complete(r)
}
