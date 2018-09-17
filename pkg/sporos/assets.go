package sporos

import (
	"fmt"
	"net"
	"net/url"

	"github.com/kubernetes-incubator/bootkube/pkg/asset"
	"github.com/kubernetes-incubator/bootkube/pkg/tlsutil"
	api "github.com/shelmangroup/sporos/pkg/apis/sporos/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func prepareAssets(cr *api.Sporos) error {
	// decode := scheme.Codecs.UniversalDeserializer().Decode

	apiserver, err := url.Parse(cr.Spec.ApiServerUrl)
	if err != nil {
		return err
	}
	_, podCIDR, err := net.ParseCIDR(cr.Spec.PodCIDR)
	if err != nil {
		return err
	}
	_, svcCIDR, err := net.ParseCIDR(cr.Spec.ServiceCIDR)
	if err != nil {
		return err
	}

	conf := asset.Config{
		EtcdServers: []*url.URL{apiserver},
		EtcdUseTLS:  true,
		APIServers:  []*url.URL{apiserver},
		AltNames: &tlsutil.AltNames{
			DNSNames: []string{
				"localhost",
				fmt.Sprintf("*.%s.svc.cluster.local", cr.Namespace),
			},
			IPs: []net.IP{
				net.ParseIP("127.0.0.1"),
			},
		},
		PodCIDR:      podCIDR,
		ServiceCIDR:  svcCIDR,
		APIServiceIP: net.ParseIP(cr.Spec.ApiServerIP),
		DNSServiceIP: net.ParseIP(cr.Spec.ApiServerIP),
		Images:       asset.DefaultImages,
	}
	assets, err := asset.NewDefaultAssets(conf)
	if err != nil {
		return err
	}

	err = createEtcdTLSsecrets(cr, assets)
	if err != nil {
		return err
	}
	err = createControlplaneSecrets(cr, assets)
	if err != nil {
		return err
	}
	return nil
}

func createEtcdTLSsecrets(cr *api.Sporos, a asset.Assets) error {
	//Create server cert secret
	serverKey, _ := a.Get(asset.AssetPathEtcdServerKey)
	serverCert, _ := a.Get(asset.AssetPathEtcdServerCert)
	serverCa, _ := a.Get(asset.AssetPathEtcdServerCA)
	serverSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdServerTLSSecretName(cr.Name),
			Namespace: cr.Namespace,
			Labels:    LabelsForSporos(cr.Name),
		},
		Data: map[string][]byte{
			"server.key":    serverKey.Data,
			"server.crt":    serverCert.Data,
			"server-ca.crt": serverCa.Data,
		},
	}
	addOwnerRefToObject(serverSecret, asOwner(cr))
	err := sdk.Create(serverSecret)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	//Create client cert secret
	clientKey, _ := a.Get(asset.AssetPathEtcdClientKey)
	clientCert, _ := a.Get(asset.AssetPathEtcdClientCert)
	clientCa, _ := a.Get(asset.AssetPathEtcdClientCA)
	clientSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdClientTLSSecretName(cr.Name),
			Namespace: cr.Namespace,
			Labels:    LabelsForSporos(cr.Name),
		},
		Data: map[string][]byte{
			"etcd-client.key":    clientKey.Data,
			"etcd-client.crt":    clientCert.Data,
			"etcd-client-ca.crt": clientCa.Data,
		},
	}
	addOwnerRefToObject(clientSecret, asOwner(cr))
	err = sdk.Create(clientSecret)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	//Create peer cert secret
	peerKey, _ := a.Get(asset.AssetPathEtcdPeerKey)
	peerCert, _ := a.Get(asset.AssetPathEtcdPeerCert)
	peerCa, _ := a.Get(asset.AssetPathEtcdPeerCA)
	peerSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdPeerTLSSecretName(cr.Name),
			Namespace: cr.Namespace,
			Labels:    LabelsForSporos(cr.Name),
		},
		Data: map[string][]byte{
			"peer.key":    peerKey.Data,
			"peer.crt":    peerCert.Data,
			"peer-ca.crt": peerCa.Data,
		},
	}
	addOwnerRefToObject(peerSecret, asOwner(cr))
	err = sdk.Create(peerSecret)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func createControlplaneSecrets(cr *api.Sporos, a asset.Assets) error {
	kubeApiserverSecret, _ := a.Get(asset.AssetPathAPIServerSecret)
	apiSecret, err := decodeSecretManifest(kubeApiserverSecret.Data)
	if err != nil {
		return err
	}
	apiSecret.ObjectMeta.Namespace = cr.Namespace
	apiSecret.ObjectMeta.Name = fmt.Sprintf("%s-kube-apiserver", cr.Name)
	apiSecret.ObjectMeta.Labels = LabelsForSporos(cr.Name)

	addOwnerRefToObject(apiSecret, asOwner(cr))
	err = sdk.Create(apiSecret)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	kubeControllerSecret, _ := a.Get(asset.AssetPathControllerManagerSecret)
	controllerSecret, err := decodeSecretManifest(kubeControllerSecret.Data)
	if err != nil {
		return err
	}
	controllerSecret.ObjectMeta.Namespace = cr.Namespace
	controllerSecret.ObjectMeta.Name = fmt.Sprintf("%s-kube-controller-manager", cr.Name)
	controllerSecret.ObjectMeta.Labels = LabelsForSporos(cr.Name)

	addOwnerRefToObject(controllerSecret, asOwner(cr))
	err = sdk.Create(controllerSecret)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	adminKubeconfig, _ := a.Get(asset.AssetPathAdminKubeConfig)
	adminSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-admin-kubeconfig", cr.Name),
			Namespace: cr.Namespace,
			Labels:    LabelsForSporos(cr.Name),
		},
		Data: map[string][]byte{
			"kubeconfig": adminKubeconfig.Data,
		},
	}
	addOwnerRefToObject(adminSecret, asOwner(cr))
	err = sdk.Create(adminSecret)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func decodeSecretManifest(b []byte) (*corev1.Secret, error) {
	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(b, nil, nil)
	if err != nil {
		return nil, err
	}
	var s *corev1.Secret
	switch o := obj.(type) {
	case *corev1.Secret:
		s = o
	default:
		return nil, fmt.Errorf("No secret")
	}
	return s, nil
}
