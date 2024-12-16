# Networking in DevPod

## SSH Tunnels

DevPod uses openssh's client to connect to a SSH server running in the devcontainer. Since the SSH server is not addressable on the local network the local ssh-agent is configured using SSH_CONFIG (~/.ssh/config) to use `devpod ssh ...` as the ProxyCommand. This establishes a connection between the local machine and devcontainer (see [cmd/ssh.go](../../cmd/ssh.go) for usage and `~/.ssh/config` for the exact command). This command uses a DevPod provider to establish the "outer tunnel", for kubernetes this is "kubectl exec", docker is "docker exec" etc. This provider command is executed in a shell from the local environment where the STDIO of the outer tunnel (e.g. kubectl exec) is mapped to the shell. The command for this outer tunnel is `devpod helper ssh-server ...`, which spawns an SSH server on the devcontainer using the STDIO of the outer tunnel (mapped to the local machines shell).

The implementation of our SSH server is provided by DevPod (helper ssh-server), where a fork of [gliderlabs/ssh](https://github.com/gliderlabs/ssh) has been used (see `pkg/ssh` and [cmd/helper/ssh_server](../../cmd/helper/ssh_server.go) for usage). The SSH server can now be thought of as a L7 application layer to provide custom functionality to DevPod, such as tunneling the STDIO from the local machine, port forwarding, agent forwarding etc.

### SSH Agent Forwarding

In order for SSH to multiplex it's listening socket, the server uses [channels](https://www.rfc-editor.org/rfc/rfc4254#section-5) to isolate the request types. Each SSH client connected to the server has an encrypted tunnel and this consists of multiple channels. Each channel performs different actions to provide functionality like agent forwarding, tcp/ip forwarding, SFTP etc. When a client connects to the server it can request these channels using a request type.

In order for DevPod to authenticate a local user with git in a remote devcontainer, it uses agent forwarding to forward the local SSH_AUTH_SOCK to the devcontainer. To do so the client sends a request for channel type "auth-agent@openssh.com", on the server side this gets set in the request's context and the handler can check for it and act according ([see usage](../../pkg/ssh/server/ssh.go)). In our particular case this involves creating a unix socket to bind the forwarded agent to and setting the environments SSH_AUTH_SOCK environment variable to co ordinate with the devcontainer's ssh-agent.

### Debugging

If agent forwarding is enabled then the env var `SSH_AUTH_SOCK` should be available in the workspace. Once inside the workspace you should verify the socket exists `ls -la $SSH_AUTH_SOCK`, if it does then you should expect `ssh -T git@github.com` to return 0 error code. Otherwise the agent forwarding is not working (if the local ssh-agent is authenticated with github).

### Useful reading
 - https://www.howtogeek.com/devops/what-is-ssh-agent-forwarding-and-how-do-you-use-it/
 - https://www.rfc-editor.org/rfc/rfc4254
 - https://www.rfc-editor.org/rfc/rfc4251
 - https://datatracker.ietf.org/doc/html/rfc4253