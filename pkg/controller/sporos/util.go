package sporos

import (
	"crypto/rand"

	api "github.com/shelmangroup/sporos/pkg/apis/shelman/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const validBootstrapTokenChars = "0123456789abcdefghijklmnopqrstuvwxyz"

// newBootstrapToken constructs a bootstrap token in conformance with the following format:
// https://kubernetes.io/docs/admin/bootstrap-tokens/#token-format
func newBootstrapToken() (id string, secret string, err error) {
	// Read 6 random bytes for the id and 16 random bytes for the token (see spec for details).
	token := make([]byte, 6+16)
	if _, err := rand.Read(token); err != nil {
		return "", "", err
	}

	for i, b := range token {
		token[i] = validBootstrapTokenChars[int(b)%len(validBootstrapTokenChars)]
	}
	return string(token[:6]), string(token[6:]), nil
}

// addOwnerRefToObject appends the desired OwnerReference to the object
func addOwnerRefToObject(o metav1.Object, r metav1.OwnerReference) {
	o.SetOwnerReferences(append(o.GetOwnerReferences(), r))
}

// LabelsForSporos returns the labels for selecting the resources
// belonging to the given spors name.
func LabelsForSporos(name string) map[string]string {
	return map[string]string{"app": "sporos", "controlplane": name}
}

// asOwner returns an owner reference set as the sporos cluster CR
func asOwner(v *api.Sporos) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: api.SchemeGroupVersion.String(),
		Kind:       "Sporos",
		Name:       v.Name,
		UID:        v.UID,
		Controller: &trueVar,
	}
}
