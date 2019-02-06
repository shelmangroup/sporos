package sporos

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (r *ReconcileSporos) csrBootstrap(client *kubernetes.Clientset) error {
	nodeBootstrapperSubjects := []rbacv1.Subject{
		rbacv1.Subject{
			Kind:     "Group",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "system:bootstrappers",
		},
		rbacv1.Subject{
			Kind:     "Group",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "system:nodes",
		},
	}
	approveNodeSubjects := []rbacv1.Subject{
		rbacv1.Subject{
			Kind:     "Group",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "system:bootstrappers",
		},
	}
	nodeRenewalSubjects := []rbacv1.Subject{
		rbacv1.Subject{
			Kind:     "Group",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     "system:nodes",
		},
	}
	_, err := client.RbacV1().ClusterRoleBindings().Create(
		CreateClusterRoleBinding("system-bootstrap-node-bootstrapper", nodeBootstrapperSubjects, "system:node-bootstrapper"),
	)
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoleBindings().Create(
		CreateClusterRoleBinding("system-bootstrap-approve-node-client-csr", approveNodeSubjects, "system:certificates.k8s.io:certificatesigningrequests:nodeclient"),
	)
	if err != nil {
		return err
	}

	_, err = client.RbacV1().ClusterRoleBindings().Create(
		CreateClusterRoleBinding("system-bootstrap-node-renewal", nodeRenewalSubjects, "system:certificates.k8s.io:certificatesigningrequests:selfnodeclient"),
	)
	if err != nil {
		return err
	}

	return nil
}

func CreateClusterRoleBinding(name string, subjects []rbacv1.Subject, roleRefName string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Subjects: subjects,
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			APIGroup: "rbac.authorization.k8s.io",
			Name:     roleRefName,
		},
	}
}
