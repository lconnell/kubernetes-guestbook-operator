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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	log "sigs.k8s.io/controller-runtime/pkg/log"

	examplev1alpha1 "connell.com/guestbook-operator/api/v1alpha1"
)

const (
	// PhaseError indicates that an error occurred during reconciliation.
	PhaseError = "Error"
	// PhaseProcessed indicates that the resource has been successfully processed.
	PhaseProcessed = "Processed"
)

// GuestbookEntryReconciler reconciles a GuestbookEntry object
type GuestbookEntryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=example.connell.com,resources=guestbookentries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=example.connell.com,resources=guestbookentries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=example.connell.com,resources=guestbookentries/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GuestbookEntry object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *GuestbookEntryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctrlLog := log.FromContext(ctx)
	ctrlLog.Info("Reconciling GuestbookEntry")

	// 1. Fetch the GuestbookEntry instance
	guestbookEntry := &examplev1alpha1.GuestbookEntry{}
	if err := r.Get(ctx, req.NamespacedName, guestbookEntry); err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			ctrlLog.Info("GuestbookEntry resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		ctrlLog.Error(err, "Failed to get GuestbookEntry")
		return ctrl.Result{}, err
	}

	// 2. Define the desired ConfigMap
	configMapName := guestbookEntry.Name + "-entry"
	desiredConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: guestbookEntry.Namespace,
			Labels:    map[string]string{"app": "guestbook", "entry": guestbookEntry.Name},
		},
		Data: map[string]string{
			"name":    guestbookEntry.Spec.Name,
			"message": guestbookEntry.Spec.Message,
		},
	}

	// Set GuestbookEntry instance as the owner and controller
	if err := ctrl.SetControllerReference(guestbookEntry, desiredConfigMap, r.Scheme); err != nil {
		ctrlLog.Error(err, "Failed to set controller reference on ConfigMap")
		// Update status and requeue
		guestbookEntry.Status.Phase = PhaseError
		guestbookEntry.Status.Message = "Failed to set owner reference for ConfigMap"
		if statusUpdateErr := r.Status().Update(ctx, guestbookEntry); statusUpdateErr != nil {
			ctrlLog.Error(statusUpdateErr, "Failed to update GuestbookEntry status after owner ref error")
		}
		return ctrl.Result{}, err
	}

	// 3. Check if the ConfigMap already exists, if not create it
	foundConfigMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: guestbookEntry.Namespace}, foundConfigMap)
	if err != nil && apierrors.IsNotFound(err) {
		ctrlLog.Info("Creating a new ConfigMap", "ConfigMap.Namespace", desiredConfigMap.Namespace, "ConfigMap.Name", desiredConfigMap.Name)
		err = r.Create(ctx, desiredConfigMap)
		if err != nil {
			ctrlLog.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", desiredConfigMap.Namespace, "ConfigMap.Name", desiredConfigMap.Name)
			guestbookEntry.Status.Phase = PhaseError
			guestbookEntry.Status.Message = "Failed to create ConfigMap"
			if statusUpdateErr := r.Status().Update(ctx, guestbookEntry); statusUpdateErr != nil {
				ctrlLog.Error(statusUpdateErr, "Failed to update GuestbookEntry status after CM create error")
			}
			return ctrl.Result{}, err
		}
		// ConfigMap created successfully - update status and requeue to check later if needed
		guestbookEntry.Status.Phase = PhaseProcessed
		guestbookEntry.Status.Message = "ConfigMap created successfully"
		if err := r.Status().Update(ctx, guestbookEntry); err != nil {
			ctrlLog.Error(err, "Failed to update GuestbookEntry status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil // Requeue to ensure status is updated and observed
	} else if err != nil {
		ctrlLog.Error(err, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	}

	// 4. If ConfigMap already exists, check if it needs update
	ctrlLog.Info("ConfigMap already exists. Ensuring data is up-to-date.")
	if foundConfigMap.Data["name"] != desiredConfigMap.Data["name"] || foundConfigMap.Data["message"] != desiredConfigMap.Data["message"] {
		ctrlLog.Info("ConfigMap data out of sync, updating...")
		foundConfigMap.Data = desiredConfigMap.Data // Update data
		err = r.Update(ctx, foundConfigMap)
		if err != nil {
			ctrlLog.Error(err, "Failed to update existing ConfigMap")
			guestbookEntry.Status.Phase = PhaseError
			guestbookEntry.Status.Message = "Failed to update ConfigMap"
			if statusUpdateErr := r.Status().Update(ctx, guestbookEntry); statusUpdateErr != nil {
				ctrlLog.Error(statusUpdateErr, "Failed to update GuestbookEntry status after CM update error")
			}
			return ctrl.Result{}, err
		}
		ctrlLog.Info("ConfigMap updated.")
		guestbookEntry.Status.Phase = PhaseProcessed
		guestbookEntry.Status.Message = "ConfigMap updated successfully"
	} else {
		ctrlLog.Info("ConfigMap data is already in sync.")
		guestbookEntry.Status.Phase = PhaseProcessed
		guestbookEntry.Status.Message = "ConfigMap is in desired state"
	}

	if err := r.Status().Update(ctx, guestbookEntry); err != nil {
		ctrlLog.Error(err, "Failed to update GuestbookEntry status")
		return ctrl.Result{}, err
	}

	ctrlLog.Info("Successfully reconciled GuestbookEntry")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GuestbookEntryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplev1alpha1.GuestbookEntry{}).
		Owns(&corev1.ConfigMap{}). // Add this to watch ConfigMaps created by GuestbookEntry
		Complete(r)
}
