package wipe

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

func autoscalingHasEnvTag(tags []*autoscaling.TagDescription, envName string) bool {
	for _, tag := range tags {
		if *tag.Key == "substrate:environment" && *tag.Value == envName {
			return true
		}
	}
	return false
}

func autoscalingNameFromTags(tags []*autoscaling.TagDescription) string {
	for _, tag := range tags {
		if *tag.Key == "Name" {
			return *tag.Value
		}
	}
	return ""
}

type autoscalingGroup struct {
	region string
	name   string
}

func (r *autoscalingGroup) String() string {
	return fmt.Sprintf("ASG %s in %s", r.name, r.region)
}

func (r *autoscalingGroup) Destroy() error {
	return fmt.Errorf("can't destroy ASGs yet")
}

func (r *autoscalingGroup) Priority() int {
	return 50
}

type autoscalingLaunchConfiguration struct {
	region string
	name   string
}

func (r *autoscalingLaunchConfiguration) String() string {
	return fmt.Sprintf("Launch configuration %s in %s", r.name, r.region)
}

func (r *autoscalingLaunchConfiguration) Destroy() error {
	return fmt.Errorf("can't destroy launch configurations yet")
}

func (r *autoscalingLaunchConfiguration) Priority() int {
	return 75
}

func discoverAutoscalingResources(region string, envName string) []destroyableResource {
	result := []destroyableResource{}
	svc := autoscaling.New(session.New(), &aws.Config{Region: aws.String(region)})

	fmt.Printf("scanning for autoscaling groups in %s...\n", region)
	groups, err := svc.DescribeAutoScalingGroups(nil)
	if err != nil {
		panic(err)
	}
	for _, group := range groups.AutoScalingGroups {
		if autoscalingHasEnvTag(group.Tags, envName) {
			result = append(result, &autoscalingGroup{
				region: region,
				name:   autoscalingNameFromTags(group.Tags),
			})
		}
	}

	fmt.Printf("scanning for launch configurations in %s...\n", region)
	configs, err := svc.DescribeLaunchConfigurations(nil)
	if err != nil {
		panic(err)
	}
	for _, config := range configs.LaunchConfigurations {
		if strings.Contains(*config.LaunchConfigurationName, envName) {
			result = append(result, &autoscalingLaunchConfiguration{
				region: region,
				name:   *config.LaunchConfigurationName,
			})
		}
	}

	return result
}
