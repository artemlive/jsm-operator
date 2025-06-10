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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	jsmv1beta1 "github.com/artemlive/jsm-operator/api/v1beta1"
	jsmclient "github.com/artemlive/jsm-operator/internal/client"
	"github.com/go-logr/logr"
)

// JSMServiceReconciler reconciles a JSMService object
type JSMServiceReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	JSMClient *jsmclient.JSMClient
}

// +kubebuilder:rbac:groups=jsm.macpaw.dev,resources=jsmservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jsm.macpaw.dev,resources=jsmservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jsm.macpaw.dev,resources=jsmservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the JSMService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *JSMServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reconcileLog := log.FromContext(ctx)
	reconcileLog.Info("Reconciling JSMService", "service", req.NamespacedName)

	var service jsmv1beta1.JSMService
	if err := r.Get(ctx, req.NamespacedName, &service); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if service.Spec.TeamRef.Name == "" {
		reconcileLog.Info("No team specified for service, skipping reconciliation", "service", service.Name)
		return ctrl.Result{}, nil
	}

	team, err := r.getReferencedTeam(ctx, service.Namespace, service.Spec.TeamRef.Name)
	if err != nil {
		reconcileLog.Error(err, "Failed to get referenced JSMTeam", "team", service.Spec.TeamRef.Name)
		return ctrl.Result{}, err
	}

	if team.Status.ID == "" {
		reconcileLog.Info("Referenced JSMTeam has no ID, skipping service creation", "team", team.Name)
		return ctrl.Result{}, nil
	}

	if r.isUpToDate(service) {
		reconcileLog.Info("Service already exists and is up-to-date", "service", service.Name, "team", team.Name)
		return ctrl.Result{}, nil
	}

	if service.Status.ID == "" {
		return r.handleServiceCreation(ctx, &service, &team, reconcileLog)
	}

	return r.handleServiceUpdate(ctx, &service, &team, reconcileLog)
}

func (r *JSMServiceReconciler) getReferencedTeam(ctx context.Context, namespace, name string) (jsmv1beta1.JSMTeam, error) {
	var team jsmv1beta1.JSMTeam
	err := r.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, &team)
	return team, err
}

func (r *JSMServiceReconciler) isUpToDate(service jsmv1beta1.JSMService) bool {
	return service.Status.ID != "" && service.Status.ObservedGeneration == service.Generation
}

func (r *JSMServiceReconciler) handleServiceCreation(ctx context.Context, service *jsmv1beta1.JSMService, team *jsmv1beta1.JSMTeam, log logr.Logger) (ctrl.Result, error) {
	jsmName := getServiceName(service)
	jsmService, err := r.JSMClient.GetServiceByName(ctx, jsmName)
	if err != nil {
		log.Error(err, "Failed to get JSMService by name")
		return ctrl.Result{}, err
	}

	if jsmService != nil {
		return r.acquireExistingService(ctx, service, team, jsmService, log)
	}

	return r.createNewService(ctx, service, team, jsmName, log)
}

func getServiceName(service *jsmv1beta1.JSMService) string {
	if service.Spec.Name != "" {
		return service.Spec.Name
	}
	return service.Name
}

func (r *JSMServiceReconciler) acquireExistingService(ctx context.Context, service *jsmv1beta1.JSMService, team *jsmv1beta1.JSMTeam, jsmService *jsmclient.Service, log logr.Logger) (ctrl.Result, error) {
	service.Status.ID = jsmService.ID
	service.Status.Revision = jsmService.Revision
	service.Status.ObservedGeneration = service.Generation
	service.Status.TierID = jsmService.TierID
	service.Status.TierLevel = jsmService.TierLevel
	service.Status.ResolvedTeamARN = team.Status.ID

	relationshipID, err := r.ensureTeamRelationship(ctx, service, team)
	if err != nil {
		log.Error(err, "Failed to ensure team relationship")
		return ctrl.Result{}, err
	}
	service.Status.TeamRelationshipID = relationshipID

	if err := r.Status().Update(ctx, service); err != nil {
		log.Error(err, "Failed to update status after acquiring service")
		return ctrl.Result{}, err
	}

	log.Info("Acquired existing JSMService", "id", jsmService.ID)
	return ctrl.Result{}, nil
}

