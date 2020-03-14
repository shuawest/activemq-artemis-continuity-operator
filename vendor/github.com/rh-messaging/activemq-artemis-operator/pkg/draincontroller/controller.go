/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package draincontroller

import (
	"fmt"
	"github.com/rh-messaging/activemq-artemis-operator/pkg/resources/environments"
	"github.com/rh-messaging/activemq-artemis-operator/pkg/resources/statefulsets"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	svc "github.com/rh-messaging/activemq-artemis-operator/pkg/resources/services"

	"encoding/json"
	"k8s.io/apimachinery/pkg/labels"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sort"
	"strconv"
	"strings"
)

var log = logf.Log.WithName("controller_activemqartemisscaledown")

const controllerAgentName = "statefulset-drain-controller"
const AnnotationStatefulSet = "statefulsets.kubernetes.io/drainer-pod-owner" // TODO: can we replace this with an OwnerReference with the StatefulSet as the owner?
const AnnotationDrainerPodTemplate = "statefulsets.kubernetes.io/drainer-pod-template"

const LabelDrainPod = "drain-pod"

const (
	SuccessCreate    = "SuccessfulCreate"
	DrainSuccess     = "DrainSuccess"
	PVCDeleteSuccess = "SuccessfulPVCDelete"
	PodDeleteSuccess = "SuccessfulDelete"

	MessageDrainPodCreated  = "create Drain Pod %s in StatefulSet %s successful"
	MessageDrainPodFinished = "drain Pod %s in StatefulSet %s completed successfully"
	MessageDrainPodDeleted  = "delete Drain Pod %s in StatefulSet %s successful"
	MessagePVCDeleted       = "delete Claim %s in StatefulSet %s successful"
)

// TODO: Remove this hack
var globalPodTemplateJson string = "{\n \"metadata\": {\n    \"labels\": {\n      \"app\": \"CRNAME-amq-drainer\"\n    }\n  },\n  \"spec\": {\n \"serviceAccount\": \"activemq-artemis-operator\",\n \"serviceAccountName\": \"activemq-artemis-operator\",\n \"terminationGracePeriodSeconds\": 5,\n    \"containers\": [\n {\n        \"env\": [\n          {\n            \"name\": \"AMQ_EXTRA_ARGS\",\n            \"value\": \"--no-autotune\"\n },\n          {\n            \"name\": \"HEADLESS_SVC_NAME\",\n            \"value\": \"HEADLESSSVCNAMEVALUE\"\n },\n          {\n            \"name\": \"PING_SVC_NAME\",\n            \"value\": \"PINGSVCNAMEVALUE\"\n },\n          {\n            \"name\": \"AMQ_USER\",\n \"value\": \"admin\"\n          },\n          {\n            \"name\": \"AMQ_PASSWORD\",\n            \"value\": \"admin\"\n },\n          {\n            \"name\": \"AMQ_ROLE\",\n \"value\": \"admin\"\n          },\n          {\n            \"name\": \"AMQ_NAME\",\n            \"value\": \"amq-broker\"\n },\n          {\n            \"name\": \"AMQ_TRANSPORTS\",\n \"value\": \"openwire,amqp,stomp,mqtt,hornetq\"\n          },\n {\n            \"name\": \"AMQ_GLOBAL_MAX_SIZE\",\n            \"value\": \"100mb\"\n          },\n          {\n            \"name\": \"AMQ_DATA_DIR\",\n            \"value\": \"/opt/CRNAME/data\"\n          },\n          {\n \"name\": \"AMQ_DATA_DIR_LOGGING\",\n            \"value\": \"true\"\n          },\n          {\n            \"name\": \"AMQ_CLUSTERED\",\n            \"value\": \"true\"\n },\n          {\n            \"name\": \"AMQ_REPLICAS\",\n \"value\": \"1\"\n          },\n          {\n            \"name\": \"AMQ_CLUSTER_USER\",\n            \"value\": \"CLUSTERUSER\"\n },\n          {\n            \"name\": \"AMQ_CLUSTER_PASSWORD\",\n            \"value\": \"CLUSTERPASS\"\n          },\n          {\n            \"name\": \"POD_NAMESPACE\",\n            \"valueFrom\": {\n \"fieldRef\": {\n                \"fieldPath\": \"metadata.namespace\"\n              }\n            }\n },\n          {\n            \"name\": \"OPENSHIFT_DNS_PING_SERVICE_PORT\",\n            \"value\": \"8888\"\n          }\n        ],\n        \"image\": \"SSIMAGE\",\n \"name\": \"drainer-amq\",\n\n        \"command\": [\"/bin/sh\", \"-c\", \"echo \\\"Starting the drainer\\\" ; /opt/amq/bin/drain.sh; echo \\\"Drain completed! Exit code $?\\\"\"],\n        \"volumeMounts\": [\n          {\n            \"name\": \"CRNAME\",\n \"mountPath\": \"/opt/CRNAME/data\"\n          }\n ]\n      }\n    ]\n }\n}"

