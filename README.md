sudo gitlab-runner register

If you are behind a proxy, add an environment variable and then run the registration command:

export HTTP_PROXY=http://yourproxyurl:3128
export HTTPS_PROXY=http://yourproxyurl:3128

sudo -E gitlab-runner register

Enter your GitLab URL:
For runners on GitLab self-managed, use the URL for your GitLab instance. For example, if your project is hosted on gitlab.example.com/yourname/yourproject, your GitLab instance URL is https://gitlab.example.com.
For runners on GitLab.com, the GitLab instance URL is https://gitlab.com.
Enter the runner authentication token.
Enter a description for the runner.
Enter the job tags, separated by commas.
Enter an optional maintenance note for the runner.
Enter the type of executor.
To register multiple runners on the same host machine, each with a different configuration, repeat the register command.
To register the same configuration on multiple host machines, use the same runner authentication token for each runner registration. For more information, see Reusing a runner configuration.
You can also use the non-interactive mode to use additional arguments to register the runner:

Linux
macOS
Windows
FreeBSD
Docker
sudo gitlab-runner register \
  --non-interactive \
  --url "https://gitlab.com/" \
  --token "$RUNNER_TOKEN" \
  --executor "docker" \
  --docker-image alpine:latest \
  --description "docker-runner"

Register with a runner registration token (deprecated)
The ability to pass a runner registration token, and support for certain configuration arguments was deprecated in GitLab 15.6 and will be removed in GitLab 18.0. Runner authentication tokens should be used instead. For more information, see Migrating to the new runner registration workflow.
Prerequisites:

Runner registration tokens must be enabled in the Admin Area.
Obtain a runner registration token at the desired instance, group, or project.
After you register the runner, the configuration is saved to the config.toml.

To register the runner with a runner registration token:

Run the register command:

Linux
macOS
Windows
FreeBSD
Docker
sudo gitlab-runner register

If you are behind a proxy, add an environment variable and then run the registration command:

export HTTP_PROXY=http://yourproxyurl:3128
export HTTPS_PROXY=http://yourproxyurl:3128

sudo -E gitlab-runner register

Enter your GitLab URL:
For GitLab self-managed runners, use the URL for your GitLab instance. For example, if your project is hosted on gitlab.example.com/yourname/yourproject, your GitLab instance URL is https://gitlab.example.com.
For GitLab.com, the GitLab instance URL is https://gitlab.com.
Enter the token you obtained to register the runner.
Enter a description for the runner.
Enter the job tags, separated by commas.
Enter an optional maintenance note for the runner.
Enter the type of executor.
To register multiple runners on the same host machine, each with a different configuration, repeat the register command.

You can also use the non-interactive mode to use additional arguments to register the runner:

Linux
macOS
Windows
FreeBSD
Docker
sudo gitlab-runner register \
  --non-interactive \
  --url "https://gitlab.com/" \
  --registration-token "$PROJECT_REGISTRATION_TOKEN" \
  --executor "docker" \
  --docker-image alpine:latest \
  --description "docker-runner" \
  --maintenance-note "Free-form maintainer notes about this runner" \
  --tag-list "docker,aws" \
  --run-untagged="true" \
  --locked="false" \
  --access-level="not_protected"

--access-level creates a protected runner.
For a protected runner, use the --access-level="ref_protected" parameter.
For an unprotected runner, use --access-level="not_protected" or leave the value undefined.
--maintenance-note allows adding information you might find helpful for runner maintenance. The maximum length is 255 characters.
Legacy-compatible registration process
History 
Runner registration tokens and several runner configuration arguments were deprecated in GitLab 15.6 and will be removed in GitLab 18.0. To ensure minimal disruption to your automation workflow, the legacy-compatible registration process triggers if a runner authentication token is specified in the legacy parameter --registration-token.

The legacy-compatible registration process ignores the following command-line parameters. These parameters can only be configured when a runner is created in the UI or with the API.

--locked
--access-level
--run-untagged
--maximum-timeout
--paused
--tag-list
--maintenance-note
Register with a configuration template
You can use a configuration template to register a runner with settings that are not supported by the register command.

Prerequisites:

The volume for the location of the template file must be mounted on the GitLab Runner container.
A runner authentication or registration token:
Obtain a runner authentication token (recommended). You can either:
Create an instance, group, or project runner.
Locate the runner authentication token in the config.toml file. Runner authentication tokens have the prefix, glrt-.
Obtain a runner registration token (deprecated) for an instance, group, or project runner.
The configuration template can be used for automated environments that do not support some arguments in the register command due to:

