package wipe

import (
	"fmt"
	"sort"

	"github.com/SimpleFinance/substrate/cmd/substrate/util"
)

// Input contains the input parameters for the wipe operation
type Input struct {
	Prompt          bool
	AWSRegions      []string
	EnvironmentName string
}

// Wipe searches for abandoned resources and destroys them
func Wipe(params *Input) error {
	resources := destroyableResources{}
	for _, region := range params.AWSRegions {
		resources = append(resources, discoverEC2Resources(region, params.EnvironmentName)...)
		resources = append(resources, discoverAutoscalingResources(region, params.EnvironmentName)...)
		resources = append(resources, discoverSQSResources(region, params.EnvironmentName)...)
		resources = append(resources, discoverS3Resources(region, params.EnvironmentName)...)
	}

	// IAM and Route53 resources always live in us-east-1
	resources = append(resources, discoverIAMResources(params.EnvironmentName)...)
	resources = append(resources, discoverRoute53Resources(params.EnvironmentName)...)

	if len(resources) == 0 {
		fmt.Printf("\nDidn't find any resources to wipe!\n\n")
		return nil
	}

	sort.Sort(resources)

	fmt.Printf("\nFound %d resources to wipe:\n", len(resources))
	for _, resource := range resources {
		fmt.Printf(" - %s\n", resource)
	}

	if params.Prompt {
		err := util.Confirm("Do you want to continue and delete all these resources?")
		if err != nil {
			return err
		}
	}
	fmt.Printf("\n")

	resourcesToDelete := len(resources)
	resourcesDeleted := 0

	for _, resource := range resources {
		fmt.Printf(" - destroying %s...", resource)
		err := resource.Destroy()
		if err == nil {
			resourcesDeleted++
			fmt.Printf("success\n")
		} else {
			fmt.Printf("failed\n    %s\n\n", err)
		}
	}

	if resourcesDeleted < resourcesToDelete {
		return fmt.Errorf(
			"failed to destroy %d of %d resources",
			resourcesToDelete-resourcesDeleted,
			resourcesToDelete)
	}
	return nil
}
