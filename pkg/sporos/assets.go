package sporos

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/kubernetes-incubator/bootkube/pkg/asset"
	"github.com/kubernetes-incubator/bootkube/pkg/tlsutil"
	api "github.com/shelmangroup/sporos/pkg/apis/sporos/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func prepareAssets(cr *api.Sporos) error {
	decode := scheme.Codecs.UniversalDeserializer().Decode

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
	assets, err := asset.NewDefaultAssets(conf)
	if err != nil {
		return err
	}
	for _, a := range assets {
		if strings.HasPrefix(a.Name, "manifests") {
			obj, _, err := decode(a.Data, nil, nil)
			if err != nil {
				return fmt.Errorf("Error while decoding YAML object. Err was: %s", err)
			}
			switch o := obj.(type) {
			case *corev1.Secret:
				fmt.Println(o.Data)
			default:
				continue
			}
		}
	}
	return nil
}
