package sporos

import (
	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	api "github.com/shelmangroup/sporos/pkg/apis/sporos/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
)

func NewKubeClient(cr *api.Sporos) (*kubernetes.Clientset, error) {
	adminSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kubeconfig", cr.Name),
			Namespace: cr.Namespace,
			Labels:    LabelsForSporos(cr.Name),
		},
	}
	err := sdk.Get(adminSecret)
	if err != nil {
		return nil, err
	}

	clientConfig, err := decodeKubeConfigManifest(adminSecret.Data["kubeconfig"])
	if err != nil {
		return nil, fmt.Errorf("decoding of config failed: %v", err)
	}

	kubeConfig, err := clientcmd.NewDefaultClientConfig(*clientConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("New default client config failed: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("New clientset failed: %v", err)
	}

	return clientset, nil
}

func decodeKubeConfigManifest(b []byte) (*clientcmdapi.Config, error) {
	obj, _, err := clientcmdlatest.Codec.Decode(b, &schema.GroupVersionKind{Version: clientcmdlatest.Version, Kind: "Config"}, nil)
	if err != nil {
		return nil, err
	}
	switch o := obj.(type) {
	case *clientcmdapi.Config:
		return o, nil
	default:
		return nil, fmt.Errorf("Object not of type api.Config")
	}
}
