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
	apiServerSelector := fmt.Sprintf("%s-kube-apiserver", cr.GetName())
	selector := LabelsForSporos(apiServerSelector)

	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kube-apiserver", cr.Name),
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

func deployControlplane(cr *api.Sporos) ([]*appsv1.Deployment, error) {
	var deployments []*appsv1.Deployment

	apiServerSecret := fmt.Sprintf("%s-kube-apiserver", cr.GetName())
	apiServerName := fmt.Sprintf("%s-kube-apiserver", cr.GetName())
	log.Debugf("Deploying kube-apiserver")
	api, err := createDeployment(cr, apiServerName, apiServerSecret, apiserverContainer)
	if err != nil {
		return nil, err
	}

	controllerSecret := fmt.Sprintf("%s-kube-controller-manager", cr.GetName())
	controllerName := fmt.Sprintf("%s-kube-controller-manager", cr.GetName())
	log.Debugf("Deploying kube-controller-manager")
	control, err := createDeployment(cr, controllerName, controllerSecret, controllerContainer)
	if err != nil {
		return nil, err
	}

	schedulerSecret := fmt.Sprintf("%s-kubeconfig", cr.GetName())
	schedulerName := fmt.Sprintf("%s-kube-scheduler", cr.GetName())
	log.Debugf("Deploying kube-scheduler")
	scheduler, err := createDeployment(cr, schedulerName, schedulerSecret, schedulerContainer)
	if err != nil {
		return nil, err
	}
	deployments = append(deployments, api, control, scheduler)

	return deployments, nil
}

func createDeployment(cr *api.Sporos, name, secretName string, containerfn func(*api.Sporos) corev1.Container) (*appsv1.Deployment, error) {
	selector := LabelsForSporos(name)

	podTempl := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.GetNamespace(),
			Labels:    selector,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{containerfn(cr)},
			Volumes: []corev1.Volume{{
				Name: "secrets",
				VolumeSource: corev1.VolumeSource{
					Projected: &corev1.ProjectedVolumeSource{
						Sources: []corev1.VolumeProjection{{
							Secret: &corev1.SecretProjection{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: secretName,
								},
							},
						}},
					},
				},
			}},
		},
	}
	if cr.Spec.Pod != nil {
		applyPodPolicy(&podTempl.Spec, cr.Spec.Pod)
	}

	var replicas int32
	replicas = 1

	d := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.GetNamespace(),
			Labels:    selector,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
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
		return nil, err
	}
	return d, nil
}

