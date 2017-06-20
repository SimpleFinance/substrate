package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {
	// serve our provider
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return &schema.Provider{
				ResourcesMap: map[string]*schema.Resource{
					"bakery_ebs_snapshot": resourceEBSSnapshot(),
				},
			}
		},
	})
}

func resourceEBSSnapshot() *schema.Resource {
	return &schema.Resource{
		Create: resourceEBSSnapshotCreate,
		Read:   resourceEBSSnapshotRead,
		Delete: resourceEBSSnapshotDelete,

		Schema: map[string]*schema.Schema{
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Default:     "",
				Description: "A description for the snapshot.",
			},
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The AWS region in which to create the snapshot.",
			},
			"builder_subnet_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The subnet to use for networking.",
			},
			"builder_security_groups": &schema.Schema{
				Type:        schema.TypeSet,
				Optional:    true,
				ForceNew:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Set:         schema.HashString,
				Description: "The security groups to attach our instances to.",
			},
			"builder_source_ami": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !regexp.MustCompile(`^ami-[0-9a-f]+$`).MatchString(value) {
						es = append(es, fmt.Errorf("%q must be a valid AMI ID", k))
					}
					return
				},
				Description: "The source AMI to with which to boot the temporary builder instance.",
			},
			"builder_availability_zone": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The AWS AZ within which to start the temporary builder instance.",
			},
			"builder_instance_type": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "t2.medium",
				ForceNew:    true,
				Description: "Type of EC2 instance to use as temporary builder instance.",
			},
			"builder_instance_tags": &schema.Schema{
				Type:        schema.TypeMap,
				Optional:    true,
				ForceNew:    true,
				Description: "Tags to add to the temporary builder instance.",
			},
			"builder_user_data": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				StateFunc: func(v interface{}) string {
					switch v.(type) {
					case string:
						hash := sha256.Sum256([]byte(v.(string)))
						return hex.EncodeToString(hash[:])
					default:
						return ""
					}
				},
				ForceNew:    true,
				Description: "User data with which to start the builder instance. This should be a script or #cloud-config stanza that provisions the instance and then shuts it down.",
			},
			"builder_iam_instance_profile": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The name of an IAM instance profile to assign to the temporary builder instance.",
			},
			"tags": &schema.Schema{
				Type:        schema.TypeMap,
				Optional:    true,
				ForceNew:    true,
				Description: "Tags to add to the snapshot.",
			},
		},
	}
}

func connectEC2(session *session.Session, d *schema.ResourceData) *ec2.EC2 {
	// TODO: this should probably mirror the options in the builtin terraform-provider-aws
	return ec2.New(session, &aws.Config{
		Region: aws.String(d.Get("region").(string)),
	})
}

// tagResource tags the AWS resource specified by resourceID with the tags specified in the schema under keyName
func tagResource(d *schema.ResourceData, ec2svc *ec2.EC2, keyName string, resourceID string) error {
	// if the tags value wasn't specified in the resource config, we're done
	tagsValue, tagsValueExists := d.GetOk(keyName)
	if !tagsValueExists {
		return nil
	}

	// the tagsValue is actually a map of strings to interface{}
	tagMap := tagsValue.(map[string]interface{})

	// if the tags value is empty (no tags), we're done
	if len(tagMap) == 0 {
		return nil
	}

	// build an array of *ec2.Tag objects from the input map
	tags := make([]*ec2.Tag, 0, len(tagMap))
	for k, v := range tagMap {
		tags = append(tags, &ec2.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		})
	}

	// call AWS to add the tags to the resource
	_, err := ec2svc.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{&resourceID},
		Tags:      tags,
	})
	return err
}

