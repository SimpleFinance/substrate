package zone

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/route53"

	"github.com/SimpleFinance/substrate/cmd/substrate/assets"
	"github.com/SimpleFinance/substrate/cmd/substrate/logwatcher"
	"github.com/SimpleFinance/substrate/cmd/substrate/util"
)

// CreateInput contains the input parameters for creating a zone
type CreateInput struct {
	Version             string
	Prompt              bool
	EnvironmentName     string
	EnvironmentDomain   string
	EnvironmentIndex    int
	ZoneIndex           int
	AWSAccountID        string
	AWSAvailabilityZone string
	OutputManifestPath  string
}

// Create spins up a new zone and saves the output into a manifest file
func Create(params *CreateInput) error {

	// fail out immediately if the output manifest already exists
	if _, err := os.Stat(params.OutputManifestPath); err == nil {
		return fmt.Errorf("zone manifest %v already exists, try `substrate zone update`?", params.OutputManifestPath)
	}

	// otherwise create the parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(params.OutputManifestPath), 0700); err != nil {
		return err
	}

	// then open the output file for writing so we can fail fast if it's not writable
	outputManifest, err := os.OpenFile(params.OutputManifestPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer outputManifest.Close()

	fmt.Printf(
		"creating zone %d in environment %q, writing zone manifest to %q...\n",
		params.ZoneIndex,
		params.EnvironmentName,
		params.OutputManifestPath)

	// create the initial manifest by filling in from parameters
	zoneManifest := &SubstrateZoneManifest{
		EnvironmentName:     params.EnvironmentName,
		EnvironmentDomain:   params.EnvironmentDomain,
		EnvironmentIndex:    params.EnvironmentIndex,
		ZoneIndex:           params.ZoneIndex,
		AWSAvailabilityZone: params.AWSAvailabilityZone,
		AWSAccountID:        params.AWSAccountID,
		Version:             params.Version,

		// TODO: this shouldn't be hardcoded (should probably just go away after we have a Bastion setup)
		SSHPublicKey: os.ExpandEnv("$HOME/.ssh/id_rsa.pub"),
	}

	// get or create the "substrate" Reusable Delegation Set in Route53
	// check if an NS lookup for `zoneXX.envdomain` in any suffix of `envdomain` points to the delegation set
	//   if not, and the `envdomain` Hosted Zone is in the current account
	//     - find the Hosted Zone for `envdomain` or a parent of `envdomain`
	//     - create an NS record set pointing at the delegation set we looked up
	//   if not, and the `envcomain` Hosted Zone is not in the account, error to the user describing the record to create
	//     "create an NS record for `zoneXX.envdomain` pointing at these 4 nameservers..."
	// print instructions about how to activate the zone by setting a wildcard CNAME at the worker name

	// this is the domain for the zone, we want make sure it's going to resolve to our delegation set once we make a real Hosted Zone
	zoneSubdomain := fmt.Sprintf("zone%02d.%s", zoneManifest.ZoneIndex, zoneManifest.EnvironmentDomain)

	// get/create the delegation set so we know what nameservers we _should_ see
	expectedNameservers, delegationSetID, err := util.GetOrCreateSubstrateReusableDelegationSet(
		route53.New(session.New(), &aws.Config{Region: aws.String(zoneManifest.AWSRegion())}),
	)
	if err != nil {
		return err
	}
	zoneManifest.DelegationSetID = delegationSetID

	// find the first suffix of zoneSubdomain that has working DNS
	suffix, suffixNameservers, err := util.FindFirstSuffixWithWorkingDNS(zoneSubdomain)
	if err != nil {
		return err
	}

	// see if the working level of DNS has the zone domain pointing at our delegation set
	actualNameservers, err := util.LookupNSUsingServer(zoneSubdomain, suffixNameservers[0])
	if err != nil {
		return err
	}

	// see if things are set correctly
	sort.Strings(expectedNameservers)
	sort.Strings(actualNameservers)
	if !util.StringSlicesEqual(expectedNameservers, actualNameservers) {
		fmt.Printf("\n\nIt appears you have some DNS issues to resolve (get it?)\n\n")

		fmt.Printf(
			"You should create NS records for %q in the zone %q pointing at nameservers:\n%s\n\n",
			strings.TrimSuffix(zoneSubdomain, "."+suffix),
			suffix,
			strings.Join(expectedNameservers, "\n"))

		suffixHostedZoneID, suffixHostedZoneExists, err := util.FindHostedZoneID(
			route53.New(session.New(), &aws.Config{Region: aws.String(zoneManifest.AWSRegion())}),
			suffix,
		)
		if err != nil {
			return err
		}

		// TODO: should check if the suffix hosted zone is actually the one we found in our suffix search (nameservers match)

		if suffixHostedZoneExists {
			fmt.Printf("Would you like to do this automatically? The domain %q exists in Route53 Hosted Zone %v in the current AWS account.\n", suffix, suffixHostedZoneID)
		} else {
			fmt.Printf("You're on your own for this one, sorry.\n")
		}
		return nil
	}

	fmt.Println("\n\nlooks like your DNS is ready to go!")

	// extract all the Terraform binaries/config into a temp directory
	extractedAssets, err := assets.ExtractSubstrateAssets()
	if err != nil {
		return nil
	}
	defer extractedAssets.Cleanup()

	statePath := extractedAssets.Path("substrate.tfstate")
	planPath := extractedAssets.Path("substrate.tfplan")
	varsPath := extractedAssets.Path("substrate.tfvars")

	// write the .tfvars file to pass parameters into Terraform
	err = ioutil.WriteFile(varsPath, []byte(zoneManifest.TFVars()), 0600)
	if err != nil {
		return nil
	}

	// print terraform version
	err = Terraform(extractedAssets, "version")
	if err != nil {
		return err
	}

	// run `terraform get` to install all our modules
	err = Terraform(extractedAssets, "get", "-no-color", "-update", "./zone")
	if err != nil {
		return err
	}

	// run `terraform plan` to generate an execution plan (.tfplan file)
	err = Terraform(
		extractedAssets,
		"plan",
		"-no-color",
		"-input=false",
		"-state", statePath,
		"-out", planPath,
		"-var-file", varsPath,
		"./zone")
	if err != nil {
		return err
	}

	// make sure the plan is legit before continuing
	if params.Prompt {
		err = util.Confirm("do you want to continue and apply this plan?")
		if err != nil {
			return err
		}
	}

	// start watching logs and dumping them out in a goroutine
	log := logwatcher.Start(
		cloudwatchlogs.New(
			session.New(),
			&aws.Config{Region: aws.String(zoneManifest.AWSRegion())}),
		zoneManifest.CloudWatchLogsGroupSystemLogs(),
	)
	// stream the log events to stdout in a background thread
	go func() {
		for event := range log.Events() {
			// only output our AMI provisioning events at INFO or higher
			if event.Record.Syslog.Identifier == "substrate-base-ami-provision" {
				if event.Record.Priority != "DEBUG" {
					fmt.Printf("base-ami-provision > %s\n", event.Record.Message)
				}
			}
		}
	}()
	// close the logwatcher before we return, outputting any errors we hit
	defer func() {
		err := log.Stop()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error watching logs: %v\n", err)
		}
	}()

	// pass the plan into `terraform apply` to create all the zone resources and dump out the resulting .tfstate file
	terraformApplyErr := Terraform(
		extractedAssets,
		"apply",
		"-no-color",
		"-refresh=false",
		"-parallelism=100",
		"-input=false",
		"-state", statePath,
		planPath)

	// keep going and save the .tfstate even if `terraform apply` failed, so we don't orphan anything
	// if anything goes wrong past this point, bail out with a prompt to the user but don't clean up the
	// temp directory yet
	bail := func(err error, msg string) error {
		if params.Prompt {
			fmt.Printf("%s: %v\n\nTemporary directory (may hold clues): %s\n", msg, err, extractedAssets.Path(""))
			util.Confirm("I'll leave the temp directory around so you can clean up. Ready to delete it?")
		}
		return err
	}

	// read the .tfstate file that `terraform apply` should have generated
	stateJSON, err := ioutil.ReadFile(statePath)
	if err != nil {
		return bail(err, "error reading .tfstate")
	}

	// parse it into the manifest
	err = json.Unmarshal(stateJSON, &zoneManifest.TerraformState)
	if err != nil {
		return bail(err, "error parsing .tfstate")
	}

	// render the output manifest to (indented) JSON
	zoneManifestJSON, err := json.MarshalIndent(zoneManifest, "", "    ")
	if err != nil {
		return bail(err, "error encoding zone manifest")
	}

	// output the final JSON to the already-opened manifest file
	_, err = io.Copy(outputManifest, bytes.NewBuffer(zoneManifestJSON))
	if err != nil {
		return bail(err, "error writing zone manifest")
	}

	return terraformApplyErr
}
