/*
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

package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/route53"
)

type InstrumentedRoute53 struct {
	*route53.Route53
}

func observe(operation string, f func() error) {
	start := time.Now()
	route53RequestCount.WithLabelValues(operation).Inc()
	defer route53RequestCount.WithLabelValues(operation).Dec()
	err := f()
	duration := time.Since(start).Seconds()
	label := resultSucceededLabel
	if err != nil {
		route53RequestErrors.WithLabelValues(operation).Inc()
		label = resultFailedLabel
	}
	route53RequestDuration.WithLabelValues(operation, label).Observe(duration)
	route53RequestTotal.WithLabelValues(operation, label).Inc()
}

func (c *InstrumentedRoute53) ListHostedZones(input *route53.ListHostedZonesInput) (output *route53.ListHostedZonesOutput, err error) {
	observe("ListHostedZones", func() error {
		output, err = c.Route53.ListHostedZones(input)
		return err
	})
	return
}

func (c *InstrumentedRoute53) ChangeResourceRecordSets(input *route53.ChangeResourceRecordSetsInput) (output *route53.ChangeResourceRecordSetsOutput, err error) {
	observe("ChangeResourceRecordSets", func() error {
		output, err = c.Route53.ChangeResourceRecordSets(input)
		return err
	})
	return
}

func (c *InstrumentedRoute53) CreateHealthCheck(input *route53.CreateHealthCheckInput) (output *route53.CreateHealthCheckOutput, err error) {
	observe("CreateHealthCheck", func() error {
		output, err = c.Route53.CreateHealthCheck(input)
		return err
	})
	return
}

func (c *InstrumentedRoute53) GetHealthCheckWithContext(ctx aws.Context, input *route53.GetHealthCheckInput, opts ...request.Option) (output *route53.GetHealthCheckOutput, err error) {
	observe("GetHealthCheckWithContext", func() error {
		output, err = c.Route53.GetHealthCheckWithContext(ctx, input, opts...)
		return err
	})
	return
}

func (c *InstrumentedRoute53) UpdateHealthCheckWithContext(ctx aws.Context, input *route53.UpdateHealthCheckInput, opts ...request.Option) (output *route53.UpdateHealthCheckOutput, err error) {
	observe("UpdateHealthCheckWithContext", func() error {
		output, err = c.Route53.UpdateHealthCheckWithContext(ctx, input, opts...)
		return err
	})
	return
}

func (c *InstrumentedRoute53) DeleteHealthCheckWithContext(ctx aws.Context, input *route53.DeleteHealthCheckInput, opts ...request.Option) (output *route53.DeleteHealthCheckOutput, err error) {
	observe("DeleteHealthCheckWithContext", func() error {
		output, err = c.Route53.DeleteHealthCheckWithContext(ctx, input, opts...)
		return err
	})
	return
}

func (c *InstrumentedRoute53) ChangeTagsForResourceWithContext(ctx aws.Context, input *route53.ChangeTagsForResourceInput, opts ...request.Option) (output *route53.ChangeTagsForResourceOutput, err error) {
	observe("ChangeTagsForResourceWithContext", func() error {
		output, err = c.Route53.ChangeTagsForResourceWithContext(ctx, input, opts...)
		return err
	})
	return
}
