package wipe

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func ec2HasEnvTag(tags []*ec2.Tag, envName string) bool {
	for _, tag := range tags {
		if *tag.Key == "substrate:environment" && *tag.Value == envName {
			return true
		}
	}
	return false
}

func ec2NameFromTags(tags []*ec2.Tag) string {
	for _, tag := range tags {
		if *tag.Key == "Name" {
			return *tag.Value
		}
	}
	return ""
}

type ec2Instance struct {
	svc    *ec2.EC2
	region string
	id     string
	name   string
}

func (r *ec2Instance) String() string {
	return fmt.Sprintf("EC2 instance %s in %s (%s)", r.name, r.region, r.id)
}

func (r *ec2Instance) Destroy() error {
	_, err := r.svc.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{&r.id},
	})
	return err
}

func (r *ec2Instance) Priority() int {
	return 100
}

type ec2EIP struct {
	svc      *ec2.EC2
	region   string
	id       string
	publicIP string
}

func (r *ec2EIP) String() string {
	return fmt.Sprintf("EIP allocation %s in %s (%s)", r.publicIP, r.region, r.id)
}

func (r *ec2EIP) Destroy() error {
	_, err := r.svc.ReleaseAddress(&ec2.ReleaseAddressInput{
		PublicIp: aws.String(r.publicIP),
	})
	return err
}

func (r *ec2EIP) Priority() int {
	return 99
}

type securityGroup struct {
	svc    *ec2.EC2
	region string
	id     string
	name   string
}

func (r *securityGroup) String() string {
	return fmt.Sprintf("Security group %s in %s (%s)", r.name, r.region, r.id)
}

func (r *securityGroup) Destroy() error {
	_, err := r.svc.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(r.id),
	})
	return err
}

func (r *securityGroup) Priority() int {
	return 110
}

type destroyableVPC struct {
	svc    *ec2.EC2
	region string
	id     string
	name   string
}

func (r *destroyableVPC) String() string {
	return fmt.Sprintf("VPC %s in %s (%s)", r.name, r.region, r.id)
}

// routeTableIsMain returns true iff the route table has an association as the "main" route table for a VPC
func routeTableIsMain(routeTable *ec2.RouteTable) bool {
	for _, association := range routeTable.Associations {
		if *association.Main {
			return true
		}
	}
	return false
}

func (r *destroyableVPC) Destroy() error {
	// find all the route tables associated with this VPC
	routeTables, err := r.svc.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("vpc-id"),
				Values: []*string{&r.id},
			},
		},
	})
	if err != nil {
		return err
	}

	// delete all of unless they are associated as the "main" table
	for _, routeTable := range routeTables.RouteTables {
		if routeTableIsMain(routeTable) {
			continue
		}
		_, err := r.svc.DeleteRouteTable(&ec2.DeleteRouteTableInput{
			RouteTableId: routeTable.RouteTableId,
		})
		if err != nil {
			return err
		}
	}

	// after all the non-main route tables are gone, we can delete the VPC itself
	_, err = r.svc.DeleteVpc(&ec2.DeleteVpcInput{
		VpcId: &r.id,
	})
	return err
}

func (r *destroyableVPC) Priority() int {
	return 200
}

type destroyableVPCSubnet struct {
	svc    *ec2.EC2
	region string
	id     string
	name   string
}

func (r *destroyableVPCSubnet) String() string {
	return fmt.Sprintf("VPC subnet %s in %s (%s)", r.name, r.region, r.id)
}

func (r *destroyableVPCSubnet) Destroy() error {
	_, err := r.svc.DeleteSubnet(&ec2.DeleteSubnetInput{
		SubnetId: &r.id,
	})
	return err
}

func (r *destroyableVPCSubnet) Priority() int {
	return 150
}

type destroyableVPCInternetGateway struct {
	svc    *ec2.EC2
	region string
	id     string
	vpcID  string
	name   string
}

func (r *destroyableVPCInternetGateway) String() string {
	return fmt.Sprintf("Internet gateway %s in %s (%s)", r.name, r.region, r.id)
}