type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface

	statefulSetLister  appslisters.StatefulSetLister
	statefulSetsSynced cache.InformerSynced
	pvcLister          corelisters.PersistentVolumeClaimLister
	pvcsSynched        cache.InformerSynced
	podLister          corelisters.PodLister
	podsSynced         cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	localOnly bool
}

// NewController returns a new sample controller
func NewController(
	kubeclientset kubernetes.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	namespace string,
	localOnly bool) *Controller {

	// obtain references to shared index informers for the Deployment and Foo
	// types.
	statefulSetInformer := kubeInformerFactory.Apps().V1().StatefulSets()
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	podInformer := kubeInformerFactory.Core().V1().Pods()

	// Create event broadcaster
	// Add statefulset-drain-controller types to the default Kubernetes Scheme so Events can be
	// logged for statefulset-drain-controller types.
	log.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Info)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events(namespace)})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
	itemExponentialFailureRateLimiter := workqueue.NewItemExponentialFailureRateLimiter(5*time.Second, 300*time.Second)

	controller := &Controller{
		kubeclientset:      kubeclientset,
		statefulSetLister:  statefulSetInformer.Lister(),
		statefulSetsSynced: statefulSetInformer.Informer().HasSynced,
		pvcLister:          pvcInformer.Lister(),
		pvcsSynched:        pvcInformer.Informer().HasSynced,
		podLister:          podInformer.Lister(),
		podsSynced:         podInformer.Informer().HasSynced,
		workqueue:          workqueue.NewNamedRateLimitingQueue(itemExponentialFailureRateLimiter, "StatefulSets"),
		recorder:           recorder,
		localOnly:          localOnly,
	}

	log.Info("Setting up event handlers")
	// Set up an event handler for when Foo resources change
	statefulSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueStatefulSet,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueStatefulSet(new)
		},
	})
	// Set up an event handler for when Pod resources change. This
	// handler will lookup the owner of the given Pod, and if it is
	// owned by a StatefulSet resource will enqueue that StatefulSet
	// resource for processing. This way, we don't need to implement
	// custom logic for handling Pod resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handlePod,
		UpdateFunc: func(old, new interface{}) {
			newPod := new.(*corev1.Pod)
			oldPod := old.(*corev1.Pod)
			if newPod.ResourceVersion == oldPod.ResourceVersion {
				// Periodic resync will send update events for all known Pods.
				// Two different versions of the same Pod will always have different RVs.
				return
			}
			controller.handlePod(newPod)
		},
		DeleteFunc: controller.handlePod,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {

	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	log.Info("Starting StatefulSet scaledown cleanup controller")

	// Wait for the caches to be synced before starting workers
	log.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.statefulSetsSynced, c.podsSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	log.Info("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	log.Info("Started workers")
	<-stopCh
	log.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		// Run the syncHandler, passing it the namespace/name string of the
		// Foo resource to be synced.
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '" + key + ": " + err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		log.V(4).Info("Successfully processed '" + key + "'")
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the StatefulSet resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {

	log.V(4).Info("--------------------------------------------------------------------")
	log.V(4).Info("SyncHandler invoked for " + key)

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)

	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: " + key))
		return nil
	}

	// Get the StatefulSet resource with this namespace/name
	sts, err := c.statefulSetLister.StatefulSets(namespace).Get(name)

	if err != nil {
		// The StatefulSet may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("StatefulSet " + key + " in work queue no longer exists"))
			return nil
		}

		return err
	}

	return c.processStatefulSet(sts)
}

