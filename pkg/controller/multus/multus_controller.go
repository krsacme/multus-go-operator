package multus

import (
	"context"

	k8sv1alpha1 "github.com/krsacme/multus-go-operator/pkg/apis/k8s/v1alpha1"
	"github.com/krsacme/multus-go-operator/pkg/util/k8sutil"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

	// Define CRD for NetworkDefinitionAttachment
	netDefAttach := newCRDForNetworkAttach(instance)

	// Set Multus instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, netDefAttach, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	_, err = r.extnCli.ApiextensionsV1beta1().CustomResourceDefinitions().Create(netDefAttach)
	if err != nil && !k8sutil.IsKubernetesResourceAlreadyExistError(err) {
		log.Error(err, "failed to create net-attach-def CRD resource")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

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
