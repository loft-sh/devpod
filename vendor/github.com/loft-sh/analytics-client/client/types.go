package client

type Event map[string]map[string]interface{}

type Client interface {
	// RecordEvent will record a new event in the client
	RecordEvent(Event)

	// Flush forces sending queued events to the server
	Flush()
}

type Request struct {
	Data []Event `json:"data,omitempty"`
}