Size limits on environment variables based on the environment.
Command-line options that are not available for executor volumes for Kubernetes.
The configuration template supports only a single [[runners]] section and does not support global options.
To register a runner:

Create a configuration template file with the .toml format and add your specifications. For example:

[[runners]]
  [runners.kubernetes]
  [runners.kubernetes.volumes]
    [[runners.kubernetes.volumes.empty_dir]]
      name = "empty_dir"
      mount_path = "/path/to/empty_dir"
      medium = "Memory"

Add the path to the file. You can use either:

The non-interactive mode in the command line:

$ sudo gitlab-runner register \
    --template-config /tmp/test-config.template.toml \
    --non-interactive \
    --url "https://gitlab.com" \
    --token <TOKEN> \ "# --registration-token if using the deprecated runner registration token"
    --name test-runner \
    --executor kubernetes
    --host = "http://localhost:9876/"

The environment variable in the .gitlab.yaml file:

variables:
  TEMPLATE_CONFIG_FILE = <file_path>

If you update the environment variable, you do not need to add the file path in the register command each time you register.

After you register the runner, the settings in the configuration template are merged with the [[runners]] entry created in the config.toml:

concurrent = 1
check_interval = 0

[session_server]
  session_timeout = 1800

[[runners]]
  name = "test-runner"
  url = "https://gitlab.com"
  token = "glrt-<TOKEN>"
  executor = "kubernetes"
  [runners.kubernetes]
    host = "http://localhost:9876/"
    bearer_token_overwrite_allowed = false
    image = ""
    namespace = ""
    namespace_overwrite_allowed = ""
    privileged = false
    service_account_overwrite_allowed = ""
    pod_labels_overwrite_allowed = ""
    pod_annotations_overwrite_allowed = ""
    [runners.kubernetes.volumes]

      [[runners.kubernetes.volumes.empty_dir]]
        name = "empty_dir"
        mount_path = "/path/to/empty_dir"
        medium = "Memory"

Template settings are merged only for options that are:

Empty strings
Null or non-existent entries
Zeroes
Command-line arguments or environment variables take precedence over settings in the configuration template. For example, if the template specifies a docker executor, but the command line specifies shell, the configured executor is shell.

Register a runner for GitLab Community Edition integration tests
To test GitLab Community Edition integrations, use a configuration template to register a runner with a confined Docker executor.

Create a project runner.
Create a template with the [[runners.docker.services]] section:

$ cat > /tmp/test-config.template.toml << EOF
[[runners]]
[runners.docker]
[[runners.docker.services]]
name = "mysql:latest"
[[runners.docker.services]]
name = "redis:latest"

EOF

Register the runner:

Linux
macOS
Windows
FreeBSD
Docker
sudo gitlab-runner register \
  --non-interactive \
  --url "https://gitlab.com" \
  --token "$RUNNER_AUTHENTICATION_TOKEN" \
  --template-config /tmp/test-config.template.toml \
  --description "gitlab-ce-ruby-2.7" \
  --executor "docker" \
  --docker-image ruby:2.7

For more configuration options, see Advanced configuration.

Registering runners with Docker
After you register the runner with a Docker container:

The configuration is written to your configuration volume. For example, /srv/gitlab-runner/config.
The container uses the configuration volume to load the runner.
If gitlab-runner restart runs in a Docker container, GitLab Runner starts a new process instead of restarting the existing process. To apply configuration changes, restart the Docker container instead.
Troubleshooting
Check registration token error
The check registration token error message displays when the GitLab instance does not recognize the runner registration token entered during registration. This issue can occur when either:

The instance, group, or project runner registration token was changed in GitLab.
An incorrect runner registration token was entered.
When this error occurs, you can ask a GitLab administrator to:

Verify that the runner registration token is valid.
Confirm that runner registration in the project or group is permitted.
410 Gone - runner registration disallowed error
The 410 Gone - runner registration disallowed error message displays when runner registration through registration tokens has been disabled.

When this error occurs, you can ask a GitLab administrator to:

Verify that the runner registration token is valid.
Confirm that runner registration in the instance is permitted.
In the case of a group or project runner registration token, verify that runner registration in the respective group and/or project is allowed.
 Help & feedback
