package multus

import (
	k8sv1alpha1 "github.com/krsacme/multus-go-operator/pkg/apis/k8s/v1alpha1"

	corev1 "k8s.io/api/core/v1"
)

type SriovExtension struct {
}

func (e *SriovExtension) Type() k8sv1alpha1.ExtensionType {
	return k8sv1alpha1.ExtensionTypeSriov
}

func configureSriovExtension(cr *k8sv1alpha1.Multus, r *ReconcileMultus) error {
	//sriovExt := cr.Spec.Extensions[k8sv1alpha1.ExtensionTypeSriov]
	return nil
}

func getSriovInitContainers(cr *k8sv1alpha1.Multus) corev1.Container {
	return corev1.Container{
		Name: cr.Name + "-cni",
		// TODO: get it from the cr
		Image:           "nfvpe/sriov-cni:latest",
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts:    getVolumeMounts(),
		Command:         []string{"sh", "-c", "rm -f /opt/cni/bin/sriov; cp -f /usr/src/sriov-cni/bin/sriov /opt/cni/bin/"},
	}
}
