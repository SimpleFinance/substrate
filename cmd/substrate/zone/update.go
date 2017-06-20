package zone

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"

	"github.com/SimpleFinance/substrate/cmd/substrate/assets"
	"github.com/SimpleFinance/substrate/cmd/substrate/logwatcher"
	"github.com/SimpleFinance/substrate/cmd/substrate/util"
)

// UpdateInput contains the input parameters for updating a zone
type UpdateInput struct {
	Version       string
	Prompt        bool
	ManifestPath  string
	UnsafeUpgrade bool
}

// IsCompatibleUpgrade takes an old version number and a current version number
// and returns nil if an in-place upgrade is possible, or else an error describing why not
func IsCompatibleUpgrade(old string, new string) error {
	if old == new {
		return nil
	}
	// TODO: allow patch/minor version bumps
	return fmt.Errorf("in-place upgrade from Substrate %s to %s is not supported", old, new)
}

// Update reads an existing manifest, updates the zone in place, overwriting the manifest.
func Update(params *UpdateInput) error {
	zoneManifest, err := ReadManifest(params.ManifestPath)
	if err != nil {
		return err
	}

	// check version compatibility
	err = IsCompatibleUpgrade(zoneManifest.Version, params.Version)
	if err != nil {
		if params.UnsafeUpgrade {
			fmt.Printf("Warning: %v. Continuing anyway because of --unsafe...\n", err)
		} else {
			return fmt.Errorf("%v. You need to destroy and recreate the zone (or bypass this check with --unsafe if you are sure you want to do this)", err)
		}
	}

	// extract all the Terraform binaries/config into a temp directory
	extractedAssets, err := assets.ExtractSubstrateAssets()
	if err != nil {
		return nil
	}
	defer extractedAssets.Cleanup()

	// extract the saved .tfstate from the manifest
	stateJSON, err := json.MarshalIndent(zoneManifest.TerraformState, "", "    ")
	if err != nil {
		return err
	}

	// write the .tfstate
	statePath := extractedAssets.Path("substrate.tfstate")
	err = ioutil.WriteFile(statePath, stateJSON, 0600)
	if err != nil {
		return err
	}

	// write the .tfvars file to pass parameters into Terraform
	varsPath := extractedAssets.Path("substrate.tfvars")
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
	planPath := extractedAssets.Path("substrate.tfplan")
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
	updatedStateJSON, err := ioutil.ReadFile(statePath)
	if err != nil {
		return bail(err, "error reading .tfstate")
	}

	// parse it into the manifest
	err = json.Unmarshal(updatedStateJSON, &zoneManifest.TerraformState)
	if err != nil {
		return bail(err, "error parsing .tfstate")
	}

	// render the output manifest to (indented) JSON
	updatedZoneManifestJSON, err := json.MarshalIndent(zoneManifest, "", "    ")
	if err != nil {
		return bail(err, "error encoding zone manifest")
	}

	err = os.Rename(params.ManifestPath, params.ManifestPath+".bak")
	if err != nil {
		return bail(err, "saving backup zone manifest")
	}

	err = ioutil.WriteFile(params.ManifestPath, updatedZoneManifestJSON, 0600)
	if err != nil {
		return bail(err, "saving updated zone manifest")
	}

	return terraformApplyErr
}
