package message

import (
	"context"
	"fmt"
	"time"

	pohsienshihv1 "github.com/pohsienshih/chatbot-operator/chatbot-operator/pkg/apis/pohsienshih/v1"
	etcdclient "go.etcd.io/etcd/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_message")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Message Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMessage{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("message-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Message
	err = c.Watch(&source.Kind{Type: &pohsienshihv1.Message{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Message
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &pohsienshihv1.Message{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileMessage implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileMessage{}

// ReconcileMessage reconciles a Message object
type ReconcileMessage struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Message object and makes changes based on the state read
// and what is in the Message.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMessage) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Message")

	// Fetch the Message instance
	instance := &pohsienshihv1.Message{}
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

	//-------------------------------------------------------------------

	// Get the etcd Service by using bot name
	for i := 0; i < len(instance.Spec.Botname); i++ {
		foundDBService := &corev1.Service{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Spec.Botname[i] + "-etcd", Namespace: instance.Namespace}, foundDBService)
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Can not found the DB Service by using Bot name.")
			continue
			// return reconcile.Result{}, err
		}

		// Do not use the Service DNS name to instead the etcd IP here. The controller cannot resolve
		// the Service DNS name because it doesn't know which Namespace it belongs to.
		err = storeData(instance, foundDBService.Spec.ClusterIP)
		if err != nil {
			reqLogger.Info("Can not write value into etcd.")
			fmt.Println(err)
		}
	}
	reqLogger.Info("Write value into etcd with specific botname complete")

	// Get the etcd services by using group
	// https://github.com/operator-framework/operator-sdk/blob/master/doc/user/client.md
	for i := 0; i < len(instance.Spec.Group); i++ {
		lab := map[string]string{
			"group": instance.Spec.Group[i],
		}

		foundDBServices := &corev1.ServiceList{}
		// Get all database services in specific namespace and group
		err = r.client.List(context.TODO(), foundDBServices, &client.ListOptions{Namespace: instance.Namespace, LabelSelector: labels.Set(lab).AsSelector()})

		if err != nil {
			reqLogger.Info("Can not found the DB Service by using Bot group.")
			continue
			// return reconcile.Result{}, err
		}
		for _, item := range foundDBServices.Items {

			// Do not use the Service DNS name to instead the etcd IP here. The controller cannot resolve
			// the Service DNS name because it doesn't know which Namespace it belongs to.
			err = storeData(instance, item.Spec.ClusterIP)
			if err != nil {
				reqLogger.Info("Can not write a value into etcd.")
				fmt.Println(err)
			}
		}
	}
	reqLogger.Info("Write value into etcd with specific group complete")
	return reconcile.Result{}, nil
}

// SyncAllCrData will sync all of Message resouces into etcd (Must in same Namespace)
func SyncAllCrData(bot *pohsienshihv1.Bot, etcdIP string) error {
	var r *ReconcileMessage
	foundMessages := &pohsienshihv1.MessageList{}

	// Get all the database Service in specific namespace.
	err := r.client.List(context.TODO(), foundMessages, &client.ListOptions{Namespace: bot.Namespace})
	if err != nil {
		fmt.Println("Can not found the Message Service by using namespace.")
		return err
	}
	for _, item := range foundMessages.Items {
	f2:
		for _, itemBotName := range item.Spec.Botname {
			if itemBotName == bot.Name {
				err = storeData(&item, etcdIP)
				if err != nil {
					return err
				}
				// Jump out the second loop.
				// This will prevent the duplicate writing value when both of botname and group be matched.
				break f2
			}
		}
	f3:
		for _, itemGroup := range item.Spec.Group {
			if itemGroup == bot.Spec.Group {
				err = storeData(&item, etcdIP)
				if err != nil {
					return err
				}
				break f3
			}
		}
	}
	return nil

}

func storeData(cr *pohsienshihv1.Message, etcdIP string) error {
	//  Connect and initial the etcd server
	cfg := etcdclient.Config{
		Endpoints: []string{"http://" + etcdIP + ":2379"},
		Transport: etcdclient.DefaultTransport,
		// Set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}
	c, err := etcdclient.New(cfg)
	if err != nil {
		fmt.Println(err)
	}
	kapi := etcdclient.NewKeysAPI(c)

	// Write a value into specific etcd server
	_, err = kapi.Set(context.Background(), cr.Spec.Keyword, cr.Spec.Response, nil)
	if err != nil {
		return err
	}
	fmt.Println("Write a value into " + etcdIP + " successfully")
	return nil

}