func resourceEBSSnapshotCreate(d *schema.ResourceData, m interface{}) error {
	awssess := session.New()
	svc := connectEC2(awssess, d)

	strConf := func(k string) *string { return aws.String(d.Get(k).(string)) }

	var builderSecGroups []*string
	if v := d.Get("builder_security_groups"); v != nil {
		sgs := v.(*schema.Set).List()
		for _, v := range sgs {
			str := v.(string)
			builderSecGroups = append(builderSecGroups, aws.String(str))
		}
	}

	params := &ec2.RunInstancesInput{
		ImageId:  strConf("builder_source_ami"),
		MaxCount: aws.Int64(1),
		MinCount: aws.Int64(1),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &ec2.EbsBlockDevice{
					DeleteOnTermination: aws.Bool(true),
					VolumeSize:          aws.Int64(8),
					VolumeType:          aws.String("gp2"),
				},
			},
		},
		InstanceInitiatedShutdownBehavior: aws.String("stop"),
		InstanceType:                      strConf("builder_instance_type"),
		Placement: &ec2.Placement{
			AvailabilityZone: strConf("builder_availability_zone"),
		},
		NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
			&ec2.InstanceNetworkInterfaceSpecification{
				AssociatePublicIpAddress: aws.Bool(true),
				DeviceIndex:              aws.Int64(int64(0)),
				SubnetId:                 strConf("builder_subnet_id"),
				Groups:                   builderSecGroups,
			},
		},
		UserData: aws.String(base64.StdEncoding.EncodeToString(
			[]byte(d.Get("builder_user_data").(string)))),
	}

	// if builder_iam_instance_profile is configured, reference it by name (vs. ARN)
	if _, iamProfileSet := d.GetOk("builder_iam_instance_profile"); iamProfileSet {
		params.IamInstanceProfile = &ec2.IamInstanceProfileSpecification{
			Name: strConf("builder_iam_instance_profile"),
		}
	}

	runInstancesResp, err := svc.RunInstances(params)
	if err != nil {
		return err
	}

	// TODO: the temporary instance ID should go in the state file in case we die mid-run
	builderInstance := *runInstancesResp.Instances[0].InstanceId
	defer func() {
		log.Printf("terminating builder instance %s...", builderInstance)
		svc.TerminateInstances(&ec2.TerminateInstancesInput{
			InstanceIds: []*string{&builderInstance}})
	}()
	log.Printf("started instance %s as builder", builderInstance)

	// tag the builder if configured
	err = tagResource(d, svc, "builder_instance_tags", builderInstance)
	if err != nil {
		return err
	}

	waitForStoppedInstance := &resource.StateChangeConf{
		Target: []string{
			ec2.InstanceStateNameStopped,
		},
		Pending: []string{
			ec2.InstanceStateNamePending,
			ec2.InstanceStateNameRunning,
			ec2.InstanceStateNameStopping,
		},
		Timeout:    15 * time.Minute,
		MinTimeout: 5 * time.Second,
		Refresh: func() (interface{}, string, error) {
			describeInstancesResp, err := svc.DescribeInstances(
				&ec2.DescribeInstancesInput{
					InstanceIds: []*string{&builderInstance},
				})
			if err != nil {
				return nil, "", err
			}

			if len(describeInstancesResp.Reservations) != 1 {
				return nil, "", nil
			}
			reservation := describeInstancesResp.Reservations[0]

			if len(reservation.Instances) != 1 {
				return nil, "", nil
			}
			instance := describeInstancesResp.Reservations[0].Instances[0]

			return instance, *instance.State.Name, nil
		},
	}
	stoppedInstance, err := waitForStoppedInstance.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for builder instance to stop (%s): %s", builderInstance, err)
	}
	rootVolumeID := *stoppedInstance.(*ec2.Instance).BlockDeviceMappings[0].Ebs.VolumeId

	// create snapshot of root volume
	log.Printf("root volume ID: %s", rootVolumeID)

	createSnapshotResp, err := svc.CreateSnapshot(
		&ec2.CreateSnapshotInput{
			VolumeId:    aws.String(rootVolumeID),
			Description: strConf("description"),
		})
	if err != nil {
		return err
	}
	snapshotID := *createSnapshotResp.SnapshotId
	d.SetId(snapshotID)

	// tag the snapshot if configured
	err = tagResource(d, svc, "tags", snapshotID)
	if err != nil {
		return err
	}

	waitForSnapshotReady := &resource.StateChangeConf{
		Target:     []string{ec2.SnapshotStateCompleted},
		Pending:    []string{ec2.SnapshotStatePending},
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Refresh: func() (interface{}, string, error) {
			describeSnapshotsResp, err := svc.DescribeSnapshots(
				&ec2.DescribeSnapshotsInput{
					SnapshotIds: []*string{aws.String(snapshotID)},
				})
			if err != nil {
				return nil, "", err
			}
			if len(describeSnapshotsResp.Snapshots) != 1 {
				return nil, "", nil
			}
			snapshot := describeSnapshotsResp.Snapshots[0]
			return *snapshot.SnapshotId, *snapshot.State, nil
		},
	}
	_, err = waitForSnapshotReady.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for snapshot (%s) to be ready: %s", snapshotID, err)
	}
	return nil
}

func resourceEBSSnapshotRead(d *schema.ResourceData, m interface{}) error {
	// TODO: we should at least detect the case where the snapshot has been deleted
	return nil
}

func resourceEBSSnapshotDelete(d *schema.ResourceData, m interface{}) error {
	awssess := session.New()
	_, err := connectEC2(awssess, d).DeleteSnapshot(&ec2.DeleteSnapshotInput{
		SnapshotId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}
