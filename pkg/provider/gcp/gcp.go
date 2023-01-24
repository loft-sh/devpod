package gcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/log"
	"github.com/loft-sh/devpod/pkg/provider/gcp/snapshot"
	"github.com/loft-sh/devpod/pkg/provider/gcp/vm"
	"github.com/loft-sh/devpod/pkg/provider/types"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/ssh/server"
	"github.com/loft-sh/devpod/pkg/stdio"
	"github.com/loft-sh/devpod/pkg/template"
	"github.com/loft-sh/devpod/pkg/terraform"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/loft-sh/devpod/scripts"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const providerName = "gcp"

type ProviderConfig struct {
	UseIAPTunnel bool
}

type gcpProvider struct {
	log log.Logger
}

type terraformVariables struct {
	Name         string `json:"name,omitempty"`
	Project      string `json:"project,omitempty"`
	Snapshot     string `json:"snapshot,omitempty"`
	InitScript   string `json:"init_script,omitempty"`
	Zone         string `json:"zone,omitempty"`
	MachineType  string `json:"machine_type,omitempty"`
	MachineImage string `json:"machine_image,omitempty"`
}

type terraformSnapshotVariables struct {
	Name    string `json:"name,omitempty"`
	Project string `json:"project,omitempty"`
	Zone    string `json:"zone,omitempty"`
}

func NewGCPProvider(log log.Logger) types.Provider {
	return &gcpProvider{
		log: log,
	}
}

func (g *gcpProvider) Stop(ctx context.Context, workspace *types.Workspace, options types.StopOptions) error {
	// get workspace dir
	workspaceDir, err := config.GetWorkspaceDir(providerName, workspace.ID)
	if err != nil {
		return err
	}

	// check if dir exists
	_, err = os.Stat(workspaceDir)
	if err != nil {
		return fmt.Errorf("workspace %s does not exist: %v", workspace.ID, err)
	}

	// create ssh client
	handler, err := g.RemoteCommandHost(ctx, workspace, types.RemoteCommandOptions{})
	if err != nil {
		return err
	}
	defer handler.Close()

	// shut down via SSH
	g.log.Infof("Shutting down VM...")
	buf := &bytes.Buffer{}
	err = handler.Run(context.TODO(), "sudo shutdown -t 3 now", nil, buf, buf)
	if err != nil {
		if !strings.Contains(err.Error(), "remote command exited without exit status or exit signal") {
			return errors.Wrapf(err, "shutting down: %v", buf.String())
		}
	}

	g.log.Donef("Successfully shutdown VM")
	return nil
}

func (g *gcpProvider) ApplySnapshot(ctx context.Context, workspace *types.Workspace, options types.ApplySnapshotOptions) error {
	// get workspace dir
	workspaceDir, err := config.GetWorkspaceDir(providerName, workspace.ID)
	if err != nil {
		return err
	}

	// check if dir exists
	_, err = os.Stat(workspaceDir)
	if err != nil {
		return fmt.Errorf("workspace %s does not exist: %v", workspace.ID, err)
	}

	// get snapshot dir
	snapshotDir, err := config.GetSnapshotDir(providerName, workspace.ID)
	if err != nil {
		return err
	}

	// check if dir already exists
	_, err = os.Stat(snapshotDir)
	if err == nil || !os.IsNotExist(err) {
		return nil
	}

	// create dir
	err = os.MkdirAll(snapshotDir, 0755)
	if err != nil {
		return err
	}

	// read variables
	workspaceVariables := &terraformVariables{}
	err = terraform.ReadVariables(filepath.Join(workspaceDir, terraform.VariablesFile), workspaceVariables)
	if err != nil {
		return errors.Wrap(err, "read variables")
	}

	// get project & zone & name
	out, err := json.Marshal(&terraformSnapshotVariables{
		Name:    workspaceVariables.Name,
		Project: workspaceVariables.Project,
		Zone:    workspaceVariables.Zone,
	})
	if err != nil {
		return err
	}

	// create files
	err = template.WriteFiles(snapshotDir, map[string]string{
		"main.tf":               snapshot.GCPTerraformTemplate,
		terraform.VariablesFile: string(out),
	})
	if err != nil {
		return err
	}

	// apply terraform
	g.log.Infof("Creating Snapshot...")
	err = g.terraformApply(ctx, snapshotDir)
	if err != nil {
		// delete the snapshot folder
		_ = os.RemoveAll(snapshotDir)
		return err
	}

	g.log.Donef("Successfully created snapshot")
	return nil
}

