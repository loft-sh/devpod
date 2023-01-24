// Package tfjson is a de-coupled helper library containing types for
// the plan format output by "terraform show -json" command. This
// command is designed for the export of Terraform plan data in
// a format that can be easily processed by tools unrelated to
// Terraform.
//
// This format is stable and should be used over the binary plan data
// whenever possible.
package tfjson
