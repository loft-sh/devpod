# terraform-json

[![CircleCI](https://circleci.com/gh/hashicorp/terraform-json/tree/main.svg?style=svg)](https://circleci.com/gh/hashicorp/terraform-json/tree/main)
[![GoDoc](https://godoc.org/github.com/hashicorp/terraform-json?status.svg)](https://godoc.org/github.com/hashicorp/terraform-json)

This repository houses data types designed to help parse the data produced by
two [Terraform](https://www.terraform.io/) commands:

* [`terraform show -json`](https://www.terraform.io/docs/commands/show.html#json-output)
* [`terraform providers schema -json`](https://www.terraform.io/docs/commands/providers/schema.html#json)

While containing mostly data types, there are also a few helpers to assist with
working with the data.

This repository also serves as de facto documentation for the formats produced
by these commands. For more details, see the
[GoDoc](https://godoc.org/github.com/hashicorp/terraform-json).

## Why a Separate Repository?

To reduce dependencies on any of Terraform core's internals, we've made a design
decision to make any helpers or libraries that work with the external JSON data
external and not a part of the Terraform GitHub repository itself.

While Terraform core will change often and be relatively unstable, this library
will see a smaller amount of change. Most of the major changes have already
happened leading up to 0.12, so you can expect this library to only see minor
incremental changes going forward.

For this reason, `terraform show -json` and `terraform providers schema -json`
is the recommended format for working with Terraform data externally, and as
such, if you require any help working with the data in these formats, or even a
reference of how the JSON is formatted, use this repository.
