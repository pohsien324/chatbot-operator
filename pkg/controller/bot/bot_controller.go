package bot

import (
	"context"
	"fmt"

	pohsienshihv1 "github.com/pohsienshih/chatbot-operator/chatbot-operator/pkg/apis/pohsienshih/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/pohsienshih/chatbot-operator/chatbot-operator/pkg/controller/message"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_bot")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Bot Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileBot{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("bot-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Bot
	err = c.Watch(&source.Kind{Type: &pohsienshihv1.Bot{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Bot
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &pohsienshihv1.Bot{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileBot implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileBot{}

// ReconcileBot reconciles a Bot object
type ReconcileBot struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Bot object and makes changes based on the state read
// and what is in the Bot.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileBot) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Bot")

	// Fetch the Bot instance
	instance := &pohsienshihv1.Bot{}
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

	//----------------------------------------------------------------------------
	if instance.Spec.Group == "" {
		instance.Spec.Group = instance.Name
	}

	reqLogger.Info(instance.Namespace)
	reqLogger.Info(instance.ObjectMeta.Namespace)

	// Define the Namespace
	customNamespace := instance.Namespace
	namespace := newNamespace(customNamespace)
	if err := controllerutil.SetControllerReference(instance, namespace, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if the Namesapce exists
	foundNamespace := &corev1.Namespace{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: namespace.Name, Namespace: namespace.Namespace}, foundNamespace)

	if err != nil && errors.IsNotFound(err) {
		// Create the Namespace
		err = r.client.Create(context.TODO(), namespace)
		reqLogger.Info("Create the Namespace")
		if err != nil {
			reqLogger.Info("Failed to create the Namespace")
			return reconcile.Result{}, err
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Define DB Service (etcd)
	dbservice := newDbService(instance, customNamespace)
	if err := controllerutil.SetControllerReference(instance, dbservice, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if the DB Service exists (etcd)
	foundDBService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: dbservice.Name, Namespace: dbservice.Namespace}, foundDBService)
	if err != nil && errors.IsNotFound(err) {
		// Create the DB Service
		err = r.client.Create(context.TODO(), dbservice)
		reqLogger.Info("Create the DB service")
		if err != nil {
			reqLogger.Info("Failed to create the DB Service")
			return reconcile.Result{}, err

		}

	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Define the DB Pod (etcd)
	dbpod := newDBPod(instance, customNamespace)
	if err := controllerutil.SetControllerReference(instance, dbpod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if the DB Pod exists (etcd)
	sync := false
	foundDBPod := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: dbpod.Name, Namespace: dbpod.Namespace}, foundDBPod)
	if err != nil && errors.IsNotFound(err) {
		// Create DB Pod (etcd)
		err = r.client.Create(context.TODO(), dbpod)
		reqLogger.Info("Create the DB Pod")
		if err != nil {
			reqLogger.Info("Failed to create the DB Pod")
			return reconcile.Result{}, err
		}
		//  Sync all exist Message CR into MySQL DB
		sync = true

	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Define the bot Deployment
	botdeployment := newBotDeployment(instance, customNamespace, dbservice.Name)

	if err := controllerutil.SetControllerReference(instance, botdeployment, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if the bot Deployment exists
	foundBotDeployment := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: botdeployment.Name, Namespace: botdeployment.Namespace}, foundBotDeployment)
	if err != nil && errors.IsNotFound(err) {
		// Create secret
		err = r.client.Create(context.TODO(), botdeployment)
		reqLogger.Info("Create the bot Deployment")
		if err != nil {
			reqLogger.Info("Failed to create the bot Deployment")
			return reconcile.Result{}, err

		}
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Create the bot Service
	botservice := newBotService(instance, customNamespace)
	if err := controllerutil.SetControllerReference(instance, botservice, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if the bot Service exists
	foundBotService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: botservice.Name, Namespace: botservice.Namespace}, foundBotService)
	if err != nil && errors.IsNotFound(err) {
		// create the Service
		err = r.client.Create(context.TODO(), botservice)
		reqLogger.Info("Create the bot Service")
		if err != nil {
			reqLogger.Info("Failed to create the bot Service")
		}
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Synchronize the etcd database with all Message resources
	if sync == true {
		etcdIP := instance.Name + "-etcd" + "." + instance.Namespace + ".svc.cluster.local"
		err = message.SyncAllCrData(instance, etcdIP)
		if err != nil {
			reqLogger.Info("etcd database synchronization has failed.")
			fmt.Println(err)
		} else {
			reqLogger.Info("etcd database synchronized successfully")
		}

	}

	return reconcile.Result{}, nil
}

/* Naming rules
			Bot					etcd
1. Name:   cr.Name				cr.Name + "- etcd"
2. Label:  cr.Name			 	cr.Name  + "-etcd"
*/
func newNamespace(customname string) *corev1.Namespace {
	labels := map[string]string{
		"name": customname,
	}
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   customname,
			Labels: labels,
		},
	}
}

func newDBPod(cr *pohsienshihv1.Bot, namespace string) *corev1.Pod {
	labels := map[string]string{
		"name":     cr.Name + "-etcd",
		"group":    cr.Spec.Group,
		"bot_type": cr.Spec.Bottype,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-etcd",
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "etcd",
					Image: "quay.io/coreos/etcd:v3.3.18",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 2379,
						},
						{
							ContainerPort: 2380,
						},
					},
					Command: []string{
						"/usr/local/bin/etcd",
					},
					Args: []string{
						"--data-dir=/etcd-data",
						"--name",
						"node1",
						"--initial-advertise-peer-urls",
						"http://" + cr.Name + "-etcd" + ":2380",
						"--listen-peer-urls",
						"http://0.0.0.0:2380",
						"--advertise-client-urls",
						"http://" + cr.Name + "-etcd" + ":2379",
						"--listen-client-urls",
						"http://0.0.0.0:2379",
						"--initial-cluster",
						"node1=http://" + cr.Name + "-etcd" + ":2380",
					},
				},
			},
		},
	}
}