Sign in to GitLab.com
Twitter
Facebook
YouTube
LinkedIn
Docs Repo
About GitLab
Terms
Privacy Statement
Cookie Preferences
Contact
View page source - Edit in Web IDE Creative Commons License

On this page
Requirements
Register with a runner authentication token
Register with a runner registration token (deprecated)
Legacy-compatible registration process
Register with a configuration template
Register a runner for GitLab Community Edition integration tests
Registering runners with Docker
Troubleshooting
Check registration token error
410 Gone - runner registration <br>
<a href="https://www.devpod.sh">
  <picture width="500">
    <source media="(prefers-color-scheme: dark)" srcset="docs/static/media/devpod_dark.png">
    <img alt="DevPod wordmark" width="500" src="docs/static/media/devpod.png">
  </picture>
</a>

### **[Website](https://www.devpod.sh)** • **[Quickstart](https://www.devpod.sh/docs/getting-started/install)** • **[Documentation](https://www.devpod.sh/docs/what-is-devpod)** • **[Blog](https://loft.sh/blog)** • **[𝕏 (Twitter)](https://x.com/loft_sh)** • **[Slack](https://slack.loft.sh/)**

[![Join us on Slack!](docs/static/media/slack.svg)](https://slack.loft.sh/) [![Open in DevPod!](https://devpod.sh/assets/open-in-devpod.svg)](https://devpod.sh/open#https://github.com/loft-sh/devpod)

**[We are hiring!](https://www.loft.sh/jobs/5185495004) Come build the future of remote development environments with us.**

DevPod is a client-only tool to create reproducible developer environments based on a [devcontainer.json](https://containers.dev/) on any backend. Each developer environment runs in a container and is specified through a [devcontainer.json](https://containers.dev/). Through DevPod providers, these environments can be created on any backend, such as the local computer, a Kubernetes cluster, any reachable remote machine, or in a VM in the cloud.

![Codespaces](docs/static/media/codespaces-but.png)

You can think of DevPod as the glue that connects your local IDE to a machine where you want to develop. So depending on the requirements of your project, you can either create a workspace locally on the computer, on a beefy cloud machine with many GPUs, or a spare remote computer. Within DevPod, every workspace is managed the same way, which also makes it easy to switch between workspaces that might be hosted somewhere else.

![DevPod Flow](docs/static/media/devpod-flow.gif)

## Quickstart

Download DevPod Desktop:
- [MacOS Silicon/ARM](https://github.com/loft-sh/devpod/releases/latest/download/DevPod_macos_aarch64.dmg)
- [MacOS Intel/AMD](https://github.com/loft-sh/devpod/releases/latest/download/DevPod_macos_x64.dmg)
- [Windows](https://github.com/loft-sh/devpod/releases/latest/download/DevPod_windows_x64_en-US.msi)
- [Linux AppImage](https://github.com/loft-sh/devpod/releases/latest/download/DevPod_linux_amd64.AppImage)

Take a look at the [DevPod Docs](https://devpod.sh/docs/getting-started/install) for more information.

## Why DevPod?

DevPod reuses the open [DevContainer standard](https://containers.dev/) (used by GitHub Codespaces and VSCode DevContainers) to create a consistent developer experience no matter what backend you want to use.

Compared to hosted services such as Github Codespaces, JetBrains Spaces, or Google Cloud Workstations, DevPod has the following advantages:
* **Cost savings**: DevPod is usually around 5-10 times cheaper than existing services with comparable feature sets because it uses bare virtual machines in any cloud and shuts down unused virtual machines automatically.
* **No vendor lock-in**: Choose whatever cloud provider suits you best, be it the cheapest one or the most powerful, DevPod supports all cloud providers. If you are tired of using a provider, change it with a single command.
* **Local development**: You get the same developer experience also locally, so you don't need to rely on a cloud provider at all.
* **Cross IDE support**: VSCode and the full JetBrains suite is supported, all others can be connected through simple ssh.
* **Client-only**: No need to install a server backend, DevPod runs only on your computer.
* **Open-Source**: DevPod is 100% open-source and extensible. A provider doesn't exist? Just create your own.
* **Rich feature set**: DevPod already supports prebuilds, auto inactivity shutdown, git & docker credentials sync, and many more features to come.
* **Desktop App**: DevPod comes with an easy-to-use desktop application that abstracts all the complexity away. If you want to build your own integration, DevPod offers a feature-rich CLI as well.