func (c *Controller) processStatefulSet(sts *appsv1.StatefulSet) error {
	// TODO: think about scale-down during a rolling upgrade

	if 0 == *sts.Spec.Replicas {
		// Ensure data is not touched in the case of complete scaledown
		log.V(5).Info("Ignoring StatefulSet " + sts.Name + " because replicas set to 0.")
		return nil
	}

	log.V(5).Info("Statefulset " + sts.Name + " Spec.Replicas set to " + strconv.Itoa(int(*sts.Spec.Replicas)))

	if len(sts.Spec.VolumeClaimTemplates) == 0 {
		// nothing to do, as the stateful pods don't use any PVCs
		log.Info("Ignoring StatefulSet " + sts.Name + " because it does not use any PersistentVolumeClaims.")
		return nil
	}
	log.V(5).Info("Statefulset " + sts.Name + " Spec.VolumeClaimTemplates is " + strconv.Itoa((len(sts.Spec.VolumeClaimTemplates))))

	//if sts.Annotations[AnnotationDrainerPodTemplate] == "" {
	//	log.Info("Ignoring StatefulSet '%s' because it does not define a drain pod template.", sts.Name)
	//	return nil
	//}

	claimsGroupedByOrdinal, err := c.getClaims(sts)
	if err != nil {
		err = fmt.Errorf("Error while getting list of PVCs in namespace %s: %s", sts.Namespace, err)
		log.Error(err, "Error while getting list of PVCs in namespace "+sts.Namespace)
		return err
	}

	ordinals := make([]int, 0, len(claimsGroupedByOrdinal))
	for k := range claimsGroupedByOrdinal {
		ordinals = append(ordinals, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ordinals)))

	for _, ordinal := range ordinals {

		if 0 == ordinal {
			// This assumes order on scale up and down is enforced, i.e. the system waits for n, n-1,... 2, 1 to scaledown before attempting 0
			log.V(5).Info("Ignoring ordinal 0 as no other pod to drain to.")
			continue
		}

		// TODO check if the number of claims matches the number of StatefulSet's volumeClaimTemplates. What if it doesn't?

		podName := getPodName(sts, ordinal)

		pod, err := c.podLister.Pods(sts.Namespace).Get(podName)

		if err != nil && !errors.IsNotFound(err) {
			log.Error(err, "Error while getting Pod "+podName)
			return err
		}

		// Is it a drain pod or a regular stateful pod?
		if isDrainPod(pod) {
			err = c.cleanUpDrainPodIfNeeded(sts, pod, ordinal)
			if err != nil {
				return err
			}

			if sts.Spec.PodManagementPolicy == appsv1.OrderedReadyPodManagement {
				// don't create additional drain pods; they will be created in one of the
				// next invocations of this method, when the current drain pod finishes
				break
			}
		} else {
			// DO nothing. Pod is a regular stateful pod
			//log.Info("Pod '%s' exists. Not taking any action.", podName)
			//return nil
		}

		// TODO: scale down to zero? should what happens on such events be configurable? there may or may not be anywhere to drain to
		if int32(ordinal) >= *sts.Spec.Replicas {
			// PVC exists, but its ordinal is higher than the current last stateful pod's ordinal;
			// this means the PVC is an orphan and should be drained & deleted

			// If the Pod doesn't exist, we'll create it
			if pod == nil { // TODO: what if the PVC doesn't exist here (or what if it's deleted just after we create the pod)
				log.Info("Found orphaned PVC(s) for ordinal " + strconv.Itoa(ordinal) + ". Creating drain pod " + podName)

				// Check to ensure we have a pod to drain to
				ordinalZeroPodName := getPodName(sts, 0)
				ordinalZeroPod, err := c.podLister.Pods(sts.Namespace).Get(ordinalZeroPodName)
				if err != nil {
					log.Error(err, "Error while getting ordinal zero pod "+podName+": "+err.Error())
					return err
				}

				// Ensure that at least the ordinal zero pod is running
				if corev1.PodRunning != ordinalZeroPod.Status.Phase {
					//log.Info("Ordinal zero pod '%s' status phase '%s', waiting for it to be Running.", sts.Name, pod.Status.Phase)
					log.Info("Ordinal zero pod " + sts.Name + " status phase not PodRunning, waiting for it to be Running.")
					continue
				}

				// Ensure that at least the ordinal zero pod is Ready
				podConditions := ordinalZeroPod.Status.Conditions

				ordinalZeroPodReady := false
				for _, podCondition := range podConditions {
					//log.V(5).Info("Ordinal zero pod condition %s", podCondition)
					if corev1.PodReady == podCondition.Type {
						if corev1.ConditionTrue != podCondition.Status {
							log.Info("Ordinal zero pod " + sts.Name + " podCondition Ready not True, waiting for it to True.")
						}
						if corev1.ConditionTrue == podCondition.Status {
							log.Info("Ordinal zero pod " + sts.Name + " podCondition Ready True, proceeding to create drainer pod.")
							ordinalZeroPodReady = true
						}
					}
				}

				if false == ordinalZeroPodReady {
					continue
				}

				pod, err := newPod(sts, ordinal)
				if err != nil {
					return fmt.Errorf("Can't create drain Pod object: %s", err)
				}
				pod, err = c.kubeclientset.CoreV1().Pods(sts.Namespace).Create(pod)

				// If an error occurs during Create, we'll requeue the item so we can
				// attempt processing again later. This could have been caused by a
				// temporary network failure, or any other transient reason.
				if err != nil {
					log.Error(err, "Error while creating drain Pod "+podName+": ")
					return err
				}

				if !c.localOnly {
					c.recorder.Event(sts, corev1.EventTypeNormal, SuccessCreate, fmt.Sprintf(MessageDrainPodCreated, podName, sts.Name))
				}

				continue
				//} else {
				//	log.Info("Pod '%s' exists. Not taking any action.", podName)
			}
		}
	}

	// TODO: add status annotation (what info?)
	return nil
}

