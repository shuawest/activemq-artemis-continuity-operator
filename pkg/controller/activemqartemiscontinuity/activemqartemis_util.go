package activemqartemiscontinuity

import (
	ctyapi "github.com/rh-messaging/activemq-artemis-continuity-operator/pkg/apis/broker/v2alpha1"
	superapi "github.com/rh-messaging/activemq-artemis-operator/pkg/apis/broker/v2alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ConvertToActiveMQArtemis(cty ctyapi.ActiveMQArtemisContinuity) superapi.ActiveMQArtemis {
	to := superapi.ActiveMQArtemis{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cty.ObjectMeta.Name,
			Namespace: cty.ObjectMeta.Namespace,
			Labels:    cty.ObjectMeta.Labels,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       cty.TypeMeta.Kind,
			APIVersion: cty.TypeMeta.APIVersion,
		},
		Spec: ConvertToActiveMQArtemisSpec(cty.Spec),
	}

	return to
}

func ConvertToActiveMQArtemisSpec(ctyspec ctyapi.ActiveMQArtemisContinuitySpec) superapi.ActiveMQArtemisSpec {
	to := superapi.ActiveMQArtemisSpec{
		AdminUser:     ctyspec.AdminUser,
		AdminPassword: ctyspec.AdminPassword,
		DeploymentPlan: superapi.DeploymentPlanType{
			Image:              ctyspec.DeploymentPlan.Image,
			Size:               ctyspec.DeploymentPlan.Size,
			RequireLogin:       ctyspec.DeploymentPlan.RequireLogin,
			PersistenceEnabled: ctyspec.DeploymentPlan.PersistenceEnabled,
			JournalType:        ctyspec.DeploymentPlan.JournalType,
			MessageMigration:   ctyspec.DeploymentPlan.MessageMigration,
		},
		Console: superapi.ConsoleType{
			Expose:        ctyspec.Console.Expose,
			SSLEnabled:    ctyspec.Console.SSLEnabled,
			SSLSecret:     ctyspec.Console.SSLSecret,
			UseClientAuth: ctyspec.Console.UseClientAuth,
		},
	}

	acceptorCount := len(ctyspec.Acceptors)
	superAcceptors := make([]superapi.AcceptorType, acceptorCount)
	for index, fromAcceptor := range ctyspec.Acceptors {
		superAcceptors[index] = ConvertToAcceptorType(fromAcceptor)
	}
	to.Acceptors = superAcceptors

	connectorCount := len(ctyspec.Connectors)
	superConnectors := make([]superapi.ConnectorType, connectorCount)
	for index, fromConnector := range ctyspec.Connectors {
		superConnectors[index] = ConvertToConnectorType(fromConnector)
	}
	to.Connectors = superConnectors

	return to
}

func ConvertToAcceptorType(from ctyapi.AcceptorType) superapi.AcceptorType {
	to := superapi.AcceptorType{
		Name:                from.Name,
		Port:                from.Port,
		Protocols:           from.Protocols,
		SSLEnabled:          from.SSLEnabled,
		SSLSecret:           from.SSLSecret,
		EnabledCipherSuites: from.EnabledCipherSuites,
		EnabledProtocols:    from.EnabledProtocols,
		NeedClientAuth:      from.NeedClientAuth,
		WantClientAuth:      from.WantClientAuth,
		VerifyHost:          from.VerifyHost,
		SSLProvider:         from.SSLProvider,
		SNIHost:             from.SNIHost,
		Expose:              from.Expose,
		AnycastPrefix:       from.AnycastPrefix,
		MulticastPrefix:     from.MulticastPrefix,
		ConnectionsAllowed:  from.ConnectionsAllowed,
	}
	return to
}

func ConvertToConnectorType(from ctyapi.ConnectorType) superapi.ConnectorType {
	to := superapi.ConnectorType{
		Name:                from.Name,
		Type:                from.Type,
		Host:                from.Host,
		Port:                from.Port,
		SSLEnabled:          from.SSLEnabled,
		SSLSecret:           from.SSLSecret,
		EnabledCipherSuites: from.EnabledCipherSuites,
		EnabledProtocols:    from.EnabledProtocols,
		NeedClientAuth:      from.NeedClientAuth,
		WantClientAuth:      from.WantClientAuth,
		VerifyHost:          from.VerifyHost,
		SSLProvider:         from.SSLProvider,
		SNIHost:             from.SNIHost,
		Expose:              from.Expose,
	}
	return to
}
