package devcontainer

import (
	"github.com/loft-sh/devpod/pkg/agent"
	"github.com/loft-sh/devpod/pkg/devcontainer/config"
	"github.com/loft-sh/devpod/pkg/docker"
	devssh "github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/devpod/pkg/stdio"
	"github.com/loft-sh/devpod/pkg/token"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"os"
	"strings"
)

func (r *Runner) setupContainer(containerDetails *docker.ContainerDetails, mergedConfig *config.MergedDevContainerConfig) error {
	// create tunnel
	// 1. Setup environment variables & profile
	// 2. Probe remote environment
	// 3. Run post create scripts as User
	// 4. Install VSCode extensions
	client, err := r.createSSHTunnel(containerDetails.Id)
	if err != nil {
		return errors.Wrap(err, "create container tunnel")
	}
	defer client.Close()

	// chown user dir
	err = r.chownWorkspace(client, containerDetails, mergedConfig)
	if err != nil {
		return errors.Wrap(err, "chown workspace")
	}

	// patch remote env
	err = r.patchEtcEnvironment(client, mergedConfig)
	if err != nil {
		return errors.Wrap(err, "patch etc environment")
	}

	// patch etc profile
	err = r.patchEtcProfile(client)
	if err != nil {
		return errors.Wrap(err, "patch etc profile")
	}

	// substitute config with container env
	newMergedConfig := &config.MergedDevContainerConfig{}
	err = config.SubstituteContainerEnv(config.ListToObject(containerDetails.Config.Env), mergedConfig, newMergedConfig)
	if err != nil {
		return errors.Wrap(err, "substitute container env")
	}

	// TODO: run post commands
	// TODO: install dot files

	// TODO: install openvscode, extensions & settings
	// TODO: install vscode, extensions & settings
	return nil
}

func (r *Runner) chownWorkspace(client *ssh.Client, containerDetails *docker.ContainerDetails, mergedConfig *config.MergedDevContainerConfig) error {
	user := mergedConfig.RemoteUser
	if mergedConfig.RemoteUser == "" {
		user = containerDetails.Config.User
	}
	if user == "" {
		user = "root"
	}

	_, err := devssh.CombinedOutput(client, "ls /var/devcontainer/.chownWorkspace")
	if err == nil {
		return nil
	}

	out, err := devssh.CombinedOutput(client, "mkdir -p /var/devcontainer && touch /var/devcontainer/.chownWorkspace")
	if err != nil {
		return errors.Wrapf(err, "create marker file: %v", string(out))
	}

	out, err = devssh.CombinedOutput(client, `chown -R `+user+` `+r.SubstitutionContext.ContainerWorkspaceFolder)
	if err != nil {
		return errors.Wrapf(err, "create remote environment: %v", string(out))
	}

	return nil
}

func (r *Runner) patchEtcProfile(client *ssh.Client) error {
	_, err := devssh.CombinedOutput(client, "ls /var/devcontainer/.patchEtcProfileMarker")
	if err == nil {
		return nil
	}

	out, err := devssh.CombinedOutput(client, "mkdir -p /var/devcontainer && touch /var/devcontainer/.patchEtcProfileMarker")
	if err != nil {
		return errors.Wrapf(err, "create marker file: %v", string(out))
	}

	out, err = devssh.CombinedOutput(client, `sed -i -E 's/((^|\s)PATH=)([^\$]*)$/\1${PATH:-\3}/g' /etc/profile || true`)
	if err != nil {
		return errors.Wrapf(err, "create remote environment: %v", string(out))
	}

	return nil
}

func (r *Runner) patchEtcEnvironment(client *ssh.Client, mergedConfig *config.MergedDevContainerConfig) error {
	if len(mergedConfig.RemoteEnv) == 0 {
		return nil
	}

	_, err := devssh.CombinedOutput(client, "ls /var/devcontainer/.patchEtcEnvironmentMarker")
	if err == nil {
		return nil
	}

	out, err := devssh.CombinedOutput(client, "mkdir -p /var/devcontainer && touch /var/devcontainer/.patchEtcEnvironmentMarker")
	if err != nil {
		return errors.Wrapf(err, "create marker file: %v", string(out))
	}

	// build remote env
	remoteEnvs := []string{}
	for k, v := range mergedConfig.RemoteEnv {
		remoteEnvs = append(remoteEnvs, k+"=\""+v+"\"")
	}

	out, err = devssh.CombinedOutput(client, `cat >> /etc/environment <<'etcEnvrionmentEOF'
`+strings.Join(remoteEnvs, "\n")+`
etcEnvrionmentEOF
`)
	if err != nil {
		return errors.Wrapf(err, "create remote environment: %v", string(out))
	}

	return nil
}

func (r *Runner) createSSHTunnel(containerID string) (*ssh.Client, error) {
	tok, err := token.GenerateTemporaryToken()
	if err != nil {
		return nil, errors.Wrap(err, "temp token")
	}

	privateKeyRaw, err := devssh.GetTempPrivateKeyRaw()
	if err != nil {
		return nil, errors.Wrap(err, "get private key")
	}

	signer, err := ssh.ParsePrivateKey(privateKeyRaw)
	if err != nil {
		return nil, errors.Wrap(err, "parse private key")
	}

	clientConfig := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// create our connection
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	err = r.Docker.Tunnel(agent.RemoteDevPodHelperLocation, agent.DefaultAgentDownloadURL, containerID, tok, stdinReader, stdoutWriter, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create docker tunnel")
	}

	// create stdio connection
	conn := stdio.NewStdioStream(stdoutReader, stdinWriter, false)
	c, chans, reqs, err := ssh.NewClientConn(conn, "stdio", clientConfig)
	if err != nil {
		return nil, err
	}

	return ssh.NewClient(c, chans, reqs), nil
}
