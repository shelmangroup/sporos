package sporos

import (
	api "github.com/shelmangroup/sporos/pkg/apis/sporos/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
