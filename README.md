# Sporos

## Installation
```
$ kubectl create ns sporos
$ helm install --namespace sporos -n sporos-etcd-operator stable/etcd-operator
$ kubectl create -n sporos -f deploy/crd.yaml
$ kubectl create -n sporos -f deploy/rbac.yaml
$ kubectl create -n sporos -f deploy/operator.yaml
```

## Create new control plane
create a custom resource like so:
```yaml
apiVersion: sporos.shelman.io/v1alpha1
kind: Sporos
metadata:
  name: helloworld
spec:
  baseImage: k8s.gcr.io/hyperkube
  version: v1.11.3
  podCIDR: "10.200.0.0/16"
  serviceCIDR: "10.32.0.0/24"
```

Create the new control plane:
```
$ kubectl create -n sporos -f cr.yaml
```

watch logs for creation progress
```
$ kubectl logs -n infra sporos-867b6fb498-26ssr -f

time="2018-10-02T08:53:47Z" level=info msg="Go Version: go1.11"
time="2018-10-02T08:53:47Z" level=info msg="Go OS/Arch: linux/amd64"
time="2018-10-02T08:53:47Z" level=info msg="operator-sdk Version: 0.0.6+git"
time="2018-10-02T08:53:47Z" level=info msg="Metrics service sporos created"
time="2018-10-02T08:53:47Z" level=info msg="Watching sporos.shelman.io/v1alpha1, Sporos, infra, 5000000000"
time="2018-10-02T08:54:36Z" level=info msg="Waiting for service (helloworld-kube-apiserver) to become ready"
time="2018-10-02T08:54:37Z" level=info msg="Waiting for service (helloworld-kube-apiserver) to become ready"
time="2018-10-02T08:54:42Z" level=info msg="Waiting for service (helloworld-kube-apiserver) to become ready"
time="2018-10-02T08:54:47Z" level=info msg="Waiting for service (helloworld-kube-apiserver) to become ready"
time="2018-10-02T08:54:52Z" level=info msg="Waiting for service (helloworld-kube-apiserver) to become ready"
time="2018-10-02T08:54:57Z" level=info msg="Waiting for service (helloworld-kube-apiserver) to become ready"
time="2018-10-02T08:55:02Z" level=info msg="Waiting for service (helloworld-kube-apiserver) to become ready"
time="2018-10-02T08:55:07Z" level=info msg="Waiting for service (helloworld-kube-apiserver) to become ready"
time="2018-10-02T08:55:12Z" level=info msg="Waiting for service (helloworld-kube-apiserver) to become ready"
time="2018-10-02T08:55:17Z" level=info msg="Waiting for service (helloworld-kube-apiserver) to become ready"
time="2018-10-02T08:55:22Z" level=info msg="Waiting for service (helloworld-kube-apiserver) to become ready"
time="2018-10-02T08:55:27Z" level=info msg="Waiting for service (helloworld-kube-apiserver) to become ready"
time="2018-10-02T08:55:32Z" level=info msg="API server endpoint: https://35.123.22.211"
time="2018-10-02T08:55:34Z" level=info msg="Waiting for EtcdCluster (helloworld-etcd) to become ready"
time="2018-10-02T08:55:34Z" level=info msg="Waiting for EtcdCluster (helloworld-etcd) to become ready"
time="2018-10-02T08:55:37Z" level=info msg="Waiting for EtcdCluster (helloworld-etcd) to become ready"
time="2018-10-02T08:55:42Z" level=info msg="helloworld is ready!"
```

Get admin kubeconfig
```
$ export API_ENDPOINT=35.123.22.211
$ kubectl get secrets -n sporos helloworld-kubeconfig -o json | jq -r '.data.kubeconfig' | base64 -d | sed -i s#helloworld-kube-apiserver.infra.svc#${API_ENDPOINT}# > kubecfg
```

Try it out.
```
$ KUBECONFIG=./kubecfg kubectl get pods --all-namespaces
NAMESPACE     NAME                       READY   STATUS    RESTARTS   AGE
kube-system   coredns-7c6fbb4f4b-f7vlr   1/1     Running   0          1h
kube-system   kube-router-rdnx5          1/1     Running   0          1h
kube-system   kube-router-wgchk          1/1     Running   0          1h
```

