package multus

import (
	"context"
	"encoding/json"

	k8sv1alpha1 "github.com/krsacme/multus-go-operator/pkg/apis/k8s/v1alpha1"
	"github.com/krsacme/multus-go-operator/pkg/util/k8sutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_multus")

// Add creates a new Multus Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMultus{client: mgr.GetClient(), scheme: mgr.GetScheme(), extnCli: k8sutil.MustNewKubeExtClient()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("multus-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		log.Error(err, "failed to create multus controller")
		return err
	}

	// Watch for changes to primary resource Multus
	err = c.Watch(&source.Kind{Type: &k8sv1alpha1.Multus{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		log.Error(err, "failed to watch Multus kind")
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner Multus
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &k8sv1alpha1.Multus{},
	})
	if err != nil {
		log.Error(err, "failed to watch Pod kind with multus as owner")
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileMultus{}

// ReconcileMultus reconciles a Multus object
type ReconcileMultus struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client  client.Client
	scheme  *runtime.Scheme
	extnCli apiextensionsclient.Interface
}

// Reconcile reads that state of the cluster for a Multus object and makes changes based on the state read
// and what is in the Multus.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMultus) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Multus")

	// Fetch the Multus instance
	instance := &k8sv1alpha1.Multus{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	// TODO: validate the number of multus instance, it should be only one, as having multiple version of 'multus-cni' and '.conf' files has no significance

	// TODO: can net-attach-def be moved out of go (directl yaml resource)?
	// TODO: Any advantage on maintainig via operator?
	if err = ensureNetAttachDef(instance, r); err != nil {
		return reconcile.Result{}, err
	}

	if err = ensureConfigMap(instance, r); err != nil {
		return reconcile.Result{}, err
	}

	if err = ensureDaemonSet(instance, r); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func ensureConfigMap(cr *k8sv1alpha1.Multus, r *ReconcileMultus) error {
	cmName := cr.Name + "-cm"
	cmFound := &corev1.ConfigMap{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: cmName, Namespace: cr.Namespace}, cmFound)
	if err != nil && errors.IsNotFound(err) {
		// No ConfigMap found for multus, create a new one
		var cmMultus *corev1.ConfigMap
		if cmMultus, err = newConfigMapForMultusConfig(cr, cmName); err != nil {
			return err
		}
		if err = r.client.Create(context.TODO(), cmMultus); err != nil {
			log.Error(err, "error in creating a new configmap resource")
			return err
		}
	} else if err != nil {
		// Error in getting multus ConfigMap
		log.Error(err, "error in getting configmap")
		return err
	} else {
		// Check for version match to see if any upgrade is required
		log.Info("TODO: configmap exists, check for updates")
	}
	return nil
}

// Ensure Multus DaemonSet is create to copy the cni executable and create cni conf for multus
func ensureDaemonSet(cr *k8sv1alpha1.Multus, r *ReconcileMultus) error {
	dsName := cr.Name + "-ds"
	dsFound := &appsv1.DaemonSet{}
	log.Info("in the daemonset", "Name", cr.Name)
	log.Info("in the daemonset", "Namespace", cr.Namespace)
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: dsName, Namespace: cr.Namespace}, dsFound)
	if err != nil && errors.IsNotFound(err) {
		// No DaemonSet found for multus, create a new one
		log.Info("daemonset is not found, create one")
		dsMultus := newDaemonSetForMultusConfig(cr, dsName)
		if err = r.client.Create(context.TODO(), dsMultus); err != nil {
			log.Error(err, "error in creating a new daemonset resource")
			return err
		}
	} else if err != nil {
		// Error in getting multus DaemonSet
		log.Error(err, "error in getting daemonset")
		return err
	} else {
		// Check for version match to see if an upgrade is required
		log.Info("TODO: daemonset exists, check for updates")
	}
	return nil
}

// Ensure NetworkAttachDefinition object is created and owned by Multus
func ensureNetAttachDef(cr *k8sv1alpha1.Multus, r *ReconcileMultus) error {

	// Define CRD for NetworkDefinitionAttachment
	netDefAttach := newCRDForNetworkAttach(cr)

	// Set Multus instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, netDefAttach, r.scheme); err != nil {
		return err
	}

	_, err := r.extnCli.ApiextensionsV1beta1().CustomResourceDefinitions().Create(netDefAttach)
	if err != nil && !k8sutil.IsKubernetesResourceAlreadyExistError(err) {
		log.Error(err, "failed to create net-attach-def CRD resource")
		return err
	}
	return nil
}

// Create ConfigMap with multus config
func newConfigMapForMultusConfig(cr *k8sv1alpha1.Multus, cmName string) (*corev1.ConfigMap, error) {
	data, err := json.Marshal(cr.Spec.Delegates)
	if err != nil {
		log.Error(err, "failed to marshal multus config to write to configmap resource")
		return nil, err
	}
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: cr.Namespace,
		},
		Data: map[string]string{
			"70-multus-flannel.conf": string(data),
		},
	}, nil
}

// Create new DaemonSet object for the provided multus config
func newDaemonSetForMultusConfig(cr *k8sv1alpha1.Multus, dsName string) *appsv1.DaemonSet {
	directoryOrCreate := corev1.HostPathDirectoryOrCreate
	imageName := cr.Spec.Image + ":" + cr.Spec.Release
	log.Info("newDaemonSetForMultusConfig:", "Image", cr.Spec.Image)
	log.Info("newDaemonSetForMultusConfig:", "Release", cr.Spec.Release)
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: cr.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "multus",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "multus",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "cni-bin",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/opt/cni/bin",
									Type: &directoryOrCreate,
								},
							},
						},
						{
							Name: "cni-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "multus-cni-config",
									},
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:  cr.Name + "-init",
							Image: imageName,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "cni-bin",
									MountPath: "/opt/cni/bin/",
								},
								{
									Name:      "cni-config",
									MountPath: "/etc/multus-cni",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  cr.Name + "-conf-create",
							Image: imageName,
							// TODO: Change the command it later
							Command: []string{
								"/bin/sh",
							},
							Args: []string{
								"-c", "while true; do echo hello; sleep 10;done",
							},
						},
					},
				},
			},
		},
	}
}

// Create new CRD for NetworkAttachDefinition resource
func newCRDForNetworkAttach(cr *k8sv1alpha1.Multus) *apiextensionsv1beta1.CustomResourceDefinition {
	return &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "network-attachment-definitions." + k8sv1alpha1.GroupName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   k8sv1alpha1.GroupName,
			Version: "v1",
			Scope:   "Namespaced",
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural:   "network-attachment-definitions",
				Singular: "network-attachment-definition",
				Kind:     "NetworkAttachmentDefinition",
				ShortNames: []string{
					"net-attach-def",
				},
			},
			Validation: &apiextensionsv1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &apiextensionsv1beta1.JSONSchemaProps{
					Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
						"spec": apiextensionsv1beta1.JSONSchemaProps{
							Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
								"config": apiextensionsv1beta1.JSONSchemaProps{
									Type: "string",
								},
							},
						},
					},
				},
			},
		},
	}
}
