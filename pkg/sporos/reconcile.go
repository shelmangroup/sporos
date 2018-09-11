package sporos

import (
	"fmt"
	api "github.com/shelmangroup/sporos/pkg/apis/sporos/v1alpha1"
	log "github.com/sirupsen/logrus"
)

func Reconcile(cr *api.Sporos) (err error) {
	cr = cr.DeepCopy()

	// After first time reconcile, phase will switch to "Running".
	if cr.Status.Phase == api.ControlplanePhaseInitial {
		err = prepareAssets(cr)
		if err != nil {
			return err
		}
		// etcd cluster should only be created in first time reconcile.
		ec, err := deployEtcdCluster(cr)
		if err != nil {
			return err
		}
		// Check if etcd cluster is up and running.
		// If not, we need to wait until etcd cluster is up before proceeding to the next state;
		// Hence, we return from here and let the Watch triggers the handler again.
		ready, err := isEtcdClusterReady(ec)
		if err != nil {
			return fmt.Errorf("failed to check if etcd cluster is ready: %v", err)
		}
		if !ready {
			log.Infof("Waiting for EtcdCluster (%v) to become ready", ec.Name)
			return nil
		}
	}

	return nil
}
