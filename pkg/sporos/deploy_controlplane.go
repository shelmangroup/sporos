package sporos

import (
	"fmt"

	api "github.com/shelmangroup/sporos/pkg/apis/sporos/v1alpha1"
	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func createExternalEndpoint(cr *api.Sporos) (*corev1.Service, error) {
	selector := LabelsForSporos(cr.Name)

	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kube-api-server", cr.Name),
			Namespace: cr.Namespace,
			Labels:    selector,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeLoadBalancer,
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Name:     "https",
					Protocol: corev1.ProtocolTCP,
					Port:     443,
				},
			},
		},
	}
	addOwnerRefToObject(svc, asOwner(cr))
	err := sdk.Create(svc)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("failed to create sporos service: %v", err)
	}
	return svc, nil
}

func isServiceEndpointReady(cr *api.Sporos, s *corev1.Service) (bool, error) {
	err := sdk.Get(s)
	if err != nil {
		return false, err
	}
	if len(s.Status.LoadBalancer.Ingress) < 1 {
		return false, nil
	}
	cr.Status.ApiServerIP = s.Status.LoadBalancer.Ingress[0].IP
	sdk.Update(cr)
	log.Infof("API server endpoint: %s", cr.Status.ApiServerIP)
	return true, nil
}

func deployControlplane(cr *api.Sporos) error {
	selector := LabelsForSporos(cr.GetName())

	podTempl := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.GetName(),
			Namespace: cr.GetNamespace(),
			Labels:    selector,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{apiserverContainer(cr)},
			Volumes: []corev1.Volume{{
				Name: "secrets",
				VolumeSource: corev1.VolumeSource{
					Projected: &corev1.ProjectedVolumeSource{
						Sources: []corev1.VolumeProjection{{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "kube-apiserver-secrets",
								},
							},
						}},
					},
				},
			}},
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:    func(i int64) *int64 { return &i }(65543),
				RunAsNonRoot: func(b bool) *bool { return &b }(true),
			},
		},
	}
	if cr.Spec.Pod != nil {
		applyPodPolicy(&podTempl.Spec, cr.Spec.Pod)
	}

	d := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.GetName(),
			Namespace: cr.GetNamespace(),
			Labels:    selector,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &cr.Spec.Nodes,
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: podTempl,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: func(a intstr.IntOrString) *intstr.IntOrString { return &a }(intstr.FromInt(1)),
					MaxSurge:       func(a intstr.IntOrString) *intstr.IntOrString { return &a }(intstr.FromInt(1)),
				},
			},
		},
	}
	addOwnerRefToObject(d, asOwner(cr))
	err := sdk.Create(d)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func apiserverContainer(cr *api.Sporos) corev1.Container {
	return corev1.Container{
		Name:  "kube-apiserver",
		Image: fmt.Sprintf("%s:%s", cr.Spec.BaseImage, cr.Spec.Version),
		Command: []string{
			"/hyperkube",
			"apiserver",
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "secrets",
			MountPath: "/etc/kubernetes/secrets",
		}},
		Ports: []corev1.ContainerPort{{
			Name:          "https",
			ContainerPort: int32(443),
		}},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/v1/health",
					Port:   intstr.FromInt(443),
					Scheme: corev1.URISchemeHTTPS,
				},
			},
			InitialDelaySeconds: 10,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			FailureThreshold:    3,
		},
	}
}
func controllerContainer(cr *api.Sporos) corev1.Container {
	return corev1.Container{
		Name:  "kube-apiserver",
		Image: fmt.Sprintf("%s:%s", cr.Spec.BaseImage, cr.Spec.Version),
		Command: []string{
			"/hyperkube",
			"controller-manager",
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "secrets",
			MountPath: "/etc/kubernetes/secrets",
		}},
		Ports: []corev1.ContainerPort{{
			Name:          "https",
			ContainerPort: int32(443),
		}},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/v1/health",
					Port:   intstr.FromInt(443),
					Scheme: corev1.URISchemeHTTPS,
				},
			},
			InitialDelaySeconds: 10,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			FailureThreshold:    3,
		},
	}
}

func schedulerContainer(cr *api.Sporos) corev1.Container {
	return corev1.Container{
		Name:  "kube-scheduler",
		Image: fmt.Sprintf("%s:%s", cr.Spec.BaseImage, cr.Spec.Version),
		Command: []string{
			"/hyperkube",
			"scheduler",
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "secrets",
			MountPath: "/etc/kubernetes/secrets",
		}},
		Ports: []corev1.ContainerPort{{
			Name:          "https",
			ContainerPort: int32(443),
		}},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/v1/health",
					Port:   intstr.FromInt(443),
					Scheme: corev1.URISchemeHTTPS,
				},
			},
			InitialDelaySeconds: 10,
			TimeoutSeconds:      10,
			PeriodSeconds:       10,
			FailureThreshold:    3,
		},
	}
}

func applyPodPolicy(s *corev1.PodSpec, p *api.PodPolicy) {
	for i := range s.Containers {
		s.Containers[i].Resources = p.Resources
	}

	for i := range s.InitContainers {
		s.InitContainers[i].Resources = p.Resources
	}
}

// IsPodReady checks the status of the
// pod for the Ready condition
func IsPodReady(p corev1.Pod) bool {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}
