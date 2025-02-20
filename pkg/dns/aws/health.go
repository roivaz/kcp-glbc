package aws

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/rs/xid"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"

	v1 "github.com/kuadrant/kcp-glbc/pkg/apis/kuadrant/v1"
)

const (
	idTag = "kuadrant.dev/healthcheck"
)

var (
	callerReference func(id string) *string
)

type Route53HealthCheckReconciler struct {
	client *InstrumentedRoute53
	logger logr.Logger
}

func newRoute53HealthCheckReconciler(c *InstrumentedRoute53, l logr.Logger) *Route53HealthCheckReconciler {
	return &Route53HealthCheckReconciler{
		client: c,
		logger: l.WithName("health"),
	}
}

func (r *Route53HealthCheckReconciler) reconcile(ctx context.Context, spec v1.HealthCheck, endpoint *v1.Endpoint) error {
	healthCheck, exists, err := r.findHealthCheck(ctx, endpoint)
	if err != nil {
		return err
	}

	defer func() {
		if healthCheck != nil {
			endpoint.SetProviderSpecific(ProviderSpecificHealthCheckID, *healthCheck.Id)
		}
	}()

	if exists {
		return r.updateHealthCheck(ctx, spec, endpoint, healthCheck)
	}

	healthCheck, err = r.createHealthCheck(ctx, spec, endpoint)
	return err
}

func (r *Route53HealthCheckReconciler) deleteHealthCheck(ctx context.Context, endpoint *v1.Endpoint) error {
	healthCheck, found, err := r.findHealthCheck(ctx, endpoint)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	_, err = r.client.DeleteHealthCheckWithContext(ctx, &route53.DeleteHealthCheckInput{
		HealthCheckId: healthCheck.Id,
	})

	if err != nil {
		return err
	}

	endpoint.DeleteProviderSpecific(ProviderSpecificHealthCheckID)
	return err
}

func (r *Route53HealthCheckReconciler) findHealthCheck(ctx context.Context, endpoint *v1.Endpoint) (*route53.HealthCheck, bool, error) {
	id, hasId := getHealthCheckId(endpoint)
	if !hasId {
		return nil, false, nil
	}

	response, err := r.client.GetHealthCheckWithContext(ctx, &route53.GetHealthCheckInput{
		HealthCheckId: &id,
	})
	if err != nil {
		return nil, false, err
	}

	return response.HealthCheck, true, nil

}

func (r *Route53HealthCheckReconciler) createHealthCheck(ctx context.Context, spec v1.HealthCheck, endpoint *v1.Endpoint) (*route53.HealthCheck, error) {
	address, _ := endpoint.GetAddress()
	host := endpoint.DNSName

	// Create the health check
	output, err := r.client.CreateHealthCheck(&route53.CreateHealthCheckInput{
		CallerReference: callerReference(spec.Id),
		HealthCheckConfig: &route53.HealthCheckConfig{
			IPAddress:                &address,
			FullyQualifiedDomainName: &host,
			Port:                     spec.Port,
			ResourcePath:             &spec.Path,
			Type:                     healthCheckType(spec.Protocol),
			FailureThreshold:         spec.FailureThreshold,
		},
	})
	if err != nil {
		return nil, err
	}

	name := spec.Name
	// Add the tag to identify it
	_, err = r.client.ChangeTagsForResourceWithContext(ctx, &route53.ChangeTagsForResourceInput{
		AddTags: []*route53.Tag{
			{
				Key:   aws.String(idTag),
				Value: aws.String(spec.Id),
			},
			{
				Key:   aws.String("Name"),
				Value: &name,
			},
		},
		ResourceId:   output.HealthCheck.Id,
		ResourceType: aws.String(route53.TagResourceTypeHealthcheck),
	})
	if err != nil {
		return nil, err
	}

	return output.HealthCheck, nil
}

func (r *Route53HealthCheckReconciler) updateHealthCheck(ctx context.Context, spec v1.HealthCheck, endpoint *v1.Endpoint, healthCheck *route53.HealthCheck) error {
	diff := healthCheckDiff(healthCheck, spec, endpoint)
	if diff == nil {
		return nil
	}

	r.logger.Info("Updating health check", "id", *healthCheck.Id, "change", diff)

	_, err := r.client.UpdateHealthCheckWithContext(ctx, diff)
	if err != nil {
		return err
	}

	return nil
}

// healthCheckDiff creates a `UpdateHealthCheckInput` object with the fields to
// update on healthCheck based on the given spec.
// If the health check matches the spec, returns `nil`
func healthCheckDiff(healthCheck *route53.HealthCheck, spec v1.HealthCheck, endpoint *v1.Endpoint) *route53.UpdateHealthCheckInput {
	var result *route53.UpdateHealthCheckInput

	diff := func() *route53.UpdateHealthCheckInput {
		if result == nil {
			result = &route53.UpdateHealthCheckInput{
				HealthCheckId: healthCheck.Id,
			}
		}

		return result
	}

	if !strValuesEqual(&endpoint.DNSName, healthCheck.HealthCheckConfig.FullyQualifiedDomainName) {
		diff().FullyQualifiedDomainName = &endpoint.DNSName
	}

	address, _ := endpoint.GetAddress()
	if !strValuesEqual(&address, healthCheck.HealthCheckConfig.IPAddress) {
		diff().IPAddress = &address
	}
	if !strValuesEqual(&spec.Path, healthCheck.HealthCheckConfig.ResourcePath) {
		diff().ResourcePath = &spec.Path
	}

	if !intValuesEqual(spec.Port, healthCheck.HealthCheckConfig.Port) {
		diff().Port = spec.Port
	}
	if !intValuesEqual(spec.FailureThreshold, healthCheck.HealthCheckConfig.FailureThreshold) {
		diff().FailureThreshold = spec.FailureThreshold
	}

	return result
}

func init() {
	sid := xid.New()
	callerReference = func(s string) *string {
		return aws.String(fmt.Sprintf("%s.%s", s, sid))
	}
}

func healthCheckType(protocol *v1.HealthCheckProtocol) *string {
	if protocol == nil {
		return nil
	}

	switch *protocol {
	case v1.HealthCheckProtocolHTTP:
		return aws.String(route53.HealthCheckTypeHttp)

	case v1.HealthCheckProtocolHTTPS:
		return aws.String(route53.HealthCheckTypeHttps)
	}

	return nil
}

func strValuesEqual(str1, str2 *string) bool {
	if str1 == nil && str2 != nil {
		return false
	}
	if str2 == nil && str1 != nil {
		return false
	}
	if str1 == nil && str2 == nil {
		return true
	}

	return *str1 == *str2
}

func intValuesEqual(int1, int2 *int64) bool {
	if int1 == nil && int2 != nil ||
		int2 == nil && int1 != nil {
		return false
	}
	if int1 == nil && int2 == nil {
		return true
	}

	return *int1 == *int2
}

func getHealthCheckId(endpoint *v1.Endpoint) (string, bool) {
	return endpoint.GetProviderSpecific(ProviderSpecificHealthCheckID)
}
