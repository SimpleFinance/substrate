package zone

import (
	"fmt"
	"os"
	"os/exec"
)

// SSHInput contains the input parameters for SSHing
type SSHInput struct {
	ManifestPath string
	Host         string
	Args         []string
}

func getTerraformOutput(name string, tfState interface{}) (string, error) {
	tfStateMap, ok := tfState.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("malformed Terraform state")
	}

	modules, ok := tfStateMap["modules"]
	if !ok {
		return "", fmt.Errorf("couldn't find \"modules\" list in Terraform state")
	}

	modulesList, ok := modules.([]interface{})
	if !ok {
		return "", fmt.Errorf("malformed modules list in Terraform state")
	}

	if len(modulesList) < 1 {
		return "", fmt.Errorf("empty modules list in Terraform state")
	}

	rootModuleMap, ok := modulesList[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("malformed root module in Terraform state")
	}

	outputs, ok := rootModuleMap["outputs"]
	if !ok {
		return "", fmt.Errorf("couldn't find \"outputs\" map in root module of Terraform state")
	}

	outputsMap, ok := outputs.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("malformed \"outputs\" map in root module of Terraform state")
	}

	specificOutput, ok := outputsMap[name]
	if !ok {
		return "", fmt.Errorf("could not find output %q in root module of Terraform state", name)
	}

	specificOutputMap, ok := specificOutput.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("malformed specific output map in root module of Terraform state")
	}

	result, ok := specificOutputMap["value"]
	if !ok {
		return "", fmt.Errorf("malformed specific output map missing \"value\" field in root module of Terraform state")
	}

	resultString, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("output %q in root module of Terraform state was not a string", name)
	}

	return resultString, nil
}

// SSH connects to a zone instance over SSH
func SSH(params *SSHInput) error {
	// read the manifest
	zoneManifest, err := ReadManifest(params.ManifestPath)
	if err != nil {
		return err
	}

	borderEIP, err := getTerraformOutput("border_eip", zoneManifest.TerraformState)
	if err != nil {
		return err
	}

	args := []string{
		"-q",
		"-o",
		fmt.Sprintf(
			"ProxyCommand=ssh -q -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -W %%h:%%p ubuntu@%s", borderEIP),
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("ubuntu@%s", params.Host),
	}
	args = append(args, params.Args...)

	cmd := exec.Command("ssh", args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
