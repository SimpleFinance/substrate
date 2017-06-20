package logwatcher

import (
	"time"
)

// Event represents a single CloudWatch Logs event parsed from journald-cloudwatch-logs
type Event struct {
	// the CloudWatch Logs log group from which we read this event
	LogGroupName string

	// the CloudWatch Logs log stream from which we read this event
	LogStreamName string

	// when this event was generated (as recorded on the originating instance)
	Timestamp time.Time

	// when this event was ingested into CloudWatch Logs (as recorded by AWS)
	IngestedTimestamp time.Time

	// the parsed journald-cloudwatch-logs record (maps ~1:1 to a journald event)
	Record Record
}

// Record type from journald-cloudwatch-logs (with a few small tweaks)
type Record struct {
	InstanceID  string       `json:"instanceId,omitempty"`
	PID         int          `json:"pid" journald:"_PID"`
	UID         int          `json:"uid" journald:"_UID"`
	GID         int          `json:"gid" journald:"_GID"`
	Command     string       `json:"cmdName,omitempty" journald:"_COMM"`
	Executable  string       `json:"exe,omitempty" journald:"_EXE"`
	CommandLine string       `json:"cmdLine,omitempty" journald:"_CMDLINE"`
	SystemdUnit string       `json:"systemdUnit,omitempty" journald:"_SYSTEMD_UNIT"`
	BootID      string       `json:"bootId,omitempty" journald:"_BOOT_ID"`
	MachineID   string       `json:"machineId,omitempty" journald:"_MACHINE_ID"`
	Hostname    string       `json:"hostname,omitempty" journald:"_HOSTNAME"`
	Transport   string       `json:"transport,omitempty" journald:"_TRANSPORT"`
	Priority    string       `json:"priority" journald:"PRIORITY"`
	Message     string       `json:"message" journald:"MESSAGE"`
	MessageID   string       `json:"messageId,omitempty" journald:"MESSAGE_ID"`
	Errno       int          `json:"machineId,omitempty" journald:"ERRNO"`
	Syslog      RecordSyslog `json:"syslog,omitempty"`
	Kernel      RecordKernel `json:"kernel,omitempty"`
}

// RecordSyslog type from journald-cloudwatch-logs
type RecordSyslog struct {
	Facility   int    `json:"facility,omitempty" journald:"SYSLOG_FACILITY"`
	Identifier string `json:"ident,omitempty" journald:"SYSLOG_IDENTIFIER"`
	PID        int    `json:"pid,omitempty" journald:"SYSLOG_PID"`
}

// RecordKernel type from journald-cloudwatch-logs
type RecordKernel struct {
	Device    string `json:"device,omitempty" journald:"_KERNEL_DEVICE"`
	Subsystem string `json:"subsystem,omitempty" journald:"_KERNEL_SUBSYSTEM"`
	SysName   string `json:"sysName,omitempty" journald:"_UDEV_SYSNAME"`
	DevNode   string `json:"devNode,omitempty" journald:"_UDEV_DEVNODE"`
}
