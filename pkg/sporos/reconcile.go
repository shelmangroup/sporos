package sporos

import (
	"fmt"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	api "github.com/shelmangroup/sporos/pkg/apis/sporos/v1alpha1"
	log "github.com/sirupsen/logrus"
)

func Reconcile(cr *api.Sporos) (err error) {
	cr = cr.DeepCopy()

	// After first time reconcile, phase will switch to "Running".
	if cr.Status.Phase == api.ControlplanePhaseInitial {
		if cr.Status.ApiServerIP == "" {
			svc, err := createExternalEndpoint(cr)
			if err != nil {
				return err
			}
			svcReady, err := isServiceEndpointReady(cr, svc)
			if err != nil {
				return fmt.Errorf("failed to check if etcd cluster is ready: %v", err)
			}
			if !svcReady {
				log.Infof("Waiting for service (%v) to become ready", svc.Name)
				return nil
			}
			err = prepareAssets(cr)
			if err != nil {
				return err
			}
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

		deploys, err := deployControlplane(cr)
		if err != nil {
			return err
		}
		for _, d := range deploys {
			ready, err := IsControlplaneReady(d)
			if err != nil {
				return fmt.Errorf("failed to check if %v cluster is ready: %v", d.GetName(), err)
			}
			if !ready {
				log.Infof("Waiting for controlplane (%v) to become ready", d.GetName())
				return nil
			}
		}

		log.Infof("%v is ready!", cr.Name)
		cr.Status.Phase = "Running"
		sdk.Update(cr)
	}

	_, err = NewKubeClient(cr)
	if err != nil {
		return err
	}

	return nil
}