func apiserverContainer(cr *api.Sporos) corev1.Container {
	return corev1.Container{
		Name:  "kube-apiserver",
		Image: fmt.Sprintf("%s:%s", cr.Spec.BaseImage, cr.Spec.Version),
		Command: []string{
			"/hyperkube",
			"apiserver",
			"--enable-admission-plugins=NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultTolerationSeconds,DefaultStorageClass,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota,NodeRestriction",
			"--advertise-address=" + cr.Status.ApiServerIP,
			"--allow-privileged=true",
			"--anonymous-auth=false",
			"--authorization-mode=Node,RBAC",
			"--bind-address=0.0.0.0",
			"--client-ca-file=/etc/kubernetes/secrets/ca.crt",
			"--enable-bootstrap-token-auth=true",
			"--etcd-cafile=/etc/kubernetes/secrets/etcd-client-ca.crt",
			"--etcd-certfile=/etc/kubernetes/secrets/etcd-client.crt",
			"--etcd-keyfile=/etc/kubernetes/secrets/etcd-client.key",
			"--etcd-servers=" + etcdURLForSporos(cr.Name),
			"--insecure-port=0",
			"--kubelet-client-certificate=/etc/kubernetes/secrets/apiserver.crt",
			"--kubelet-client-key=/etc/kubernetes/secrets/apiserver.key",
			"--secure-port=443",
			"--service-account-key-file=/etc/kubernetes/secrets/service-account.pub",
			"--service-cluster-ip-range=" + cr.Spec.ServiceCIDR,
			"--storage-backend=etcd3",
			"--tls-cert-file=/etc/kubernetes/secrets/apiserver.crt",
			"--tls-private-key-file=/etc/kubernetes/secrets/apiserver.key",
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "secrets",
			MountPath: "/etc/kubernetes/secrets",
		}},
		Ports: []corev1.ContainerPort{{
			Name:          "https",
			ContainerPort: int32(443),
		}},
		// ReadinessProbe: &corev1.Probe{
		// 	Handler: corev1.Handler{
		// 		HTTPGet: &corev1.HTTPGetAction{
		// 			Path:   "/healthz",
		// 			Port:   intstr.FromInt(443),
		// 			Scheme: corev1.URISchemeHTTPS,
		// 		},
		// 	},
		// 	InitialDelaySeconds: 10,
		// 	TimeoutSeconds:      10,
		// 	PeriodSeconds:       10,
		// 	FailureThreshold:    3,
		// },
	}
}
func controllerContainer(cr *api.Sporos) corev1.Container {
	return corev1.Container{
		Name:  "kube-controller-manager",
		Image: fmt.Sprintf("%s:%s", cr.Spec.BaseImage, cr.Spec.Version),
		Command: []string{
			"/hyperkube",
			"controller-manager",
			"--use-service-account-credentials",
			"--cluster-cidr=" + cr.Spec.PodCIDR,
			"--allocate-node-cidrs=true",
			"--service-cluster-ip-range=" + cr.Spec.ServiceCIDR,
			"--kubeconfig=/etc/kubernetes/secrets/kubeconfig",
			"--cluster-signing-cert-file=/etc/kubernetes/secrets/ca.crt",
			"--cluster-signing-key-file=/etc/kubernetes/secrets/ca.key",
			"--configure-cloud-routes=false",
			"--leader-elect=true",
			"--root-ca-file=/etc/kubernetes/secrets/ca.crt",
			"--service-account-private-key-file=/etc/kubernetes/secrets/service-account.key",
			"--use-service-account-credentials=true",
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "secrets",
			MountPath: "/etc/kubernetes/secrets",
		}},
		// LivenessProbe: &corev1.Probe{
		// 	Handler: corev1.Handler{
		// 		HTTPGet: &corev1.HTTPGetAction{
		// 			Path:   "/healthz",
		// 			Port:   intstr.FromInt(10252),
		// 			Scheme: corev1.URISchemeHTTPS,
		// 		},
		// 	},
		// 	InitialDelaySeconds: 15,
		// 	TimeoutSeconds:      15,
		// },
	}
}

func schedulerContainer(cr *api.Sporos) corev1.Container {
	return corev1.Container{
		Name:  "kube-scheduler",
		Image: fmt.Sprintf("%s:%s", cr.Spec.BaseImage, cr.Spec.Version),
		Command: []string{
			"/hyperkube",
			"scheduler",
			"--kubeconfig=/etc/kubernetes/secrets/kubeconfig",
			"--leader-elect=true",
		},
		VolumeMounts: []corev1.VolumeMount{{
			Name:      "secrets",
			MountPath: "/etc/kubernetes/secrets",
		}},
		// LivenessProbe: &corev1.Probe{
		// 	Handler: corev1.Handler{
		// 		HTTPGet: &corev1.HTTPGetAction{
		// 			Path:   "/healthz",
		// 			Port:   intstr.FromInt(10251),
		// 			Scheme: corev1.URISchemeHTTPS,
		// 		},
		// 	},
		// 	InitialDelaySeconds: 15,
		// 	TimeoutSeconds:      15,
		// },
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

// IsControlplaneReady checks the status of the
// pod for the Ready condition
func IsControlplaneReady(d *appsv1.Deployment) (bool, error) {
	err := sdk.Get(d)
	if err != nil {
		return false, err
	}
	for _, c := range d.Status.Conditions {
		if c.Type == appsv1.DeploymentProgressing {
			return c.Status == corev1.ConditionTrue, nil
		}
	}
	return false, nil
}
