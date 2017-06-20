package main

import (
	"fmt"
	"os"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/SimpleFinance/substrate/cmd/substrate/util"
	"github.com/SimpleFinance/substrate/cmd/substrate/wipe"
	"github.com/SimpleFinance/substrate/cmd/substrate/zone"
)

var version string
var commit string
var defaultEnvironment = fmt.Sprintf("%s-dev", util.CurrentUser())

var defaultManifest = os.ExpandEnv("$HOME/.substrate/default-zone.json")
var defaultUser = "ubuntu"

var (
	app = kingpin.New(
		"substrate",
		"Substrate: a tool for provisioning lower level infrastructure",
	).Version(fmt.Sprintf("Substrate %s (commit %s)", version, commit))

	// I'd rather this showed up as --no-prompt directly in the --help output,
	// but we hit this bug https://github.com/alecthomas/kingpin/issues/54
	prompt = app.Flag(
		"prompt",
		"prompt before anything potentially destructive (default: true, disable with --no-prompt)",
	).Default("true").Bool()
)

var (
	zoneCommand = app.Command("zone", "commands for working with zones")
)

// `substrate zone create` command and options
var (
	createCommand = zoneCommand.Command("create", "create a new zone")

	createEnvironmentName = EnvironmentName(createCommand.Flag(
		"environment",
		"Environment name. Defaults to \"$(whoami)-dev\".",
	).PlaceHolder("ENV").Envar("SUBSTRATE_ENVIRONMENT").Default(defaultEnvironment))

	createEnvironmentDomain = EnvironmentDomain(createCommand.Flag(
		"environment-domain",
		"Environment domain name.",
	).PlaceHolder("ENV").Envar("SUBSTRATE_ENVIRONMENT_DOMAIN").Required())

	createEnvironmentIndex = EnvironmentIndex(createCommand.Flag(
		"environment-index",
		"Numeric index of the environment (0-127). Defaults to 127.",
	).PlaceHolder("N").Envar("SUBSTRATE_ENVIRONMENT_INDEX").Default("127"))

	createZoneIndex = ZoneIndex(createCommand.Flag(
		"zone-index",
		"numeric index of zone within the environment (0-15). Defaults to 0.",
	).PlaceHolder("M").Envar("SUBSTRATE_ZONE_INDEX").Default("0"))

	createAvailabilityZone = createCommand.Flag(
		"aws-availability-zone",
		"AWS availability zone in which to create the zone. Defaults to \"us-west-2a\".",
	).PlaceHolder("AZ").Default("us-west-2a").String()

	createAWSAccountID = createCommand.Flag(
		"aws-account-id",
		"AWS account ID (e.g., \"123456789012\"). Defaults to $AWS_ACCOUNT_ID.",
	).PlaceHolder("ID").Envar("AWS_ACCOUNT_ID").Required().String()

	createManifestOut = createCommand.Flag(
		"manifest",
		"output path for new zone manifest file",
	).Default(defaultManifest).String()
)

var (
	updateCommand       = zoneCommand.Command("update", "update a zone")
	updateUnsafeUpgrade = updateCommand.Flag(
		"unsafe",
		"force an in-place upgrade even when it may not be safe",
	).Bool()
	updateManifestPath = updateCommand.Flag(
		"manifest",
		"path to zone manifest (will be overwritten with updated manifest)",
	).Default(defaultManifest).ExistingFile()
)

var (
	destroyCommand      = zoneCommand.Command("destroy", "destroy a zone")
	destroyManifestPath = destroyCommand.Flag(
		"manifest",
		"path to zone manifest (will be deleted when the zone is destroyed)",
	).Default(defaultManifest).ExistingFile()
)

var (
	wipeCommand = app.Command(
		"wipe",
		"wipe an entire environment (see `zone destroy` for a safer option).",
	)
	wipeAWSRegions = wipeCommand.Flag(
		"aws-region",
		"AWS region(s) to scan (e.g., \"us-west-2\")",
	).PlaceHolder("REGION").Required().Strings()
	wipeEnvironmentName = EnvironmentName(wipeCommand.Flag(
		"environment",
		"Environment name. Defaults to \"$(whoami)-dev\".",
	).PlaceHolder("ENV").Envar("SUBSTRATE_ENVIRONMENT").Default(defaultEnvironment))
)

