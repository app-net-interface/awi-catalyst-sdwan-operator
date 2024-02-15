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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	awiv1 "awi.cisco.awi/api/v1"
	awiClient "awi.cisco.awi/client"
)

// InterNetworkDomainReconciler reconciles a InterNetworkDomain object
type InterNetworkDomainReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	AwiClient *awiClient.AwiGrpcClient
}

//+kubebuilder:rbac:groups=awi.cisco.awi,resources=internetworkdomains,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=awi.cisco.awi,resources=internetworkdomains/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=awi.cisco.awi,resources=internetworkdomains/finalizers,verbs=update

func (r *InterNetworkDomainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Request received", "req:", req.String())
	logger.Info("Reconciler called")

	var conn awiv1.InterNetworkDomain

	if err := r.Get(ctx, req.NamespacedName, &conn); err != nil {
		logger.Error(err, "unable to fetch InterNetworkDomain object", "namespace:", req.Namespace, "name:", req.Name)
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// name of our custom finalizer
	myFinalizerName := "internetworkdomain.awi.cisco.awi/finalizer"

	// examine DeletionTimestamp to determine if object is under deletion
	if conn.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&conn, myFinalizerName) {
			controllerutil.AddFinalizer(&conn, myFinalizerName)
			if err := r.Update(ctx, &conn); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		logger.Info("InterNetworkDomain is being deleted", "namespace", req.Namespace, "name", req.Name)
		if controllerutil.ContainsFinalizer(&conn, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.removeConnection(&conn); err != nil {
				// if fail to disconnect here, return with error
				// so that it can be retried
				logger.Error(err, "Failed to send disconnect request to awi server")
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&conn, myFinalizerName)
			if err := r.Update(ctx, &conn); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	err := r.AwiClient.ConnectionRequest(&conn.Spec)
	if err != nil {
		logger.Error(err, "Failed to send connection request to awi server")
	}
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *InterNetworkDomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awiv1.InterNetworkDomain{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// update of network domains connection is not supported, so we ignore all update events
				// except for cases when deletion timestamp is not zero, this means object is being deleted, and
				// we want to call finalizer
				return !e.ObjectNew.GetDeletionTimestamp().IsZero()
			},
			DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
				// ignore delete events as delete logic is being handled by finalizer
				return false
			},
		}).
		Complete(r)
}

func (r *InterNetworkDomainReconciler) removeConnection(conn *awiv1.InterNetworkDomain) error {
	// sending disconnect request
	return r.AwiClient.DisconnectRequest(&conn.Spec)
}
