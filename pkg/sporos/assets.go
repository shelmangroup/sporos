package sporos

import (
	"fmt"
	"net/url"

	api "github.com/shelmangroup/sporos/pkg/apis/sporos/v1alpha1"
	// log "github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type Asset struct {
	Name string
	Data []byte
}

type Assets []Asset

func (as Assets) Get(name string) (Asset, error) {
	for _, asset := range as {
		if asset.Name == name {
			return asset, nil
		}
	}
	return Asset{}, fmt.Errorf("asset %q does not exist", name)
}

func prepareAssets(cr *api.Sporos) error {
	caKey, caCert, err := newCACert()
	if err != nil {
		return err
	}

	etcdUrl, err := url.Parse(etcdURLForSporos(cr.Name))
	if err != nil {
		return err
	}
	etcdServers := []string{"localhost", etcdUrl.Hostname()}
	etcdAssets, err := newEtcdTLSAssets(nil, nil, nil, caCert, caKey, etcdServers)
	if err != nil {
		return err
	}
	err = createEtcdTLSsecrets(cr, etcdAssets)
	if err != nil {
		return err
	}

	apiServers := []string{"localhost", fmt.Sprintf("%s-kube-api-server.%s.svc", cr.Name, cr.Namespace), cr.Status.ApiServerIP}
	controlplaneAssets, err := newTLSAssets(caCert, caKey, apiServers)
	if err != nil {
		return err
	}
	err = createControlplaneSecrets(cr, append(controlplaneAssets, etcdAssets...))
	if err != nil {
		return err
	}
	return nil
}

func createEtcdTLSsecrets(cr *api.Sporos, a Assets) error {
	//Create server cert secret
	serverKey, _ := a.Get("server.key")
	serverCert, _ := a.Get("server.crt")
	serverCa, _ := a.Get("server-ca.crt")
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
	clientKey, _ := a.Get("etcd-client.key")
	clientCert, _ := a.Get("etc-client.crt")
	clientCa, _ := a.Get("etcd-client-ca.crt")
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
	peerKey, _ := a.Get("peer.key")
	peerCert, _ := a.Get("peer.crt")
	peerCa, _ := a.Get("peer-ca.key")
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

func createControlplaneSecrets(cr *api.Sporos, a Assets) error {
	//Create apiserver cert secret
	caCert, _ := a.Get("ca.crt")
	caKey, _ := a.Get("ca.key")
	apiserverKey, _ := a.Get("apiserver.key")
	apiserverCert, _ := a.Get("apiserver.crt")
	etcdClientKey, _ := a.Get("etcd-client.crt")
	etcdClientCert, _ := a.Get("etcd-client.key")
	etcdClientCa, _ := a.Get("etcd-client-ca.crt")
	serviceAccountPub, _ := a.Get("service-account.pub")
	apiserverSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-kube-apiserver", cr.Name),
			Namespace: cr.Namespace,
			Labels:    LabelsForSporos(cr.Name),
		},
		Data: map[string][]byte{
			"apiserver.key":       apiserverKey.Data,
			"apiserver.crt":       apiserverCert.Data,
			"ca.crt":              caCert.Data,
			"ca.key":              caKey.Data,
			"etcd-client.key":     etcdClientKey.Data,
			"etcd-client.crt":     etcdClientCert.Data,
			"etcd-client-ca.crt":  etcdClientCa.Data,
			"service-account.pub": serviceAccountPub.Data,
		},
	}
	addOwnerRefToObject(apiserverSecret, asOwner(cr))
	err := sdk.Create(apiserverSecret)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	adminKey, _ := a.Get("admin.key")
	adminCert, _ := a.Get("admin.crt")
	adminConfig := clientcmdapi.NewConfig()
	adminConfig.Clusters["local"].Server = fmt.Sprintf("https://%s-kube-api-server.%s.svc", cr.Name, cr.Namespace)
	adminConfig.Clusters["local"].CertificateAuthorityData = caCert.Data
	adminConfig.AuthInfos["admin"].ClientCertificateData = adminCert.Data
	adminConfig.AuthInfos["admin"].ClientKeyData = adminKey.Data
	adminConfig.Contexts["context"].AuthInfo = "admin"
	adminConfig.Contexts["context"].Cluster = "local"
	adminConfigData, err := clientcmd.Write(*adminConfig)
	if err != nil {
		return err
	}
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
			"kubeconfig": adminConfigData,
		},
	}
	addOwnerRefToObject(adminSecret, asOwner(cr))
	err = sdk.Create(adminSecret)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}
