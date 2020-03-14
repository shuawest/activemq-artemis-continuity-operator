package v2alpha1

import (
	"github.com/RHsyseng/operator-utils/pkg/olm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ActiveMQArtemisContinuitySpec defines the desired state of ActiveMQArtemisContinuity
// +k8s:openapi-gen=true
type ActiveMQArtemisContinuitySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	AdminUser                      string             `json:"adminUser,omitempty"`
	AdminPassword                  string             `json:"adminPassword,omitempty"`
	DeploymentPlan                 DeploymentPlanType `json:"deploymentPlan,omitempty"`
	Acceptors                      []AcceptorType     `json:"acceptors,omitempty"`
	Connectors                     []ConnectorType    `json:"connectors,omitempty"`
	Console                        ConsoleType        `json:"console,omitempty"`
	SiteId                         string             `json:"siteId"`
	LocalContinuityUser            string             `json:"localContinuityUser"`
	LocalContinuityPass            string             `json:"localContinuityPass"`
	PeerSiteUrl                    string             `json:"peerSiteUrl"`
	PeerContinuityUser             string             `json:"peerContinuityUser"`
	PeerContinuityPass             string             `json:"peerContinuityPass"`
	ServingAcceptors               []string           `json:"servingAcceptors"`
	ActiveOnStart                  bool               `json:"activeOnStart"`
	BrokerIdCacheSize              int32              `json:"brokerIdCacheSize,omitempty"`
	InflowStagingDelay             int32              `json:"inflowStagingDelay,omitempty"`
	BridgeInterval                 int32              `json:"bridgeInterval,omitempty"`
	BridgeIntervalMultiplier       float32            `json:"bridgeIntervalMultiplier,omitempty"`
	InflowAcksConsumedPollDuration int32              `json:"inflowAcksConsumedPollDuration,omitempty"`
	ActivationTimeout              int32              `json:"activationTimeout,omitempty"`
	ReorgManagement                bool               `json:"reorgManagement,omitempty"`
	ContinuityLogLevel             string             `json:"continuityLogLevel,omitempty"`
}

type DeploymentPlanType struct {
	Image              string `json:"image,omitempty"`
	Size               int32  `json:"size,omitempty"`
	RequireLogin       bool   `json:"requireLogin,omitempty"`
	PersistenceEnabled bool   `json:"persistenceEnabled,omitempty"`
	JournalType        string `json:"journalType,omitempty"`
	MessageMigration   bool   `json:"messageMigration,omitempty"`
}

type AcceptorType struct {
	Name                string `json:"name"`
	Port                int32  `json:"port,omitempty"`
	Protocols           string `json:"protocols,omitempty"`
	SSLEnabled          bool   `json:"sslEnabled,omitempty"`
	SSLSecret           string `json:"sslSecret,omitempty"`
	EnabledCipherSuites string `json:"enabledCipherSuites,omitempty"`
	EnabledProtocols    string `json:"enabledProtocols,omitempty"`
	NeedClientAuth      bool   `json:"needClientAuth,omitempty"`
	WantClientAuth      bool   `json:"wantClientAuth,omitempty"`
	VerifyHost          bool   `json:"verifyHost,omitempty"`
	SSLProvider         string `json:"sslProvider,omitempty"`
	SNIHost             string `json:"sniHost,omitempty"`
	Expose              bool   `json:"expose,omitempty"`
	AnycastPrefix       string `json:"anycastPrefix,omitempty"`
	MulticastPrefix     string `json:"multicastPrefix,omitempty"`
	ConnectionsAllowed  int    `json:"connectionsAllowed,omitempty"`
}

type ConnectorType struct {
	Name                string `json:"name"`
	Type                string `json:"type,omitempty"`
	Host                string `json:"host"`
	Port                int32  `json:"port"`
	SSLEnabled          bool   `json:"sslEnabled,omitempty"`
	SSLSecret           string `json:"sslSecret,omitempty"`
	EnabledCipherSuites string `json:"enabledCipherSuites,omitempty"`
	EnabledProtocols    string `json:"enabledProtocols,omitempty"`
	NeedClientAuth      bool   `json:"needClientAuth,omitempty"`
	WantClientAuth      bool   `json:"wantClientAuth,omitempty"`
	VerifyHost          bool   `json:"verifyHost,omitempty"`
	SSLProvider         string `json:"sslProvider,omitempty"`
	SNIHost             string `json:"sniHost,omitempty"`
	Expose              bool   `json:"expose,omitempty"`
}

type ConsoleType struct {
	Expose        bool   `json:"expose,omitempty"`
	SSLEnabled    bool   `json:"sslEnabled,omitempty"`
	SSLSecret     string `json:"sslSecret,omitempty"`
	UseClientAuth bool   `json:"useClientAuth,omitempty"`
}

// ActiveMQArtemisContinuityStatus defines the observed state of ActiveMQArtemisContinuity
// +k8s:openapi-gen=true
type ActiveMQArtemisContinuityStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html
	PodStatus olm.DeploymentStatus `json:"podStatus"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ActiveMQArtemisContinuity is the Schema for the activemqartemiscontinuities API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type ActiveMQArtemisContinuity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActiveMQArtemisContinuitySpec   `json:"spec,omitempty"`
	Status ActiveMQArtemisContinuityStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ActiveMQArtemisContinuityList contains a list of ActiveMQArtemisContinuity
type ActiveMQArtemisContinuityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ActiveMQArtemisContinuity `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ActiveMQArtemisContinuity{}, &ActiveMQArtemisContinuityList{})
}
