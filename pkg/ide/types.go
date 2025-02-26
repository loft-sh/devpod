package ide

import (
	"io"
	"time"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/log"
)

type IDE interface {
	Install() error
}

type Options map[string]Option

type Option struct {
	// Name is the name of the IDE option
	Name string `json:"name,omitempty"`

	// Description is the description of the IDE option
	Description string `json:"description,omitempty"`

	// Default is the default value for this option
	Default string `json:"default,omitempty"`

	// Enum is the possible values for this option
	Enum []string `json:"enum,omitempty"`

	// ValidationPattern to use to validate this option
	ValidationPattern string `json:"validationPattern,omitempty"`

	// ValidationMessage to print if validation fails
	ValidationMessage string `json:"validationMessage,omitempty"`
}

func (o Options) GetValue(values map[string]config.OptionValue, key string) string {
	if values != nil && values[key].Value != "" {
		return values[key].Value
	} else if o[key].Default != "" {
		return o[key].Default
	}

	return ""
}

// ReusesAuthSock determines if the --reuse-ssh-auth-sock flag should be passed to the ssh server helper based on the IDE.
// Browser based IDEs use a browser tunnel to communicate with the remote server instead of an independent ssh connection
func ReusesAuthSock(ide string) bool {
	return ide == "openvscode" || ide == "jupyternotebook"
}

type ProgressReader struct {
	Reader    io.Reader
	TotalSize int64
	Log       log.Logger

	lastMessage time.Time
	bytesRead   int64
}

func (p *ProgressReader) Read(b []byte) (n int, err error) {
	n, err = p.Reader.Read(b)
	p.bytesRead += int64(n)
	if time.Since(p.lastMessage) > time.Second*1 {
		p.Log.Infof("Downloaded %.2f / %.2f MB", float64(p.bytesRead)/1024/1024, float64(p.TotalSize/1024/1024))
		p.lastMessage = time.Now()
	}

	return n, err
}