func (r *destroyableVPCInternetGateway) Destroy() error {
	_, err := r.svc.DetachInternetGateway(&ec2.DetachInternetGatewayInput{
		InternetGatewayId: &r.id,
		VpcId:             &r.vpcID,
	})
	if err != nil {
		return err
	}

	_, err = r.svc.DeleteInternetGateway(&ec2.DeleteInternetGatewayInput{
		InternetGatewayId: &r.id,
	})
	return err
}

func (r *destroyableVPCInternetGateway) Priority() int {
	return 195
}

type destroyableVPCDHCPOptions struct {
	svc    *ec2.EC2
	region string
	id     string
	name   string
}

func (r *destroyableVPCDHCPOptions) String() string {
	return fmt.Sprintf("VPC DHCP options %s in %s (%s)", r.name, r.region, r.id)
}

func (r *destroyableVPCDHCPOptions) Destroy() error {
	_, err := r.svc.DeleteDhcpOptions(&ec2.DeleteDhcpOptionsInput{
		DhcpOptionsId: &r.id,
	})
	return err
}

func (r *destroyableVPCDHCPOptions) Priority() int {
	return 300
}

type ec2KeyPair struct {
	svc    *ec2.EC2
	region string
	name   string
}

func (r *ec2KeyPair) String() string {
	return fmt.Sprintf("SSH key pair %s in %s", r.name, r.region)
}

func (r *ec2KeyPair) Destroy() error {
	_, err := r.svc.DeleteKeyPair(&ec2.DeleteKeyPairInput{
		KeyName: aws.String(r.name),
	})
	return err
}

func (r *ec2KeyPair) Priority() int {
	return 190
}

type ec2AMI struct {
	region string
	id     string
	name   string
}

func (r *ec2AMI) String() string {
	return fmt.Sprintf("AMI %s in %s (%s)", r.name, r.region, r.id)
}

func (r *ec2AMI) Destroy() error {
	return fmt.Errorf("can't destroy AMIs yet")
}

func (r *ec2AMI) Priority() int {
	return 140
}

type ec2EBSSnapshot struct {
	region string
	id     string
	name   string
}

func (r *ec2EBSSnapshot) String() string {
	return fmt.Sprintf("EBS snapshot %s in %s (%s)", r.name, r.region, r.id)
}

func (r *ec2EBSSnapshot) Destroy() error {
	return fmt.Errorf("can't destroy EBS snapshots yet")
}

func (r *ec2EBSSnapshot) Priority() int {
	return 150
}