func (g *gcpProvider) DestroySnapshot(ctx context.Context, workspace *types.Workspace, options types.DestroySnapshotOptions) error {
	// check snapshot dir
	snapshotDir, err := config.GetSnapshotDir(providerName, workspace.ID)
	if err != nil {
		return err
	}

	// check if dir exists
	_, err = os.Stat(snapshotDir)
	if err != nil {
		// folder doesn't exist, exit here
		return nil
	}

	// check if terraform was used
	_, err = os.Stat(filepath.Join(snapshotDir, "terraform.tfstate"))
	if err == nil {
		var (
			stdout io.Writer
			stderr io.Writer
		)

		if g.log.GetLevel() == logrus.DebugLevel {
			stdout = os.Stdout
			stderr = os.Stderr
		} else {
			buf := &bytes.Buffer{}
			stdout = buf
			stderr = buf

			defer func() {
				if err != nil {
					err = errors.Wrapf(err, "%s", buf.String())
				}
			}()
		}

		// create terraform client
		tf, err := g.newTerraformClient(ctx, snapshotDir, stdout, stderr)
		if err != nil {
			return err
		}

		// destroy the workspace
		g.log.Infof("Destroying snapshot...")
		err = tf.Destroy(ctx)
		if err != nil {
			return err
		}
	}

	// delete the snapshot folder
	err = os.RemoveAll(snapshotDir)
	if err != nil {
		return errors.Wrap(err, "remove snapshot")
	}

	g.log.Donef("Successfully destroyed snapshot")
	return nil
}

func (g *gcpProvider) Apply(ctx context.Context, workspace *types.Workspace, options types.ApplyOptions) error {
	// create workspace dir
	workspaceDir, err := config.GetWorkspaceDir(providerName, workspace.ID)
	if err != nil {
		return err
	}

	// check if dir already exists
	_, err = os.Stat(workspaceDir)
	if err == nil || !os.IsNotExist(err) {
		// apply changes
		err = g.terraformApply(ctx, workspaceDir)
		if err != nil {
			return err
		}

		// We need to call apply 2 times if the instance was stopped
		// because it gets restarted and then a different public IP
		// assigned, which terraform doesn't pick up immediately.
		err = g.terraformApply(ctx, workspaceDir)
		if err != nil {
			return err
		}

		return nil
	}

	// create dir
	err = os.MkdirAll(workspaceDir, 0755)
	if err != nil {
		return err
	}

	// get token
	t, err := token.GenerateToken()
	if err != nil {
		return err
	}

	// fill init script
	initScript, err := template.FillTemplate(scripts.InstallDevPodTemplate, map[string]string{
		"BaseUrl": "https://github.com/FabianKramm/foundation/releases/download/test",
		"Token":   t,
	})
	if err != nil {
		return err
	}

	// does snapshot exist?
	var snapshotName string
	if !options.DisableSnapshot {
		snapshotName, err = g.getSnapshot(workspace)
		if err != nil {
			return err
		}

		if snapshotName != "" {
			g.log.Infof("Using Snapshot to create VM...")
		}
	}

	// get default project
	defaultProject, _ := getDefaultProject()
	zone := "europe-west1-b"
	out, err := json.Marshal(&terraformVariables{
		Name:       workspace.ID,
		Project:    defaultProject,
		Zone:       zone,
		Snapshot:   snapshotName,
		InitScript: base64.StdEncoding.EncodeToString([]byte(initScript)),
	})

	// create files
	err = template.WriteFiles(workspaceDir, map[string]string{
		"main.tf":                 vm.GCPTerraformTemplate,
		"cloud-config.yaml.tftpl": vm.GCPCloudConfigTemplate,
		terraform.VariablesFile:   string(out),
	})
	if err != nil {
		return err
	}

	// apply terraform
	g.log.Infof("Deploying VM...")
	err = g.terraformApply(ctx, workspaceDir)
	if err != nil {
		// delete the workspace folder
		_ = os.RemoveAll(workspaceDir)
		return err
	}

	g.log.Donef("Successfully deployed VM")
	return nil
}

