package types

/*
* Keep the dependencies of this package minimal to make it easy to import
 */

type TelemetryRequest struct {
	EventType          EventType          `json:"type,omitempty"`
	Event              interface{}        `json:"event,omitempty"`
	InstanceProperties InstanceProperties `json:"instanceProperties,omitempty"`
	Token              string             `json:"token,omitempty"`
}

// Uses semver spec
type Version struct {
	Major      string `json:"major,omitempty"`
	Minor      string `json:"minor,omitempty"`
	Patch      string `json:"patch,omitempty"`
	PreRelease string `json:"prerelease,omitempty"`
	Build      string `json:"build,omitempty"`
}

type InstanceProperties struct {
	Timestamp   int64   `json:"timestamp,omitempty"`
	ExecutionID string  `json:"executionID,omitempty"`
	UID         string  `json:"uid,omitempty"`
	Arch        string  `json:"arch,omitempty"`
	OS          string  `json:"os,omitempty"`
	Version     Version `json:"version,omitempty"`
	Flags       Flags   `json:"flags,omitempty"`
	UI          bool    `json:"ui"`
}

type EventType string

const (
	EventCommandStarted  EventType = "cmdstarted"
	EventCommandFinished EventType = "cmdfinished"
)

type Event interface{}

type CMDStartedEvent struct {
	// Timestamp represents Unix timestampt in microseconds
	Timestamp       int64  `json:"timestamp,omitempty"`
	ExecutionID     string `json:"executionID,omitempty"`
	Command         string `json:"command,omitempty"`
	Provider        string `json:"provider,omitempty"`
	ProviderVersion string `json:"providerVersion,omitempty"` //TODO: implement a way to get this value
}

type CMDFinishedEvent struct {
	// Time represents Unix timestampt in microseconds
	Timestamp       int64  `json:"timestamp,omitempty"`
	ExecutionID     string `json:"executionID,omitempty"`
	Command         string `json:"command,omitempty"`
	Provider        string `json:"provider,omitempty"`
	ProviderVersion string `json:"providerVersion,omitempty"` //TODO: implement a way to get this value
	Success         bool   `json:"success,omitempty"`
	ProcessingTime  int    `json:"processingTime,omitempty"`
	Errors          string `json:"errors,omitempty"`
}

type Flags struct {
	SetFlags []string `json:"setFlags,omitempty"`
}
