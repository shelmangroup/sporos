package sporos

import (
	"net"
	"net/url"

	"github.com/kubernetes-incubator/bootkube/pkg/asset"
	"github.com/kubernetes-incubator/bootkube/pkg/tlsutil"
	api "github.com/shelmangroup/sporos/pkg/apis/sporos/v1alpha1"
)

func prepareAssets(cr *api.Sporos) error {
	apiserver, _ := url.Parse(cr.Spec.ApiServerUrl)
	_, podCIDR, _ := net.ParseCIDR(cr.Spec.PodCIDR)
	_, svcCIDR, _ := net.ParseCIDR(cr.Spec.ServiceCIDR)

	conf := asset.Config{
		EtcdServers:  []*url.URL{apiserver},
		EtcdUseTLS:   true,
		APIServers:   []*url.URL{apiserver},
		AltNames:     &tlsutil.AltNames{},
		PodCIDR:      podCIDR,
		ServiceCIDR:  svcCIDR,
		APIServiceIP: net.ParseIP(cr.Spec.ApiServerIP),
		DNSServiceIP: net.ParseIP(cr.Spec.ApiServerIP),
		Images:       asset.DefaultImages,
	}
	_, err := asset.NewDefaultAssets(conf)
	if err != nil {
		return err
	}
	return nil
}
