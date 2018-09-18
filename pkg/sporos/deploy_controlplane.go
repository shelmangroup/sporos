package sporos

import (
	"fmt"

	api "github.com/shelmangroup/sporos/pkg/apis/sporos/v1alpha1"
	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
					Port:     9443,
				},
			},
		},
	}
	addOwnerRefToObject(svc, asOwner(cr))
	err := sdk.Create(svc)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("failed to create vault service: %v", err)
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
