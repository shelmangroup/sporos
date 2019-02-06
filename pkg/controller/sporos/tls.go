package sporos

import (
	"crypto/rsa"
	"crypto/x509"
	"net"

	"github.com/pborman/uuid"

	"github.com/shelmangroup/sporos/pkg/tlsutil"
)

func newTLSAssets(caCert *x509.Certificate, caPrivKey *rsa.PrivateKey, altNames []string) ([]Asset, error) {
	var (
		assets []Asset
		err    error
	)

	apiKey, apiCert, err := newAPIKeyAndCert(caCert, caPrivKey, altNames)
	if err != nil {
		return assets, err
	}

	saPrivKey, err := tlsutil.NewPrivateKey()
	if err != nil {
		return assets, err
	}

	saPubKey, err := tlsutil.EncodePublicKeyPEM(&saPrivKey.PublicKey)
	if err != nil {
		return assets, err
	}

	adminKey, adminCert, err := newAdminKeyAndCert(caCert, caPrivKey)
	if err != nil {
		return assets, err
	}

	assets = append(assets, []Asset{
		{Name: "ca.key", Data: tlsutil.EncodePrivateKeyPEM(caPrivKey)},
		{Name: "ca.crt", Data: tlsutil.EncodeCertificatePEM(caCert)},
		{Name: "apiserver.key", Data: tlsutil.EncodePrivateKeyPEM(apiKey)},
		{Name: "apiserver.crt", Data: tlsutil.EncodeCertificatePEM(apiCert)},
		{Name: "service-account.key", Data: tlsutil.EncodePrivateKeyPEM(saPrivKey)},
		{Name: "service-account.pub", Data: saPubKey},
		{Name: "admin.key", Data: tlsutil.EncodePrivateKeyPEM(adminKey)},
		{Name: "admin.crt", Data: tlsutil.EncodeCertificatePEM(adminCert)},
	}...)
	return assets, nil
}

func newCACert() (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := tlsutil.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	config := tlsutil.CertConfig{
		CommonName:         "kube-ca",
		Organization:       []string{uuid.New()},
		OrganizationalUnit: []string{"sporos"},
	}

	cert, err := tlsutil.NewSelfSignedCACertificate(config, key)
	if err != nil {
		return nil, nil, err
	}

	return key, cert, err
}

func newAPIKeyAndCert(caCert *x509.Certificate, caPrivKey *rsa.PrivateKey, addrs []string) (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := tlsutil.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}
	var altNames tlsutil.AltNames
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil {
			altNames.IPs = append(altNames.IPs, ip)
		} else {
			altNames.DNSNames = append(altNames.DNSNames, addr)
		}
	}
	altNames.DNSNames = append(altNames.DNSNames, []string{
		"kubernetes",
		"kubernetes.default",
		"kubernetes.default.svc",
		"kubernetes.default.svc.cluster.local",
	}...)

	config := tlsutil.CertConfig{
		CommonName:   "kube-apiserver",
		Organization: []string{"kube-master"},
		AltNames:     altNames,
	}
	cert, err := tlsutil.NewSignedCertificate(config, key, caCert, caPrivKey)
	if err != nil {
		return nil, nil, err
	}
	return key, cert, err
}

func newAdminKeyAndCert(caCert *x509.Certificate, caPrivKey *rsa.PrivateKey) (*rsa.PrivateKey, *x509.Certificate, error) {
	// TLS organizations map to Kubernetes groups, and "system:masters"
	// is a well-known Kubernetes group that gives a user admin power.
	const orgSystemMasters = "system:masters"

	key, err := tlsutil.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}
	config := tlsutil.CertConfig{
		CommonName:   "admin",
		Organization: []string{orgSystemMasters},
	}
	cert, err := tlsutil.NewSignedCertificate(config, key, caCert, caPrivKey)
	if err != nil {
		return nil, nil, err
	}
	return key, cert, err
}

func newEtcdTLSAssets(etcdCACert, etcdClientCert *x509.Certificate, etcdClientKey *rsa.PrivateKey, caCert *x509.Certificate, caPrivKey *rsa.PrivateKey, etcdServers []string) ([]Asset, error) {
	var assets []Asset
	if etcdCACert == nil {
		// Use the master CA to generate etcd assets.
		etcdCACert = caCert

		// Create an etcd client cert.
		var err error
		etcdClientKey, etcdClientCert, err = newKeyAndCert(caCert, caPrivKey, "etcd-client", etcdServers)
		if err != nil {
			return nil, err
		}

		// Create an etcd peer cert (not consumed by self-hosted components).
		etcdPeerKey, etcdPeerCert, err := newKeyAndCert(caCert, caPrivKey, "etcd-peer", etcdServers)
		if err != nil {
			return nil, err
		}
		etcdServerKey, etcdServerCert, err := newKeyAndCert(caCert, caPrivKey, "etcd-server", etcdServers)
		if err != nil {
			return nil, err
		}

		assets = append(assets, []Asset{
			{Name: "peer-ca.crt", Data: tlsutil.EncodeCertificatePEM(etcdCACert)},
			{Name: "peer.key", Data: tlsutil.EncodePrivateKeyPEM(etcdPeerKey)},
			{Name: "peer.crt", Data: tlsutil.EncodeCertificatePEM(etcdPeerCert)},
			{Name: "server-ca.crt", Data: tlsutil.EncodeCertificatePEM(etcdCACert)},
			{Name: "server.key", Data: tlsutil.EncodePrivateKeyPEM(etcdServerKey)},
			{Name: "server.crt", Data: tlsutil.EncodeCertificatePEM(etcdServerCert)},
		}...)
	}

	assets = append(assets, []Asset{
		{Name: "etcd-client-ca.crt", Data: tlsutil.EncodeCertificatePEM(etcdCACert)},
		{Name: "etcd-client.key", Data: tlsutil.EncodePrivateKeyPEM(etcdClientKey)},
		{Name: "etcd-client.crt", Data: tlsutil.EncodeCertificatePEM(etcdClientCert)},
	}...)

	return assets, nil
}

func newKeyAndCert(caCert *x509.Certificate, caPrivKey *rsa.PrivateKey, commonName string, addrs []string) (*rsa.PrivateKey, *x509.Certificate, error) {
	key, err := tlsutil.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}
	var altNames tlsutil.AltNames
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil {
			altNames.IPs = append(altNames.IPs, ip)
		} else {
			altNames.DNSNames = append(altNames.DNSNames, addr)
		}
	}
	config := tlsutil.CertConfig{
		CommonName:   commonName,
		Organization: []string{"etcd"},
		AltNames:     altNames,
	}
	cert, err := tlsutil.NewSignedCertificate(config, key, caCert, caPrivKey)
	if err != nil {
		return nil, nil, err
	}
	return key, cert, err
}
