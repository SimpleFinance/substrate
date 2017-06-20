package zone

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/goware/prefixer"

	"github.com/SimpleFinance/substrate/cmd/substrate/assets"
)

// Terraform runs `terraform` in the extracted working directory
func Terraform(extractedAssets *assets.SubstrateAssets, arg ...string) error {
	tpath := extractedAssets.Path("bin/terraform")
	cmd := exec.Command(tpath, arg...)
	log.Printf("%s %s", tpath, strings.Join(arg[:], " "))
	cmd.Env = []string{
		fmt.Sprintf("PATH=%s", extractedAssets.Path("bin")),
	}

	// pass through AWS_* variables as well
	for _, envVar := range os.Environ() {
		if strings.HasPrefix(envVar, "AWS_") {
			cmd.Env = append(cmd.Env, envVar)
		}
	}

	// run it with the root of the temp directory as working directory
	cmd.Dir = extractedAssets.Path("")

	// stream stdout and strderr, but prefix each line of the output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	name := fmt.Sprintf("terraform %s", arg[0])
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(os.Stdout, prefixer.New(stdoutPipe, fmt.Sprintf("%s > ", name)))
	}()
	go func() {
		defer wg.Done()
		io.Copy(os.Stderr, prefixer.New(stderrPipe, fmt.Sprintf("%s ! ", name)))
	}()

	// wait for the subprocess to finish
	err = cmd.Wait()

	// then wait for both of the stdout/stderr copying goroutines to finish
	wg.Wait()

	// return the exit status of the subprocess
	return err
}