func newDbService(cr *pohsienshihv1.Bot, namespace string) *corev1.Service {
	labels := map[string]string{
		"name":     cr.Name + "-etcd",
		"group":    cr.Spec.Group,
		"bot_type": cr.Spec.Bottype,
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-etcd",
			Labels:    labels,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: "ClusterIP",
			Ports: []corev1.ServicePort{
				{
					Name: "client",
					Port: 2379,
				},
				{
					Name: "server",
					Port: 2380,
				},
			},
			Selector: labels,
		},
	}
}

func newBotDeployment(cr *pohsienshihv1.Bot, namespace string, etcdIP string) *appsv1.Deployment {
	labels := map[string]string{
		"name":     cr.Name,
		"group":    cr.Spec.Group,
		"bot_type": cr.Spec.Bottype,
	}
	var image string
	settingEnv := []corev1.EnvVar{
		{
			Name:  "etcd_server",
			Value: etcdIP,
		},
		{
			Name:  "etcd_port",
			Value: "2379",
		},
	}
	// Set the environment variables for different bots
	switch cr.Spec.Bottype {
	case "line":
		image = "pohsienshih/linebot-webhook-etcd:1.0.0"

		channelSecret := corev1.EnvVar{
			Name:  "CHANNEL_SECRET",
			Value: cr.Spec.Channelsecret,
		}

		channelToken := corev1.EnvVar{
			Name:  "CHANNEL_TOKEN",
			Value: cr.Spec.Channeltoken,
		}
		settingEnv = append(settingEnv, channelSecret)
		settingEnv = append(settingEnv, channelToken)

	case "facebook":
		image = "pohsienshih/messengerbot-webhook-etcd:1.0.0"
		verifyToken := corev1.EnvVar{
			Name:  "VERIFY_TOKEN",
			Value: cr.Spec.Verifytoken,
		}

		pageToken := corev1.EnvVar{
			Name:  "PAGE_TOKEN",
			Value: cr.Spec.Pagetoken,
		}
		settingEnv = append(settingEnv, verifyToken)
		settingEnv = append(settingEnv, pageToken)

	case "telegram":
		image = "pohsienshih/telegrambot-webhook-etcd:1.0.0"
		telegramToken := corev1.EnvVar{
			Name:  "TELEGRAM_TOKEN",
			Value: cr.Spec.Telegramtoken,
		}
		settingEnv = append(settingEnv, telegramToken)
	}
	// Define Deployment
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: cr.Spec.Size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "webhook",
							Image: image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
							Env: settingEnv,
						},
					},
				},
			},
		},
	}
}
func newBotService(cr *pohsienshihv1.Bot, namespace string) *corev1.Service {
	labels := map[string]string{
		"name":     cr.Name,
		"group":    cr.Spec.Group,
		"bot_type": cr.Spec.Bottype,
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: "NodePort",
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
			Selector: labels,
		},
	}
}
