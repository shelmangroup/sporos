package sporos

import (
	"fmt"

	eopapi "github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	api "github.com/shelmangroup/sporos/pkg/apis/sporos/v1alpha1"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	size = 1
)

// deployEtcdCluster creates an etcd cluster for the given sporos's name via etcd operator.
func deployEtcdCluster(cr *api.Sporos) (*eopapi.EtcdCluster, error) {
	ec := &eopapi.EtcdCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       eopapi.EtcdClusterResourceKind,
			APIVersion: eopapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdNameForSporos(cr.Name),
			Namespace: cr.Namespace,
			Labels:    LabelsForSporos(cr.Name),
		},
		Spec: eopapi.ClusterSpec{
			Size: size,
			TLS: &eopapi.TLSPolicy{
				Static: &eopapi.StaticTLS{
					Member: &eopapi.MemberSecret{
						PeerSecret:   EtcdPeerTLSSecretName(cr.Name),
						ServerSecret: EtcdServerTLSSecretName(cr.Name),
					},
					OperatorSecret: EtcdClientTLSSecretName(cr.Name),
				},
			},
			Pod: &eopapi.PodPolicy{
				EtcdEnv: []v1.EnvVar{{
					Name:  "ETCD_AUTO_COMPACTION_RETENTION",
					Value: "1",
				}},
			},
		},
	}
	if cr.Spec.Pod != nil {
		ec.Spec.Pod.Resources = cr.Spec.Pod.Resources
	}
	addOwnerRefToObject(ec, asOwner(cr))
	err := sdk.Create(ec)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return ec, nil
		}
		return nil, fmt.Errorf("deploy etcd cluster failed: %v", err)
	}
	return ec, nil
}

// EtcdNameForSporos returns the etcd cluster's name for the given sporos's name
func EtcdNameForSporos(name string) string {
	return name + "-etcd"
}

func EtcdClientTLSSecretName(name string) string {
	return name + "-etcd-client-tls"
}

// EtcdServerTLSSecretName returns the name of etcd server TLS secret for the given sporos name
func EtcdServerTLSSecretName(name string) string {
	return name + "-etcd-server-tls"
}

// EtcdPeerTLSSecretName returns the name of etcd peer TLS secret for the given sporos name
func EtcdPeerTLSSecretName(name string) string {
	return name + "-etcd-peer-tls"
}

// etcdURLForSporos returns the URL to talk to etcd cluster for the given
// sporos's name
func etcdURLForSporos(name string) string {
	return fmt.Sprintf("https://%s-client:2379", EtcdNameForSporos(name))
}

func isEtcdClusterReady(ec *eopapi.EtcdCluster) (bool, error) {
	err := sdk.Get(ec)
	if err != nil {
		return false, err
	}
	return (len(ec.Status.Members.Ready) == size), nil
}
