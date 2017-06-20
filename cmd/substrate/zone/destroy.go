package zone

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/SimpleFinance/substrate/cmd/substrate/assets"
	"github.com/SimpleFinance/substrate/cmd/substrate/util"
)

// DestroyInput contains the input parameters for destroying a zone
type DestroyInput struct {
	Prompt       bool
	ManifestPath string
}

// Destroy reads an existing manifest, updates the zone in place, overwriting the manifest.
func Destroy(params *DestroyInput) error {
	// read the existing manifest
	zoneManifest, err := ReadManifest(params.ManifestPath)
	if err != nil {
		return err
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

	// run `terraform get` to install all our modules
	err = Terraform(extractedAssets, "get", "-no-color", "-update", "./zone")
	if err != nil {
		return err
	}

	// make sure the plan is legit before continuing
	if params.Prompt {
		err = util.Confirm("do you want to continue and destroy your zone?")
		if err != nil {
			return err
		}
	}

	// run `terraform plan` to generate an execution plan (.tfplan file)
	terraformDestroyErr := Terraform(
		extractedAssets,
		"destroy",
		"-force=true",
		"-no-color",
		"-input=false",
		"-parallelism=100",
		"-state", statePath,
		"-var-file", varsPath,
		"./zone")
	// on success, clean up the manifest
	if terraformDestroyErr == nil {
		return os.Remove(params.ManifestPath)
	}

	// keep going and save the .tfstate even if `terraform destroy` failed, so we don't orphan anything
	// if anything goes wrong past this point, bail out with a prompt to the user but don't clean up the
	// temp directory yet
	bail := func(err error, msg string) error {
		fmt.Printf("%s: %v\n\nTemporary directory (may hold clues): %s\n", msg, err, extractedAssets.Path(""))
		util.Confirm("I'll leave the temp directory around so you can clean up. Ready to delete it?")
		return err
	}

	// read the .tfstate file that `terraform destroy` should have generated
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

	return terraformDestroyErr
}