func discoverEC2Resources(region string, envName string) []destroyableResource {
	result := []destroyableResource{}
	svc := ec2.New(session.New(), &aws.Config{Region: aws.String(region)})

	instances, err := svc.DescribeInstances(nil)
	if err != nil {
		panic(err)
	}

	publicIPs := []*string{}

	fmt.Printf("scanning for EC2 instances in %s...\n", region)
	for idx := range instances.Reservations {
		for _, inst := range instances.Reservations[idx].Instances {
			if *inst.State.Name == ec2.InstanceStateNameTerminated {
				continue
			}

			if ec2HasEnvTag(inst.Tags, envName) {
				for _, iface := range inst.NetworkInterfaces {
					if iface.Association != nil && iface.Association.PublicIp != nil {
						publicIPs = append(publicIPs, iface.Association.PublicIp)
					}
				}
				result = append(result, &ec2Instance{
					svc:    svc,
					region: region,
					id:     *inst.InstanceId,
					name:   ec2NameFromTags(inst.Tags),
				})
			}
		}
	}

	fmt.Printf("scanning for EIP allocations in %s...\n", region)
	for _, publicIP := range publicIPs {
		describeAddrsResp, err := svc.DescribeAddresses(&ec2.DescribeAddressesInput{
			PublicIps: []*string{publicIP},
		})
		if err != nil {
			continue
		}

		for _, addr := range describeAddrsResp.Addresses {
			result = append(result, &ec2EIP{
				svc:      svc,
				region:   region,
				id:       *addr.AllocationId,
				publicIP: *addr.PublicIp,
			})
		}
	}

	fmt.Printf("scanning for SGs in %s...\n", region)
	sgs, err := svc.DescribeSecurityGroups(nil)
	if err != nil {
		panic(err)
	}
	for _, sg := range sgs.SecurityGroups {
		if ec2HasEnvTag(sg.Tags, envName) {
			result = append(result, &securityGroup{
				svc:    svc,
				region: region,
				id:     *sg.GroupId,
				name:   ec2NameFromTags(sg.Tags),
			})
		}
	}

	fmt.Printf("scanning for VPCs in %s...\n", region)
	vpcs, err := svc.DescribeVpcs(nil)
	if err != nil {
		panic(err)
	}
	for _, vpc := range vpcs.Vpcs {
		if ec2HasEnvTag(vpc.Tags, envName) {
			result = append(result, &destroyableVPC{
				svc:    svc,
				region: region,
				id:     *vpc.VpcId,
				name:   ec2NameFromTags(vpc.Tags),
			})
		}
	}

	fmt.Printf("scanning for subnets in %s...\n", region)
	subnets, err := svc.DescribeSubnets(nil)
	if err != nil {
		panic(err)
	}
	for _, subnet := range subnets.Subnets {
		if ec2HasEnvTag(subnet.Tags, envName) {
			result = append(result, &destroyableVPCSubnet{
				svc:    svc,
				region: region,
				id:     *subnet.SubnetId,
				name:   ec2NameFromTags(subnet.Tags),
			})
		}
	}

	fmt.Printf("scanning for internet gateways in %s...\n", region)
	gateways, err := svc.DescribeInternetGateways(nil)
	if err != nil {
		panic(err)
	}
	for _, gateway := range gateways.InternetGateways {
		if ec2HasEnvTag(gateway.Tags, envName) {
			if len(gateway.Attachments) != 1 {
				panic(fmt.Errorf(
					"Don't know how to handle unattached internet gateway %q in %q",
					gateway.InternetGatewayId,
					region))
			}

			result = append(result, &destroyableVPCInternetGateway{
				svc:    svc,
				region: region,
				id:     *gateway.InternetGatewayId,
				vpcID:  *gateway.Attachments[0].VpcId,
				name:   ec2NameFromTags(gateway.Tags),
			})
		}
	}

	fmt.Printf("scanning for DHCP option configurations in %s...\n", region)
	dhcpOptions, err := svc.DescribeDhcpOptions(nil)
	if err != nil {
		panic(err)
	}
	for _, dhcp := range dhcpOptions.DhcpOptions {
		if ec2HasEnvTag(dhcp.Tags, envName) {
			result = append(result, &destroyableVPCDHCPOptions{
				svc:    svc,
				region: region,
				id:     *dhcp.DhcpOptionsId,
				name:   ec2NameFromTags(dhcp.Tags),
			})
		}
	}

	fmt.Printf("scanning for SSH keypairs in %s...\n", region)
	keypairs, err := svc.DescribeKeyPairs(nil)
	if err != nil {
		panic(err)
	}

	for _, keypair := range keypairs.KeyPairs {
		if strings.Contains(*keypair.KeyName, envName) {
			result = append(result, &ec2KeyPair{
				svc:    svc,
				region: region,
				name:   *keypair.KeyName,
			})
		}

	}

	fmt.Printf("scanning for AMIs in %s...\n", region)
	images, err := svc.DescribeImages(nil)
	if err != nil {
		panic(err)
	}
	for _, image := range images.Images {
		if ec2HasEnvTag(image.Tags, envName) {
			result = append(result, &ec2AMI{
				region: region,
				id:     *image.ImageId,
				name:   ec2NameFromTags(image.Tags),
			})
		}
	}

	fmt.Printf("scanning for EBS snapshots in %s...\n", region)
	snapshots, err := svc.DescribeSnapshots(nil)
	if err != nil {
		panic(err)
	}
	for _, snapshot := range snapshots.Snapshots {
		if ec2HasEnvTag(snapshot.Tags, envName) {
			result = append(result, &ec2EBSSnapshot{
				region: region,
				id:     *snapshot.SnapshotId,
				name:   ec2NameFromTags(snapshot.Tags),
			})
		}
	}
	return result
}
