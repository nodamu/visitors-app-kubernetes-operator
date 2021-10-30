package controllers

import (
	"context"
	v1 "github.com/nodamu/visitors-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func mysqlAuthName() string {
	return "mysql-auth"
}

func mysqlServiceName() string {
	return "mysql-service"
}

func mysqlDeploymentName() string {
	return "mysql"
}

func (r *VisitorsAppReconciler) mysqlAuthSecret(v *v1.VisitorsApp) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysqlAuthName(),
			Namespace: v.Namespace,
		},
		StringData: map[string]string{
			"username": "visitors-user",
			"password": "visitors-pass",
		},
		Type: "Opaque",
	}
	controllerutil.SetControllerReference(v, secret, r.Scheme)

	return secret
}

func (r *VisitorsAppReconciler) mysqlDeployment(v *v1.VisitorsApp) *appsv1.Deployment {
	labels := map[string]string{
		"app":             "visitors",
		"visitorssite_cr": v.Name,
		"tier":            "mysql",
	}

	size := int32(1)

	userSecret := &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: mysqlAuthName()},
			Key:                  "username",
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
			Name:      mysqlDeploymentName(),
			Namespace: v.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "visitors-mysql",
							Image: "mysql:5.7",
							Ports: []corev1.ContainerPort{{
								Name:          "mysql",
								ContainerPort: 3306,
							}},
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_ROOT_PASSWORD",
									Value: "password",
								},
								{
									Name:  "MYSQL_DATABASE",
									Value: "visitors",
								},
								{
									Name:      "MYSQL_USER",
									ValueFrom: userSecret,
								},
								{
									Name:      "MYSQL_PASSWORD",
									ValueFrom: passwordSecret,
								},
							},
						},
					},
				},
			},
		},
	}

	controllerutil.SetControllerReference(v, deployment, r.Scheme)

	return deployment
}

func (r *VisitorsAppReconciler) mysqlService(v *v1.VisitorsApp) *corev1.Service {
	labels := map[string]string{
		"app":             "visitors",
		"visitorssite_cr": v.Name,
		"tier":            "mysql",
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mysqlServiceName(),
			Namespace: v.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 3306,
				},
			},
			Selector:  labels,
			ClusterIP: "None",
		},
	}

	controllerutil.SetControllerReference(v, service, r.Scheme)

	return service
}

//Check if mysql deployment is ready
func (r *VisitorsAppReconciler) mysqlIsUp(v *v1.VisitorsApp) bool {
	deployment := &appsv1.Deployment{}

	err := r.Client.Get(context.TODO(), types.NamespacedName{
		Name:      mysqlDeploymentName(),
		Namespace: v.Namespace,
	}, deployment)

	if err != nil {
		log.Error(err, "Deployment mysql not found")
		return false
	}

	if deployment.Status.ReadyReplicas == 1 {
		return true
	}

	return false
}