func (g *gcpProvider) getSnapshot(workspace *types.Workspace) (string, error) {
	snapshotDir, err := config.GetSnapshotDir(providerName, workspace.ID)
	if err != nil {
		return "", err
	}

	_, err = os.Stat(snapshotDir)
	if err != nil {
		return "", nil
	}

	vars := &terraformSnapshotVariables{}
	err = terraform.ReadVariables(filepath.Join(snapshotDir, terraform.VariablesFile), vars)
	if err != nil {
		return "", errors.Wrap(err, "read snapshot variables")
	}

	return vars.Name, nil
}

func (g *gcpProvider) terraformApply(ctx context.Context, workspaceDir string) (err error) {
	var (
		stdout io.Writer
		stderr io.Writer
	)

	if g.log.GetLevel() == logrus.DebugLevel {
		stdout = os.Stdout
		stderr = os.Stderr
	} else {
		buf := &bytes.Buffer{}
		stdout = buf
		stderr = buf

		defer func() {
			if err != nil {
				err = errors.Wrapf(err, "%s", buf.String())
			}
		}()
	}

	// create terraform client
	tf, err := g.newTerraformClient(ctx, workspaceDir, stdout, stderr)
	if err != nil {
		return err
	}

	// run terraform init
	g.log.Infof("Initializing terraform...")
	err = tf.Init(ctx)
	if err != nil {
		return errors.Wrap(err, "terraform init")
	}

	// run apply
	g.log.Infof("Applying terraform...")
	err = tf.Apply(ctx)
	if err != nil {
		return errors.Wrap(err, "terraform apply")
	}

	return nil
}

func (g *gcpProvider) RemoteCommandHost(ctx context.Context, workspace *types.Workspace, options types.RemoteCommandOptions) (types.RemoteCommandHandler, error) {
	// check workspace dir
	workspaceDir, err := config.GetWorkspaceDir(providerName, workspace.ID)
	if err != nil {
		return nil, err
	}

	// check if dir exists
	_, err = os.Stat(workspaceDir)
	if err != nil {
		// folder doesn't exist, exit here
		return nil, fmt.Errorf("workspace doesn't exist")
	}

	// read variables
	vars := &terraformVariables{}
	err = terraform.ReadVariables(filepath.Join(workspaceDir, terraform.VariablesFile), vars)
	if err != nil {
		return nil, err
	}

	// check if external ip address
	buf := &bytes.Buffer{}
	tfClient, err := g.newTerraformClient(ctx, workspaceDir, buf, buf)
	if err != nil {
		return nil, err
	}

	// check outputs
	outputs, err := tfClient.Output(ctx)
	if err != nil {
		return nil, err
	}

	// get external address
	externalAddress, ok := outputs["ip_address"]
	var conn net.Conn
	if ok {
		// dial directly
		startWaiting := time.Now()
		for {
			d := net.Dialer{Timeout: time.Second * 5}
			conn, err = d.Dial("tcp", fmt.Sprintf("%s:%d", strings.Trim(string(externalAddress.Value), "\""), server.DefaultPort))
			if err != nil {
				time.Sleep(time.Second)
				if time.Since(startWaiting) > time.Second*10 {
					g.log.Infof("Waiting for devpod agent to come up...")
					startWaiting = time.Now()
				}

				continue
			}

			break
		}
	} else {
		// TODO: this does not work immediately, we need to test the connection before we allow this
		// create tunnel
		conn, err = g.tunnel(vars.Name, vars.Zone, vars.Project)
		if err != nil {
			return nil, err
		}
	}

	// get token
	key, err := devssh.GetPrivateKeyRaw()
	if err != nil {
		return nil, errors.Wrap(err, "read private key")
	}

	// parse private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, errors.Wrap(err, "parse private key")
	}

	// create ssh client
	client, err := devssh.CreateFromConn(conn, vars.Name, &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "dial agent")
	}

	return devssh.NewSSHRemoteCommandHandler(client), nil
}

