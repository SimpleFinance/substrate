package wipe

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

func route53HasEnvTag(tags []*route53.Tag, envName string) bool {
	for _, tag := range tags {
		if *tag.Key == "substrate:environment" && *tag.Value == envName {
			return true
		}
	}
	return false
}

type route53HostedZone struct {
	svc           *route53.Route53
	name          string
	id            string
	resourceCount int64
}

func (r *route53HostedZone) String() string {
	// the numeric part of the ID is after the final "/"
	parts := strings.Split(r.id, "/")
	numericID := parts[len(parts)-1]
	return fmt.Sprintf(
		"Route53 hosted zone %s with %d records (%s)",
		r.name,
		r.resourceCount,
		numericID)
}

func (r *route53HostedZone) Destroy() error {
	// first we need to find all the non-SOA/NS records so we can delete them in a batch
	changeBatch := &route53.ChangeBatch{}
	err := r.svc.ListResourceRecordSetsPages(
		&route53.ListResourceRecordSetsInput{
			HostedZoneId: &r.id,
		},
		func(page *route53.ListResourceRecordSetsOutput, lastPage bool) bool {
			for _, resource := range page.ResourceRecordSets {
				if *resource.Type != "SOA" && *resource.Type != "NS" {
					changeBatch.Changes = append(changeBatch.Changes, &route53.Change{
						Action:            aws.String("DELETE"),
						ResourceRecordSet: resource,
					})
				}
			}
			return true
		})
	if err != nil {
		return err
	}

	// apply the batch delete, hopefully leaving only the NS and SOA records (which are special)
	_, err = r.svc.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  changeBatch,
		HostedZoneId: &r.id,
	})
	if err != nil {
		return err
	}

	// finally, delete the entire zone (including the SOA and NS records)
	_, err = r.svc.DeleteHostedZone(&route53.DeleteHostedZoneInput{
		Id: &r.id,
	})
	return err
}

func (r *route53HostedZone) Priority() int {
	return 50
}

func discoverRoute53Resources(envName string) []destroyableResource {
	result := []destroyableResource{}
	svc := route53.New(session.New())

	fmt.Printf("scanning for Route53 hosted zones...\n")
	err := svc.ListHostedZonesPages(&route53.ListHostedZonesInput{},
		func(page *route53.ListHostedZonesOutput, lastPage bool) bool {
			for _, zone := range page.HostedZones {
				tags, err := svc.ListTagsForResource(&route53.ListTagsForResourceInput{
					ResourceId:   zone.Id,
					ResourceType: aws.String("hostedzone"),
				})
				if err != nil {
					panic(err)
				}
				if route53HasEnvTag(tags.ResourceTagSet.Tags, envName) {
					result = append(result, &route53HostedZone{
						svc:           svc,
						name:          *zone.Name,
						id:            *zone.Id,
						resourceCount: *zone.ResourceRecordSetCount,
					})
				}
			}
			return true
		})
	if err != nil {
		panic(err)
	}
	return result
}
