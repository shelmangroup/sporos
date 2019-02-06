package sporos

import (
	"context"
	"fmt"
	"time"

	eopapi "github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	api "github.com/shelmangroup/sporos/pkg/apis/shelman/v1alpha1"
	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	size                  = 1
	minutesBetweenBackups = 5
)

// deployEtcdCluster creates an etcd cluster for the given sporos's name via etcd operator.
func (r *ReconcileSporos) deployEtcdCluster(cr *api.Sporos) (*eopapi.EtcdCluster, error) {
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
	err := r.client.Create(context.TODO(), ec)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return ec, nil
		}
		return nil, fmt.Errorf("deploy etcd cluster failed: %v", err)
	}
	return ec, nil
}

func (r *ReconcileSporos) backupEtcdCluster(cr *api.Sporos) (*eopapi.EtcdBackup, error) {
	bup := &eopapi.EtcdBackup{
		TypeMeta: metav1.TypeMeta{
			Kind:       eopapi.EtcdBackupResourceKind,
			APIVersion: eopapi.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            EtcdNameForSporos(cr.Name),
			Namespace:       cr.Namespace,
			Labels:          LabelsForSporos(cr.Name),
			ResourceVersion: "",
		},
		Spec: eopapi.BackupSpec{
			EtcdEndpoints:   []string{etcdURLForSporos(cr.Name)},
			ClientTLSSecret: EtcdClientTLSSecretName(cr.Name),
			StorageType:     eopapi.BackupStorageTypeS3,
			BackupSource: eopapi.BackupSource{
				S3: &eopapi.S3BackupSource{
					Path:      fmt.Sprintf("etcd-backups/%s/%v", cr.Name, time.Now().Unix()),
					AWSSecret: "sporos-aws",
					Endpoint:  "http://sporos-minio:9000",
				},
			},
		},
	}
	ready, err := r.timeToBackup(bup)
	if err != nil {
		return nil, fmt.Errorf("backup timer failed: %v", err)
	}
	if !ready {
		return bup, nil
	}
	err = r.client.Create(context.TODO(), bup)
	if err != nil {
		return nil, fmt.Errorf("backup etcd cluster failed: %v", err)
	}
	log.Info("Creating new etcd backup for %v", cr.Name)
	return bup, nil
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

func (r *ReconcileSporos) isEtcdClusterReady(ec *eopapi.EtcdCluster) (bool, error) {
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: ec.GetName(), Namespace: ec.GetNamespace()}, ec)
	if err != nil {
		return false, err
	}
	return (len(ec.Status.Members.Ready) == size), nil
}

func (r *ReconcileSporos) isEtcdClusterBackupReady(bup *eopapi.EtcdBackup) (bool, error) {
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: bup.GetName(), Namespace: bup.GetNamespace()}, bup)
	if err != nil {
		return false, err
	}
	return bup.Status.Succeeded, nil
}

func (r *ReconcileSporos) timeToBackup(bup *eopapi.EtcdBackup) (bool, error) {
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: bup.GetName(), Namespace: bup.GetNamespace()}, bup)
	if err != nil {
		return true, nil
	}

	if time.Since(bup.CreationTimestamp.Local()) < (minutesBetweenBackups * time.Minute) {
		return false, nil
	}

	err = r.client.Delete(context.TODO(), bup)
	if err != nil {
		return false, err
	}

	return true, nil
}
