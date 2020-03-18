package activemqartemiscontinuity

import (
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	ctyv2alpha1 "github.com/rh-messaging/activemq-artemis-continuity-operator/pkg/apis/broker/v2alpha1"
	"github.com/rh-messaging/activemq-artemis-operator/pkg/utils/namer"
	"github.com/rh-messaging/activemq-artemis-operator/pkg/utils/selectors"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func buildReconcileWithFakeClientWithMocks(objs []runtime.Object, t *testing.T) *ReconcileActiveMQArtemisContinuity {

	registerObjs := []runtime.Object{&ctyv2alpha1.ActiveMQArtemisContinuity{}, &corev1.Service{}, &appsv1.StatefulSet{}, &appsv1.StatefulSetList{}, &corev1.Pod{}, &routev1.Route{}, &routev1.RouteList{}, &corev1.PersistentVolumeClaimList{}, &corev1.ServiceList{}}
	registerObjs = append(registerObjs)
	ctyv2alpha1.SchemeBuilder.Register(registerObjs...)
	ctyv2alpha1.SchemeBuilder.Register()

	scheme, err := ctyv2alpha1.SchemeBuilder.Build()
	if err != nil {
		assert.Fail(t, "unable to build scheme")
	}
	client := fake.NewFakeClientWithScheme(scheme, objs...)

	return &ReconcileActiveMQArtemisContinuity{
		client: client,
		scheme: scheme,
	}

}

var (
	NameBuilder namer.NamerData

	labels = selectors.LabelBuilder.Labels()
	f      = false
	t      = true

	podTemplate = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: AMQinstance.Namespace,
			Name:      AMQinstance.Name,
			Labels:    AMQinstance.Labels,
		},
		//Spec: pods.NewPodTemplateSpecForCR(&AMQinstance).Spec,
		//Spec: pods.NewPodTemplateSpecForCR(namespacedName).Spec,
	}

	AMQinstance = ctyv2alpha1.ActiveMQArtemisContinuity{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "activemq-artemis-test",
			Namespace: "activemq-artemis-operator-ns",
			Labels:    labels,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ActiveMQArtemisContinuity",
			APIVersion: "broker.amq.io/v2alpha1",
		},
		Spec: ctyv2alpha1.ActiveMQArtemisContinuitySpec{
			AdminUser:     "admin",
			AdminPassword: "admin",
			DeploymentPlan: ctyv2alpha1.DeploymentPlanType{
				Size:               2,
				Image:              "quay.io/artemiscloud/activemq-artemis-operator:latest",
				PersistenceEnabled: false,
				RequireLogin:       false,
				MessageMigration:   &f,
			},
			Console: ctyv2alpha1.ConsoleType{
				Expose: true,
			},
			Acceptors: []ctyv2alpha1.AcceptorType{
				{
					Name:                "my",
					Protocols:           "amqp",
					Port:                5672,
					SSLEnabled:          false,
					EnabledCipherSuites: "SSL_RSA_WITH_RC4_128_SHA,SSL_DH_anon_WITH_3DES_EDE_CBC_SHA",
					EnabledProtocols:    " TLSv1,TLSv1.1,TLSv1.2",
					NeedClientAuth:      true,
					WantClientAuth:      true,
					VerifyHost:          true,
					SSLProvider:         "JDK",
					SNIHost:             "localhost",
					Expose:              true,
					AnycastPrefix:       "jms.topic",
					MulticastPrefix:     "/queue/",
				},
			},
			Connectors: []ctyv2alpha1.ConnectorType{
				{
					Name:                "my-c",
					Host:                "localhost",
					Port:                22222,
					SSLEnabled:          false,
					EnabledCipherSuites: "SSL_RSA_WITH_RC4_128_SHA,SSL_DH_anon_WITH_3DES_EDE_CBC_SHA",
					EnabledProtocols:    " TLSv1,TLSv1.1,TLSv1.2",
					NeedClientAuth:      true,
					WantClientAuth:      true,
					VerifyHost:          true,
					SSLProvider:         "JDK",
					SNIHost:             "localhost",
					Expose:              true,
				},
			},
		},
	}

	// namespacedName = types.NamespacedName{
	// 	Namespace: AMQinstance.Namespace,
	// 	Name:      AMQinstance.Name,
	// }
	// container   = containers.MakeContainer(namespacedName.Name, "quay.io/artemiscloud/activemq-artemis-operator:latest", environments.MakeEnvVarArrayForCR(&AMQinstance))
	// podTemplate = corev1.Pod{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Namespace: AMQinstance.Namespace,
	// 		Name:      AMQinstance.Name,
	// 		Labels:    AMQinstance.Labels,
	// 	},
	// 	//Spec: pods.NewPodTemplateSpecForCR(&AMQinstance).Spec,
	// 	Spec: pods.NewPodTemplateSpecForCR(namespacedName).Spec,
	// }
)