func (g *gcpProvider) tunnel(name, zone, project string) (net.Conn, error) {
	if zone == "" {
		zone = "europe-west1-b"
	}

	stdoutReader, stdoutWriter := io.Pipe()
	stdinReader, stdinWriter := io.Pipe()

	args := []string{"gcloud", "compute", "start-iap-tunnel", name, strconv.Itoa(server.DefaultPort), "--zone", zone, "--project", project, "--listen-on-stdin", "--verbosity=warning"}
	g.log.ErrorStreamOnly().Debugf("Run %s", strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = stdoutWriter
	cmd.Stdin = stdinReader
	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	go func() {
		defer stdinWriter.Close()
		defer stdoutWriter.Close()

		err = cmd.Wait()
		if err != nil {
			_, _ = os.Stderr.WriteString(err.Error())
		}

		fmt.Println("TUNNEL DONE")
	}()

	return stdio.NewStdioStream(stdoutReader, stdinWriter, false), nil
}

func (g *gcpProvider) Destroy(ctx context.Context, workspace *types.Workspace, options types.DestroyOptions) (err error) {
	// check workspace dir
	workspaceDir, err := config.GetWorkspaceDir(providerName, workspace.ID)
	if err != nil {
		return err
	}

	// check if dir exists
	_, err = os.Stat(workspaceDir)
	if err != nil {
		// folder doesn't exist, exit here
		return nil
	}

	// check if terraform was used
	_, err = os.Stat(filepath.Join(workspaceDir, "terraform.tfstate"))
	if err == nil {
		var (
			stdout io.Writer
			stderr io.Writer
		)

		if g.log.GetLevel() == logrus.DebugLevel {
			stdout = os.Stdout
			stderr = os.Stderr
		} else {
			buf := &bytes.Buffer{}
			stdout = buf
			stderr = buf

			defer func() {
				if err != nil {
					err = errors.Wrapf(err, "%s", buf.String())
				}
			}()
		}

		// create terraform client
		tf, err := g.newTerraformClient(ctx, workspaceDir, stdout, stderr)
		if err != nil {
			return err
		}

		// destroy the workspace
		g.log.Infof("Destroying environment...")
		err = tf.Destroy(ctx)
		if err != nil {
			if !options.Force {
				return err
			}

			g.log.Errorf("Error deleting environment: %v", err)
		}
	}

	// delete the workspace folder
	err = os.RemoveAll(workspaceDir)
	if err != nil {
		return errors.Wrap(err, "remove workspace")
	}

	g.log.Infof("Successfully destroyed environment...")
	return nil
}

func (g *gcpProvider) newTerraformClient(ctx context.Context, workingDir string, stdout, stderr io.Writer) (*tfexec.Terraform, error) {
	execPath, err := terraform.InstallTerraform(ctx)
	if err != nil {
		return nil, err
	}

	// create a new terraform client
	tf, err := tfexec.NewTerraform(workingDir, execPath)
	if err != nil {
		return nil, err
	}

	// set output
	tf.SetStdout(stdout)
	tf.SetStderr(stderr)
	return tf, nil
}

func getDefaultProject() (string, error) {
	out, err := exec.Command("gcloud", "config", "list", "--format", "value(core.project)").Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}