func (c *Controller) getClaims(sts *appsv1.StatefulSet) (claimsGroupedByOrdinal map[int][]*corev1.PersistentVolumeClaim, err error) {
	// shouldn't use statefulset.Spec.Selector.MatchLabels, as they don't always match; sts controller looks up pvcs by name!
	allClaims, err := c.pvcLister.PersistentVolumeClaims(sts.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	log.V(5).Info("getClaims allClaims len is %d", len(allClaims))

	claimsMap := map[int][]*corev1.PersistentVolumeClaim{}
	for _, pvc := range allClaims {
		log.V(5).Info("getClaims allClaims pvc name is " + pvc.Name)
		if pvc.DeletionTimestamp != nil {
			log.Info("PVC " + pvc.Name + " is being deleted. Ignoring it.")
			continue
		}

		name, ordinal, err := extractNameAndOrdinal(pvc.Name)
		if err != nil {
			continue
		}

		for _, t := range sts.Spec.VolumeClaimTemplates {
			if name == fmt.Sprintf("%s-%s", t.Name, sts.Name) {
				if claimsMap[ordinal] == nil {
					claimsMap[ordinal] = []*corev1.PersistentVolumeClaim{}
				}
				claimsMap[ordinal] = append(claimsMap[ordinal], pvc)
			}
		}
	}

	return claimsMap, nil
}

func (c *Controller) cleanUpDrainPodIfNeeded(sts *appsv1.StatefulSet, pod *corev1.Pod, ordinal int) error {
	// Drain Pod already exists. Check if it's done draining.
	podName := getPodName(sts, ordinal)

	switch pod.Status.Phase {
	case (corev1.PodSucceeded):
		log.Info("Drain pod " + podName + " finished.")
		if !c.localOnly {
			c.recorder.Event(sts, corev1.EventTypeNormal, DrainSuccess, fmt.Sprintf(MessageDrainPodFinished, podName, sts.Name))
		}

		for _, pvcTemplate := range sts.Spec.VolumeClaimTemplates {
			pvcName := getPVCName(sts, pvcTemplate.Name, int32(ordinal))
			log.Info("Deleting PVC " + pvcName)
			err := c.kubeclientset.CoreV1().PersistentVolumeClaims(sts.Namespace).Delete(pvcName, nil)
			if err != nil {
				return err
			}
			if !c.localOnly {
				c.recorder.Event(sts, corev1.EventTypeNormal, PVCDeleteSuccess, fmt.Sprintf(MessagePVCDeleted, pvcName, sts.Name))
			}
		}

		// TODO what if the user scales up the statefulset and the statefulset controller creates the new pod after we delete the pod but before we delete the PVC
		// TODO what if we crash after we delete the PVC, but before we delete the pod?

		log.Info("Deleting drain pod " + podName)
		err := c.kubeclientset.CoreV1().Pods(sts.Namespace).Delete(podName, nil)
		if err != nil {
			return err
		}
		if !c.localOnly {
			c.recorder.Event(sts, corev1.EventTypeNormal, PodDeleteSuccess, fmt.Sprintf(MessageDrainPodDeleted, podName, sts.Name))
		}
		break
	case (corev1.PodFailed):
		log.Info("Drain pod " + podName + " failed.")
		break
	default:
		str := fmt.Sprintf("Drain pod Phase was %s", pod.Status.Phase)
		log.Info(str)
		break
	}

	return nil
}

func isDrainPod(pod *corev1.Pod) bool {
	return pod != nil && pod.ObjectMeta.Annotations[AnnotationStatefulSet] != ""
}

// enqueueStatefulSet takes a StatefulSet resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than StatefulSet.
func (c *Controller) enqueueStatefulSet(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

// handlePod will take any resource implementing metav1.Object and attempt
// to find the StatefulSet resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that StatefulSet resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handlePod(obj interface{}) {

	if !c.cachesSynced() {
		return
	}

	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		log.V(4).Info("Recovered deleted object " + object.GetName() + " from tombstone")
	}
	log.V(5).Info("Processing object: " + object.GetName())

	stsNameFromAnnotation := object.GetAnnotations()[AnnotationStatefulSet]
	if stsNameFromAnnotation != "" {
		log.V(5).Info("Found pod with " + AnnotationStatefulSet + " annotation pointing to StatefulSet " + stsNameFromAnnotation + ". Enqueueing StatefulSet.")
		sts, err := c.statefulSetLister.StatefulSets(object.GetNamespace()).Get(stsNameFromAnnotation)
		if err != nil {
			log.V(4).Info("Error retrieving StatefulSet " + stsNameFromAnnotation + ": " + err.Error())
			return
		}

		if 0 == *sts.Spec.Replicas {
			log.V(5).Info("NameFromAnnotation not enqueueing Statefulset " + sts.Name + " as Spec.Replicas is 0.")
			return
		}

		c.enqueueStatefulSet(sts)
		return
	}

	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a StatefulSet, we should not do anything more
		// with it.
		if ownerRef.Kind != "StatefulSet" {
			return
		}

		sts, err := c.statefulSetLister.StatefulSets(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			log.V(4).Info("ignoring orphaned object " + object.GetSelfLink() + " of StatefulSet " + ownerRef.Name)
			return
		}

		if 0 == *sts.Spec.Replicas {
			log.V(5).Info("Name from ownerRef.Name not enqueueing Statefulset " + sts.Name + " as Spec.Replicas is 0.")
			return
		}

		c.enqueueStatefulSet(sts)
		return
	}
}

