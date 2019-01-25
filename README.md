# Operator Implementation for Multus CNI Deployment

> Note: It is an experimental repo to learn operator and multus

> Still under making...
## What is the role of the Multus Operator?
* Creates ``multus`` binary in ``/opt/cni/bin/`` director of all nodes
* Uses multus-cni release version (currently v3.1 is used)
* Creates config map to add provided multus config to ``/etc/cni/net.d/`` directory

## Deploy Multus Operator

```bash
git clone https://github.com/krsacme/multus-go-operator.git
cd multus-go-operator

# Prepare configs to create operator
kubectl create -f deploy/service_account.yaml
kubectl create -f deploy/rbac
kubectl create -f deploy/crds/k8s_v1alpha1_multus_crd.yaml

# Create a deployment for the operator
kubectl create -f deploy/operator.yaml

# Create a daemonset for the operator
kubectl create -f deploy/ds.yaml

# TODO: flannel daemonset creation is not included yet
# For now use below resource file to create it, which will be added to the multus operator
kubectl create -f https://raw.githubusercontent.com/intel/multus-cni/master/images/flannel-daemonset.yml

# Create 'multus' resource to configure the Multus CNI with flannel
kubectl create -f deploy/crds/k8s_v1alpha1_multus_cr_flannel_sriov.yaml

```