var (
	sshCommand      = zoneCommand.Command("ssh", "ssh to an instance in a zone")
	sshManifestPath = sshCommand.Flag(
		"manifest",
		"path to zone manifest",
	).Default(defaultManifest).ExistingFile()
	sshHost = sshCommand.Flag(
		"host",
		"hostname or IP to which you'd like to connect",
	).Default("director").HintOptions(
		"border",
		"director",
		"border-0",
		"director-0",
		"worker",
	).String()
	sshArgs = sshCommand.Arg(
		"args",
		"extra arguments to pass through to OpenSSH",
	).Strings()
)

var (
	tunnelCommand = zoneCommand.Command("tunnel", "ssh tunnel to a ip:port in a zone")

	rip = tunnelCommand.Flag(
		"rip",
		"Remote IP address from which to forward a port.",
	).IP()

	localPort = tunnelCommand.Flag(
		"local-port",
		"Local port to which to forward",
	).Default("8888").Int()

	remotePort = tunnelCommand.Flag(
		"remote-port",
		"Remote port to forward",
	).Default("8080").Int()

	manifestPath = tunnelCommand.Flag(
		"manifest",
		"path to zone manifest",
	).Default(defaultManifest).ExistingFile()

	jumpHostUser = tunnelCommand.Flag(
		"jump-host-user",
		"User on border to set up tunnel",
	).Default(defaultUser).String()
)

var (
	logsCommand      = zoneCommand.Command("logs", "tail the cluster level logs for a zone")
	logsManifestPath = logsCommand.Flag(
		"manifest",
		"path to zone manifest",
	).Default(defaultManifest).ExistingFile()
)

func main() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case createCommand.FullCommand():
		err := zone.Create(&zone.CreateInput{
			Version:             version,
			Prompt:              *prompt,
			EnvironmentName:     *createEnvironmentName,
			EnvironmentDomain:   *createEnvironmentDomain,
			EnvironmentIndex:    *createEnvironmentIndex,
			ZoneIndex:           *createZoneIndex,
			AWSAvailabilityZone: *createAvailabilityZone,
			AWSAccountID:        *createAWSAccountID,
			OutputManifestPath:  *createManifestOut,
		})
		app.FatalIfError(err, "create")
	case updateCommand.FullCommand():
		err := zone.Update(&zone.UpdateInput{
			Prompt:        *prompt,
			Version:       version,
			UnsafeUpgrade: *updateUnsafeUpgrade,
			ManifestPath:  *updateManifestPath,
		})
		app.FatalIfError(err, "update")
	case destroyCommand.FullCommand():
		err := zone.Destroy(&zone.DestroyInput{
			Prompt:       *prompt,
			ManifestPath: *destroyManifestPath,
		})
		app.FatalIfError(err, "destroy")
	case sshCommand.FullCommand():
		err := zone.SSH(&zone.SSHInput{
			ManifestPath: *sshManifestPath,
			Host:         *sshHost,
			Args:         *sshArgs,
		})
		app.FatalIfError(err, "ssh")
	case wipeCommand.FullCommand():
		err := wipe.Wipe(&wipe.Input{
			Prompt:          *prompt,
			AWSRegions:      *wipeAWSRegions,
			EnvironmentName: *wipeEnvironmentName,
		})
		app.FatalIfError(err, "wipe")
	case tunnelCommand.FullCommand():
		err := zone.MakeTunnel(&zone.TunnelInput{
			Rip:          *rip,
			LocalPort:    *localPort,
			RemotePort:   *remotePort,
			JumpHostUser: *jumpHostUser,
			ManifestPath: *manifestPath,
		})
		app.FatalIfError(err, "tunnel")
	case logsCommand.FullCommand():
		err := zone.Logs(&zone.LogsInput{
			Version:      version,
			ManifestPath: *logsManifestPath,
		})
		app.FatalIfError(err, "logs")
	}
}
