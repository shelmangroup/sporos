package sporos

import (
	"net"
	"net/url"

	"github.com/kubernetes-incubator/bootkube/pkg/asset"
	"github.com/kubernetes-incubator/bootkube/pkg/tlsutil"
	"github.com/shelmangroup/sporos/pkg/apis/sporos/v1alpha1"
	log "github.com/sirupsen/logrus"
)

func Reconcile(cr *v1alpha1.Sporos) (err error) {
	cr = cr.DeepCopy()

	// After first time reconcile, phase will switch to "Running".
	if cr.Status.Phase == v1alpha1.ClusterPhaseInitial {
		err = prepareAssets(cr)
		if err != nil {
			return err
		}
		// // etcd cluster should only be created in first time reconcile.
		// ec, err := deployEtcdCluster(vr)
		// if err != nil {
		// 	return err
		// }
		// // Check if etcd cluster is up and running.
		// // If not, we need to wait until etcd cluster is up before proceeding to the next state;
		// // Hence, we return from here and let the Watch triggers the handler again.
		// ready, err := isEtcdClusterReady(ec)
		// if err != nil {
		// 	return fmt.Errorf("failed to check if etcd cluster is ready: %v", err)
		// }
		// if !ready {
		// 	log.Infof("Waiting for EtcdCluster (%v) to become ready", ec.Name)
		// 	return nil
		// }
		log.Infof("Waiting for EtcdCluster to become ready")
	}

	// err = prepareDefaultVaultTLSSecrets(vr)
	// if err != nil {
	// 	return err
	// }
	return nil
}

func prepareAssets(cr *v1alpha1.Sporos) error {
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
