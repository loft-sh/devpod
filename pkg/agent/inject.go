package agent

import (
	"bytes"
	"fmt"
	"github.com/loft-sh/devpod/pkg/template"
	"github.com/loft-sh/devpod/scripts"
	"github.com/pkg/errors"
	"io"
	"os"
	"runtime"
)

func InjectAgent(downloadURL string, exec func(command []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error) error {
	// two methods:
	// - Use tar directly if we want to copy current binary
	// - Call small helper script to download binary
	if runtime.GOOS == "linux" {
		currentBinary, err := os.Executable()
		if err != nil {
			return err
		}

		file, err := os.Open(currentBinary)
		if err != nil {
			return errors.Wrap(err, "open agent binary")
		}
		defer file.Close()

		// use tar in this case
		buf := &bytes.Buffer{}
		err = exec([]string{"sh", "-c", fmt.Sprintf("%s version || cat > %s && chmod +x %s", RemoteDevPodHelperLocation, RemoteDevPodHelperLocation, RemoteDevPodHelperLocation)}, file, buf, buf)
		if err != nil {
			return errors.Wrapf(err, "copy agent binary: %s", buf.String())
		}
	} else {
		// use download in this case
		t, err := template.FillTemplate(scripts.InstallDevPodTemplate, map[string]string{
			"BaseUrl": downloadURL,
		})
		if err != nil {
			return err
		}

		// execute script
		buf := &bytes.Buffer{}
		err = exec([]string{"sh", "-c", t}, nil, buf, buf)
		if err != nil {
			return errors.Wrapf(err, "download agent binary: %s", buf.String())
		}
	}

	return nil
}
