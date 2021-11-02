package controllers

import (
	"context"
	v1 "github.com/nodamu/visitors-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const backendPort = 8000
const backendServicePort = 30685
const backendImage = "jdob/visitors-service:1.0.0"

func backendDeployment(v *v1.VisitorsApp) string {
	return v.Name + "-backend"
}

func backendServiceName(v *v1.VisitorsApp) string {
	return v.Name + "-backend-service"
}

func (r *VisitorsAppReconciler) backendDeployment(v *v1.VisitorsApp) *appsv1.Deployment {
	labels := labels(v, "backend")

	size := v.Spec.Size

	userSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: mysqlAuthName(),
			},
			Key: "username",
		},
	}

	passwordSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: mysqlAuthName(),
			},
			Key: "password",
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backendDeployment(v),
			Namespace: v.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      backendDeployment(v),
					Namespace: v.Namespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "visitors-backend",
						Image: backendImage,
						Ports: []corev1.ContainerPort{
							{
								Name:          "visitors-backend",
								ContainerPort: backendPort,
							},
						},
						ImagePullPolicy: corev1.PullAlways,
						Env: []corev1.EnvVar{
							{
								Name:  "MYSQL_DATABASE",
								Value: "visitors",
							},
							{
								Name:  "MYSQL_SERVICE_HOST",
								Value: mysqlServiceName(),
							},
							{
								Name:      "MYSQL_USERNAME",
								ValueFrom: userSecret,
							},
							{
								Name:      "MYSQL_PASSWORD",
								ValueFrom: passwordSecret,
							},
						},
					}},
				},
			},
		},
	}

	controllerutil.SetControllerReference(v, deployment, r.Scheme)
	return deployment
}

func (r *VisitorsAppReconciler) backendService(v *v1.VisitorsApp) *corev1.Service {
	labels := labels(v, "backend")

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backendServiceName(v),
			Namespace: v.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Port:       backendPort,
				TargetPort: intstr.FromInt(backendPort),
				NodePort:   30685,
			}},
			Selector: labels,
			Type:     corev1.ServiceTypeNodePort,
		},
	}

	controllerutil.SetControllerReference(v, service, r.Scheme)

	return service

}

func (r *VisitorsAppReconciler) updateBackendStatus(v *v1.VisitorsApp) error {
	v.Status.BackendImage = backendImage

	//Dont use context.TODO in production
	err := r.Client.Update(context.TODO(), v)

	return err
}

func (r *VisitorsAppReconciler) handleBackendChanges(v *v1.VisitorsApp) (*reconcile.Result, error) {
	found := &appsv1.Deployment{}

	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: v.Namespace,
		Name:      backendDeployment(v),
	}, found)

	if err != nil {
		// The deployment may not have been created yet, so requeue
		return &reconcile.Result{RequeueAfter: 5 * time.Second}, err
	}

	size := v.Spec.Size

	if size != *found.Spec.Replicas {
		found.Spec.Replicas = &size
		r.Client.Update(context.TODO(), found)
		if err != nil {
			log.Log.Error(err, "Failed to update Deployment.", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return &reconcile.Result{}, err
		}

		return &reconcile.Result{Requeue: true}, nil

	}

	return nil, nil

}
