package sporos

import (
	"fmt"
	"net/http"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	api "github.com/shelmangroup/sporos/pkg/apis/sporos/v1alpha1"
	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
)

var timeout = 5 * time.Minute

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

	upFn := func() (bool, error) {
		if err := apiTest(clientset); err != nil {
			log.Warnf("Unable to determine api-server readiness: %v", err)
			return false, nil
		}
		return true, nil
	}

	if err := wait.Poll(5*time.Second, timeout, upFn); err != nil {
		return nil, fmt.Errorf("API Server is not ready: %v", err)
	}

	return clientset, nil
}

func apiTest(client *kubernetes.Clientset) error {
	// API Server is responding
	healthStatus := 0
	client.Discovery().RESTClient().Get().AbsPath("/healthz").Do().StatusCode(&healthStatus)
	if healthStatus != http.StatusOK {
		return fmt.Errorf("API Server http status: %d", healthStatus)
	}

	// System namespace has been created
	_, err := client.CoreV1().Namespaces().Get("kube-system", metav1.GetOptions{})
	return err
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