func (c *Controller) cachesSynced() bool {
	return true // TODO do we even need this?
}

func newPod(sts *appsv1.StatefulSet, ordinal int) (*corev1.Pod, error) {

	//podTemplateJson := sts.Annotations[AnnotationDrainerPodTemplate]
	//TODO: Remove this blatant hack
	podTemplateJson := globalPodTemplateJson
	podTemplateJson = strings.Replace(podTemplateJson, "CRNAME", statefulsets.GLOBAL_CRNAME, -1)
	podTemplateJson = strings.Replace(podTemplateJson, "CLUSTERUSER", environments.GLOBAL_AMQ_CLUSTER_USER, 1)
	podTemplateJson = strings.Replace(podTemplateJson, "CLUSTERPASS", environments.GLOBAL_AMQ_CLUSTER_PASSWORD, 1)
	podTemplateJson = strings.Replace(podTemplateJson, "HEADLESSSVCNAMEVALUE", svc.HeadlessNameBuilder.Name(), 1)
	podTemplateJson = strings.Replace(podTemplateJson, "PINGSVCNAMEVALUE", svc.PingNameBuilder.Name(), 1)
	image := sts.Spec.Template.Spec.Containers[0].Image
	if "" == image {
		image = "registry.redhat.io/amq7/amq-broker:7.5"
	}
	podTemplateJson = strings.Replace(podTemplateJson, "SSIMAGE", image, 1)
	if podTemplateJson == "" {
		return nil, fmt.Errorf("No drain pod template configured for StatefulSet " + sts.Name)
	}
	pod := corev1.Pod{}
	err := json.Unmarshal([]byte(podTemplateJson), &pod)
	if err != nil {
		return nil, fmt.Errorf("Can't unmarshal DrainerPodTemplate JSON from annotation: " + err.Error())
	}

	pod.Name = getPodName(sts, ordinal)
	pod.Namespace = sts.Namespace

	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	pod.Labels[LabelDrainPod] = pod.Name
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	pod.Annotations[AnnotationStatefulSet] = sts.Name

	// TODO: cannot set blockOwnerDeletion if an ownerReference refers to a resource you can't set finalizers on: User "system:serviceaccount:kube-system:statefulset-drain-controller" cannot update statefulsets/finalizers.apps
	//if pod.OwnerReferences == nil {
	//	pod.OwnerReferences = []metav1.OwnerReference{}
	//}
	//pod.OwnerReferences = append(pod.OwnerReferences, *metav1.NewControllerRef(sts, schema.GroupVersionKind{
	//	Group:   appsv1beta1.SchemeGroupVersion.Group,
	//	Version: appsv1beta1.SchemeGroupVersion.Version,
	//	Kind:    "StatefulSet",
	//}))

	pod.Spec.RestartPolicy = corev1.RestartPolicyOnFailure

	for _, pvcTemplate := range sts.Spec.VolumeClaimTemplates {
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{ // TODO: override existing volumes with the same name
			Name: pvcTemplate.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: getPVCName(sts, pvcTemplate.Name, int32(ordinal)),
				},
			},
		})
	}

	return &pod, nil
}

func getPodName(sts *appsv1.StatefulSet, ordinal int) string {
	return fmt.Sprintf("%s-%d", sts.Name, ordinal)
}

func getPVCName(sts *appsv1.StatefulSet, volumeClaimName string, ordinal int32) string {
	return fmt.Sprintf("%s-%s-%d", volumeClaimName, sts.Name, ordinal)
}

func extractNameAndOrdinal(pvcName string) (string, int, error) {
	idx := strings.LastIndexAny(pvcName, "-")
	if idx == -1 {
		return "", 0, fmt.Errorf("PVC not created by a StatefulSet")
	}

	name := pvcName[:idx]
	ordinal, err := strconv.Atoi(pvcName[idx+1:])
	if err != nil {
		return "", 0, fmt.Errorf("PVC not created by a StatefulSet")
	}
	return name, ordinal, nil
}
