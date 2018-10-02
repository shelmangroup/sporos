# Sporos

## Installation
```
$ helm install --namespace <ns> -n sporos-etcd-operator stable/etcd-operator
$ kubectl create -n <ns> -f deploy/crd.yaml
$ kubectl create -n <ns> -f deploy/rbac.yaml
$ kubectl create -n <ns> -f deploy/operator.yaml
```

## Create new control plane
create a custom resource like so:
```yaml
apiVersion: "sporos.shelman.io/v1alpha1"
kind: "Sporos"
metadata:
  name: "helloworld"
spec:
  baseImage: k8s.gcr.io/hyperkube
  version: v1.11.3
  podCIDR: "10.200.0.0/16"
  serviceCIDR: "10.32.0.0/24"
```

Create the new control plane:
```
$ kubectl create -n <ns> -f cr.yaml
```