func (r *JSMServiceReconciler) createNewService(ctx context.Context, service *jsmv1beta1.JSMService, team *jsmv1beta1.JSMTeam, name string, log logr.Logger) (ctrl.Result, error) {
	serviceReq := jsmclient.CreateServiceRequest{
		Name:        name,
		Description: service.Spec.Description,
		CloudID:     r.JSMClient.CloudID,
		TierLevel:   service.Spec.TierLevel,
		ServiceType: service.Spec.ServiceTypeKey,
		TeamARNs:    []string{team.Status.ID},
	}

	newService, err := r.JSMClient.CreateService(ctx, &serviceReq)
	if err != nil {
		log.Error(err, "Failed to create JSMService")
		return ctrl.Result{}, err
	}

	service.Status.ID = newService.ID
	service.Status.Revision = newService.Revision
	service.Status.ObservedGeneration = service.Generation
	service.Status.TierID = newService.TierID
	service.Status.TierLevel = service.Spec.TierLevel
	service.Status.ResolvedTeamARN = team.Status.ID

	relationshipID, err := r.ensureTeamRelationship(ctx, service, team)
	if err != nil {
		log.Error(err, "Failed to create Opsgenie team relationship")
		return ctrl.Result{}, err
	}
	service.Status.TeamRelationshipID = relationshipID

	if err := r.Status().Update(ctx, service); err != nil {
		log.Error(err, "Failed to update status after create")
		return ctrl.Result{}, err
	}

	log.Info("Created new JSMService", "id", newService.ID)
	return ctrl.Result{}, nil
}

func (r *JSMServiceReconciler) handleServiceUpdate(ctx context.Context, service *jsmv1beta1.JSMService, team *jsmv1beta1.JSMTeam, log logr.Logger) (ctrl.Result, error) {
	jsmName := getServiceName(service)

	tierID := service.Status.TierID
	if service.Spec.TierLevel != service.Status.TierLevel {
		log.Info("Tier level changed, updating service tier", "oldTier", service.Status.TierLevel, "newTier", service.Spec.TierLevel)
		var err error
		tierID, err = r.JSMClient.GetTierIDByLevel(ctx, service.Spec.TierLevel)
		if err != nil {
			log.Error(err, "Failed to get tier ID by level, using current tier id")
		}
	}

	updateReq := jsmclient.UpdateServiceRequest{
		ID:          service.Status.ID,
		Revision:    service.Status.Revision,
		Name:        jsmName,
		Description: service.Spec.Description,
		TierID:      tierID,
		ServiceType: service.Spec.ServiceTypeKey,
		TeamARNs:    []string{team.Status.ID},
	}

	updSvc, err := r.JSMClient.UpdateService(ctx, &updateReq)
	if err != nil {
		if r.JSMClient.IsRevisionConflict(err) {
			log.Info("Revision conflict detected, refreshing state")
			latestService, err := r.JSMClient.GetServiceByName(ctx, jsmName)
			if err != nil {
				log.Error(err, "Failed to fetch latest service after conflict")
				return ctrl.Result{}, err
			}
			service.Status.Revision = latestService.Revision
			if err := r.Status().Update(ctx, service); err != nil {
				log.Error(err, "Failed to update status with latest revision")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "Failed to update JSMService")
		return ctrl.Result{}, err
	}

	service.Status.Revision = updSvc.Revision
	service.Status.ObservedGeneration = service.Generation
	service.Status.TierID = updSvc.TierID
	service.Status.TierLevel = updSvc.TierLevel

	if service.Status.ResolvedTeamARN != team.Status.ID {
		log.Info("Team has changed, updating team relationship")
		relationshipID, err := r.ensureTeamRelationship(ctx, service, team)
		if err != nil {
			log.Error(err, "Failed to update Opsgenie team relationship")
			return ctrl.Result{}, err
		}
		service.Status.TeamRelationshipID = relationshipID
		service.Status.ResolvedTeamARN = team.Status.ID
	}

	if err := r.Status().Update(ctx, service); err != nil {
		log.Error(err, "Failed to update status after update")
		return ctrl.Result{}, err
	}

	log.Info("Updated JSMService successfully", "id", service.Status.ID)
	return ctrl.Result{}, nil
}

func (r *JSMServiceReconciler) ensureTeamRelationship(ctx context.Context, service *jsmv1beta1.JSMService, team *jsmv1beta1.JSMTeam) (string, error) {
	relationshipID, err := r.JSMClient.CreateOpsgenieTeamRelationship(ctx, service.Status.ID, team.Status.ID)
	if err != nil {
		return "", err
	}

	service.Status.TeamRelationshipID = relationshipID
	service.Status.ResolvedTeamARN = team.Status.ID
	return relationshipID, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JSMServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jsmv1beta1.JSMService{}).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{})).
		Named("jsmservice").
		Complete(r)
}
