package vm

import _ "embed"

//go:embed main.tf
var GCPTerraformTemplate string

//go:embed cloud-config.yaml.tftpl
var GCPCloudConfigTemplate string
