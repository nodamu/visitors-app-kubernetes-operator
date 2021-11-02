/*
Copyright 2021.

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
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	webv1 "github.com/nodamu/visitors-operator/api/v1"
)

// VisitorsAppReconciler reconciles a VisitorsApp object
type VisitorsAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=web.nickadamu.tech,resources=visitorsapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=web.nickadamu.tech,resources=visitorsapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=web.nickadamu.tech,resources=visitorsapps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VisitorsApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *VisitorsAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	reqlogger := log.Log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqlogger.Info("Reconciling VisitorApp")

	// your logic here

	v := &webv1.VisitorsApp{}

	err := r.Client.Get(ctx, req.NamespacedName, v)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	var result *reconcile.Result

	// ========== MySQL ====================

	result, err = r.ensureSecret(req, v, r.mysqlAuthSecret(v))

	if err != nil {
		return *result, err
	}

	result, err = r.ensureDeployment(req, v, r.mysqlDeployment(v))

	if err != nil {
		return *result, err
	}

	result, err = r.ensureService(req, v, r.mysqlService(v))

	if err != nil {
		return *result, err
	}

	mysqlRunning := r.mysqlIsUp(v)

	if !mysqlRunning {
		// If MySQL isn't running yet, requeue the reconcile
		// to run again after a delay
		delay := time.Second * time.Duration(5)

		log.Log.Info(fmt.Sprintf("MySQL isn't running, waiting for %s", delay))
		return reconcile.Result{RequeueAfter: delay}, nil
	}

	// == Visitors Backend  ==========
	result, err = r.ensureDeployment(req, v, r.backendDeployment(v))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(req, v, r.backendService(v))
	if result != nil {
		return *result, err
	}

	err = r.updateBackendStatus(v)
	if err != nil {
		// Requeue the request if the status could not be updated
		return reconcile.Result{}, err
	}

	result, err = r.handleBackendChanges(v)
	if result != nil {
		return *result, err
	}

	// == Visitors Frontend ==========
	result, err = r.ensureDeployment(req, v, r.frontendDeployment(v))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(req, v, r.frontendService(v))
	if result != nil {
		return *result, err
	}

	err = r.updateFrontendStatus(v)
	if err != nil {
		// Requeue the request
		return reconcile.Result{}, err
	}

	result, err = r.handleFrontendChanges(v)
	if result != nil {
		return *result, err
	}

	// == Finish ==========
	// Everything went fine, don't requeue

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VisitorsAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//// Create a new controller
	//c, err := controller.New("visitorsapp-controller", mgr, controller.Options{Reconciler: r})
	//if err != nil {
	//	return err
	//}
	//
	//// Watch for changes to primary resource VisitorsApp
	//err = c.Watch(&source.Kind{Type: &webv1.VisitorsApp{}}, &handler.EnqueueRequestForObject{})
	//if err != nil {
	//	return err
	//}
	//
	//// TODO(user): Modify this to be the types you create that are owned by the primary resource
	//// Watch for changes to secondary resource Pods and requeue the owner VisitorsApp
	//err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
	//	IsController: true,
	//	OwnerType:    &webv1.VisitorsApp{},
	//})
	//if err != nil {
	//	return err
	//}
	//
	//err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
	//	IsController: true,
	//	OwnerType:    &webv1.VisitorsApp{},
	//})
	//if err != nil {
	//	return err
	//}

	// Operator SDK 1.0.0+
	return ctrl.NewControllerManagedBy(mgr).
		For(&webv1.VisitorsApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
