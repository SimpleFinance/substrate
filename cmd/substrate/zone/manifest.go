package zone

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"
)

// SubstratePrivateSubnet is the IP CIDR block under which *all* environments
// and their zone will live. We want to avoid overlapping with the default IP
// space used byk8s (10.0.0.0/8) as well as the IP space of any of our other networks.
const SubstratePrivateSubnet = "172.16.0.0/12"

// SubstrateEnvironmentBits is the number of bits used to index environments
const SubstrateEnvironmentBits = 7

// SubstrateZoneBits is the number of bits used to index zones within a environment
const SubstrateZoneBits = 4

// SubstrateZoneManifest represents the on-disk structure of the Substrate zone manifest file
type SubstrateZoneManifest struct {
	Version             string      `json:"substrate_version"`
	EnvironmentName     string      `json:"environment_name"`
	EnvironmentDomain   string      `json:"environment_domain"`
	EnvironmentIndex    int         `json:"environment_index"`
	ZoneIndex           int         `json:"zone_index"`
	AWSAvailabilityZone string      `json:"aws_availability_zone"`
	AWSAccountID        string      `json:"aws_account_id"`
	DelegationSetID     string      `json:"delegation_set_id"`
	SSHPublicKey        string      `json:"ssh_public_key"`
	TerraformState      interface{} `json:"terraform_state"`
}

// AWSRegion returns the AWS region name of the zone (derived from the AZ name)
func (m *SubstrateZoneManifest) AWSRegion() string {
	return m.AWSAvailabilityZone[:len(m.AWSAvailabilityZone)-1]
}

// ZoneName returns the Substrate zone name for this zone (derived from the environment name and zone index)
func (m *SubstrateZoneManifest) ZoneName() string {
	return fmt.Sprintf("%s-%02d", m.EnvironmentName, m.ZoneIndex)
}

// ZonePrefix returns the prefix for the names of all the objects in the zone (derived from the environment name and zone index)
func (m *SubstrateZoneManifest) ZonePrefix() string {
	return fmt.Sprintf("substrate-%s-%02d", m.EnvironmentName, m.ZoneIndex)
}

// EnvironmentSubnet returns the IP subnet for all the IPs in this zone's environment
func (m *SubstrateZoneManifest) EnvironmentSubnet() string {
	_, network, err := net.ParseCIDR(SubstratePrivateSubnet)
	if err != nil {
		panic(err)
	}
	result, err := cidr.Subnet(network, SubstrateEnvironmentBits, m.EnvironmentIndex)
	if err != nil {
		panic(err)
	}
	return result.String()
}

// ZoneSubnet returns the IP subnet for all the IPs in this zone
func (m *SubstrateZoneManifest) ZoneSubnet() string {
	_, network, err := net.ParseCIDR(m.EnvironmentSubnet())
	if err != nil {
		panic(err)
	}
	result, err := cidr.Subnet(network, SubstrateZoneBits, m.ZoneIndex)
	if err != nil {
		panic(err)
	}
	return result.String()
}

// CloudWatchLogsGroupSystemLogs returns the name of the CloudWatch Logs group for this zone's system logs (derived from the zone name)
func (m *SubstrateZoneManifest) CloudWatchLogsGroupSystemLogs() string {
	return fmt.Sprintf("substrate-%s-system-logs", m.ZoneName())
}

// CloudWatchLogsGroupVPCFlowLogs returns the name of the CloudWatch Logs group for this zone's VPC Flow logs (derived from the zone name)
func (m *SubstrateZoneManifest) CloudWatchLogsGroupVPCFlowLogs() string {
	return fmt.Sprintf("substrate-%s-vpc-flow-logs", m.ZoneName())
}

// TFVars renders the zone settings into a `.tfvars` format (usable with Terraform's `-var-file` option)
func (m *SubstrateZoneManifest) TFVars() string {
	var result bytes.Buffer
	varMap := map[string]string{
		"substrate_version":                             m.Version,
		"substrate_environment":                         m.EnvironmentName,
		"substrate_environment_domain":                  m.EnvironmentDomain,
		"substrate_environment_subnet":                  m.EnvironmentSubnet(),
		"substrate_zone":                                m.ZoneName(),
		"zone_prefix":                                   m.ZonePrefix(),
		"substrate_zone_subnet":                         m.ZoneSubnet(),
		"substrate_cloudwatch_logs_group_system_logs":   m.CloudWatchLogsGroupSystemLogs(),
		"substrate_cloudwatch_logs_group_vpc_flow_logs": m.CloudWatchLogsGroupVPCFlowLogs(),
		"aws_region":                                    m.AWSRegion(),
		"aws_availability_zone":                         m.AWSAvailabilityZone,
		"aws_account_id":                                m.AWSAccountID,
		"delegation_set_id":                             m.DelegationSetID,
		"ssh_public_key":                                m.SSHPublicKey,
	}
	for k, v := range varMap {
		result.WriteString(fmt.Sprintf("%s = \"%s\"\n", k, v))
	}
	return result.String()
}

// ReadManifest reads a SubstrateZoneManifest from the file at the given path
func ReadManifest(path string) (*SubstrateZoneManifest, error) {
	marshalledJSON, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var result SubstrateZoneManifest
	err = json.Unmarshal(marshalledJSON, &result)
	if err != nil {
		return nil, err
	}

	if result.Version == "" {
		return nil, fmt.Errorf("expected to find `substrate_version` key in zone manifest %q", path)
	}

	return &result, nil
}
