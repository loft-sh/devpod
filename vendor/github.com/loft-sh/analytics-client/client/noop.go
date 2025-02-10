package client

func NewNoopClient() Client {
	return &noopClient{}
}

type noopClient struct{}

func (n *noopClient) RecordEvent(event Event) {}

func (n *noopClient) Flush() {}
