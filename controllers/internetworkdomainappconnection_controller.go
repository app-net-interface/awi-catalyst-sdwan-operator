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
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	awiv1 "awi.cisco.awi/api/v1"
	awiClient "awi.cisco.awi/client"
	awipb "github.com/app-net-interface/awi-grpc/pb"
)

// AppConnectionReconciler reconciles a InterNetworkDomainAppConnection object
type AppConnectionReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	AwiClient   *awiClient.AwiGrpcClient
	ClusterName string
}

//+kubebuilder:rbac:groups=awi.cisco.awi,resources=internetworkdomainappconnections,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=awi.cisco.awi,resources=internetworkdomainappconnections/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=awi.cisco.awi,resources=internetworkdomainappconnections/finalizers,verbs=update

func (r *AppConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Request received", "req:", req.String())
	logger.Info("Reconciler called")

	var conn awiv1.InterNetworkDomainAppConnection

	if err := r.Get(ctx, req.NamespacedName, &conn); err != nil {
		logger.Error(err, "unable to fetch InterNetworkDomainAppConnection object", "namespace:", req.Namespace, "name:", req.Name)
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// name of our custom finalizer
	myFinalizerName := "internetworkdomainappconnection.awi.cisco.awi/finalizer"

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
		logger.Info("InterNetworkDomainAppConnection is being deleted", "namespace", req.Namespace, "name", req.Name)
		if controllerutil.ContainsFinalizer(&conn, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.removeAppConnection(&conn); err != nil {
				// if fail to disconnect here, return with error
				// so that it can be retried
				logger.Error(err, "Failed to send app disconnect request to AWI server")
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

	// add information about source cluster if it's not provided
	if conn.Spec.AppConnection.GetFrom().GetEndpoint() != nil &&
		strings.ToLower(conn.Spec.AppConnection.GetFrom().GetEndpoint().GetKind()) == "pod" &&
		conn.Spec.AppConnection.GetFrom().GetEndpoint().GetSelector().GetMatchCluster().GetName() == "" {
		conn.Spec.AppConnection.GetFrom().GetEndpoint().GetSelector().MatchCluster = &awipb.MatchCluster{
			Name: r.ClusterName,
		}
	}
	err := r.AwiClient.AppConnectionRequest(&conn.Spec.AppConnection)
	if err != nil {
		logger.Error(err, "Failed to send app connection request to awi server")
	}
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awiv1.InterNetworkDomainAppConnection{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				// update of app connection is not supported, so we ignore all update events
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

func (r *AppConnectionReconciler) removeAppConnection(conn *awiv1.InterNetworkDomainAppConnection) error {
	return r.AwiClient.AppDisconnectRequest(&conn.Spec.AppConnection)
}
