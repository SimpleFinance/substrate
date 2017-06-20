package util

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

// SubstrateDelegationSetNamePrefix is the prefix for the "CallerReference" of the Route53 Reusable Delegation Set we want to use for all Substrate operations.
const SubstrateDelegationSetNamePrefix = "substrate-"

func convertToSortedStringArray(input []*string) []string {
	result := make([]string, len(input))
	for i := range input {
		result[i] = *input[i]
	}
	return result
}

// GetOrCreateSubstrateReusableDelegationSet looks up a Route53 Reusable Delegation Set called "substrate" and returns the nameservers names it's hosted on as well as its ID. If no such Delegation Set exists, it creates one.
func GetOrCreateSubstrateReusableDelegationSet(svc *route53.Route53) ([]string, string, error) {

	// first, we look for an existing Delegation Set that matches our name, returning the nameservers if we find it
	var params route53.ListReusableDelegationSetsInput
	for {
		resp, err := svc.ListReusableDelegationSets(&params)
		if err != nil {
			return []string{}, "", fmt.Errorf("error looking for the right Route53 Reusable Delegation Set: %v", err)
		}

		// if one of the Delegation Sets in this page matches our name, return its nameservers
		for _, ds := range resp.DelegationSets {
			if strings.HasPrefix(*ds.CallerReference, SubstrateDelegationSetNamePrefix) {
				nsArray := convertToSortedStringArray(ds.NameServers)
				dsID := strings.TrimPrefix(*ds.Id, "/delegationset/")
				return nsArray, dsID, nil
			}
		}

		// if there are no more pages, we haven't found what we're looking for
		if !*resp.IsTruncated {
			break
		}

		// otherwise move on to the next page
		params.Marker = resp.NextMarker
	}

	// if we didn't find it, we'll make one

	// generate a random caller reference that starts with our chosen prefix
	callerReference := SubstrateDelegationSetNamePrefix + RandomHex(8)

	// create the new delegation set
	resp, err := svc.CreateReusableDelegationSet(&route53.CreateReusableDelegationSetInput{
		CallerReference: aws.String(callerReference),
	})
	if err != nil {
		return []string{}, "", fmt.Errorf("error creating a Route53 Reusable Delegation Set: %v", err)
	}

	// return the nameserver entries and ID of the new delegation set
	nsArray := convertToSortedStringArray(resp.DelegationSet.NameServers)
	dsID := strings.TrimPrefix(*resp.DelegationSet.Id, "/delegationset/")
	return nsArray, dsID, nil
}

// FindHostedZoneID finds the Route53 Hosted Zone ID for any zone hosting the specified domain. Returns the Hosted Zone ID, a boolean indicating whether one was found, or an error if something bad happens.
func FindHostedZoneID(svc *route53.Route53, domain string) (string, bool, error) {

	resp, err := svc.ListHostedZonesByName(&route53.ListHostedZonesByNameInput{
		DNSName:  aws.String(domain),
		MaxItems: aws.String("1"),
	})
	if err != nil {
		return "", false, err
	}

	for _, hostedZone := range resp.HostedZones {
		if strings.TrimSuffix(*hostedZone.Name, ".") == domain {
			return *hostedZone.Id, true, nil
		}
	}

	return "", false, nil
}
