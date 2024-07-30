# v1.38.0 (2023-09-25)

* **Feature**: This release updates the enum values for ResourceType in SSM DescribeInstanceInformation input and ConnectionStatus in GetConnectionStatus output.

# v1.37.5 (2023-08-21)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.37.4 (2023-08-18)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.37.3 (2023-08-17)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.37.2 (2023-08-07)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.37.1 (2023-08-01)

* No change notes available for this release.

# v1.37.0 (2023-07-31)

* **Feature**: Adds support for smithy-modeled endpoint resolution. A new rules-based endpoint resolution will be added to the SDK which will supercede and deprecate existing endpoint resolution. Specifically, EndpointResolver will be deprecated while BaseEndpoint and EndpointResolverV2 will take its place. For more information, please see the Endpoints section in our Developer Guide.
* **Dependency Update**: Updated to the latest SDK module versions

# v1.36.9 (2023-07-28)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.36.8 (2023-07-13)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.36.7 (2023-06-27)

* **Documentation**: Systems Manager doc-only update for June 2023.

# v1.36.6 (2023-06-15)

* No change notes available for this release.

# v1.36.5 (2023-06-13)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.36.4 (2023-05-04)

* No change notes available for this release.

# v1.36.3 (2023-04-24)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.36.2 (2023-04-10)

* No change notes available for this release.

# v1.36.1 (2023-04-07)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.36.0 (2023-03-22)

* **Feature**: This Patch Manager release supports creating, updating, and deleting Patch Baselines for AmazonLinux2023, AlmaLinux.

# v1.35.7 (2023-03-21)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.35.6 (2023-03-10)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.35.5 (2023-02-22)

* **Bug Fix**: Prevent nil pointer dereference when retrieving error codes.
* **Documentation**: Document only update for Feb 2023

# v1.35.4 (2023-02-20)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.35.3 (2023-02-15)

* **Announcement**: When receiving an error response in restJson-based services, an incorrect error type may have been returned based on the content of the response. This has been fixed via PR #2012 tracked in issue #1910.
* **Bug Fix**: Correct error type parsing for restJson services.

# v1.35.2 (2023-02-03)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.35.1 (2023-01-23)

* No change notes available for this release.

# v1.35.0 (2023-01-05)

* **Feature**: Add `ErrorCodeOverride` field to all error structs (aws/smithy-go#401).

# v1.34.0 (2023-01-04)

* **Feature**: Adding support for QuickSetup Document Type in Systems Manager

# v1.33.4 (2022-12-21)

* **Documentation**: Doc-only updates for December 2022.

# v1.33.3 (2022-12-15)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.33.2 (2022-12-02)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.33.1 (2022-11-22)

* No change notes available for this release.

# v1.33.0 (2022-11-16)

* **Feature**: This release adds support for cross account access in CreateOpsItem, UpdateOpsItem and GetOpsItem. It introduces new APIs to setup resource policies for SSM resources: PutResourcePolicy, GetResourcePolicies and DeleteResourcePolicy.

# v1.32.1 (2022-11-10)

* No change notes available for this release.

# v1.32.0 (2022-11-07)

* **Feature**: This release includes support for applying a CloudWatch alarm to multi account multi region Systems Manager Automation

# v1.31.3 (2022-10-24)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.31.2 (2022-10-21)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.31.1 (2022-10-20)

* No change notes available for this release.

# v1.31.0 (2022-10-13)

* **Feature**: Support of AmazonLinux2022 by Patch Manager

# v1.30.0 (2022-09-26)

* **Feature**: This release includes support for applying a CloudWatch alarm to Systems Manager capabilities like Automation, Run Command, State Manager, and Maintenance Windows.

# v1.29.0 (2022-09-23)

* **Feature**: This release adds new SSM document types ConformancePackTemplate and CloudFormation

# v1.28.1 (2022-09-20)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.28.0 (2022-09-14)

* **Feature**: Fixed a bug in the API client generation which caused some operation parameters to be incorrectly generated as value types instead of pointer types. The service API always required these affected parameters to be nilable. This fixes the SDK client to match the expectations of the the service API.
* **Feature**: This release adds support for Systems Manager State Manager Association tagging.
* **Dependency Update**: Updated to the latest SDK module versions

# v1.27.13 (2022-09-02)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.27.12 (2022-08-31)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.27.11 (2022-08-30)

* No change notes available for this release.

# v1.27.10 (2022-08-29)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.27.9 (2022-08-11)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.27.8 (2022-08-09)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.27.7 (2022-08-08)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.27.6 (2022-08-01)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.27.5 (2022-07-27)

* **Documentation**: Adding doc updates for OpsCenter support in Service Setting actions.

# v1.27.4 (2022-07-05)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.27.3 (2022-06-29)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.27.2 (2022-06-07)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.27.1 (2022-05-17)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.27.0 (2022-05-04)

* **Feature**: This release adds the TargetMaps parameter in SSM State Manager API.

# v1.26.0 (2022-04-29)

* **Feature**: Update the StartChangeRequestExecution, adding TargetMaps to the Runbook parameter

# v1.25.1 (2022-04-25)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.25.0 (2022-04-19)

* **Feature**: Added offset support for specifying the number of days to wait after the date and time specified by a CRON expression when creating SSM association.

# v1.24.1 (2022-03-30)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.24.0 (2022-03-25)

* **Feature**: This Patch Manager release supports creating, updating, and deleting Patch Baselines for Rocky Linux OS.

# v1.23.1 (2022-03-24)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.23.0 (2022-03-23)

* **Feature**: Update AddTagsToResource, ListTagsForResource, and RemoveTagsFromResource APIs to reflect the support for tagging Automation resources. Includes other minor documentation updates.
* **Dependency Update**: Updated to the latest SDK module versions

# v1.22.0 (2022-03-08)

* **Feature**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.21.0 (2022-02-24)

* **Feature**: API client updated
* **Feature**: Adds RetryMaxAttempts and RetryMod to API client Options. This allows the API clients' default Retryer to be configured from the shared configuration files or environment variables. Adding a new Retry mode of `Adaptive`. `Adaptive` retry mode is an experimental mode, adding client rate limiting when throttles reponses are received from an API. See [retry.AdaptiveMode](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/aws/retry#AdaptiveMode) for more details, and configuration options.
* **Feature**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.20.0 (2022-01-14)

* **Feature**: Updated API models
* **Feature**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.19.0 (2022-01-07)

* **Feature**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.18.0 (2021-12-21)

* **Feature**: API Paginators now support specifying the initial starting token, and support stopping on empty string tokens.
* **Feature**: Updated to latest service endpoints

# v1.17.1 (2021-12-02)

* **Bug Fix**: Fixes a bug that prevented aws.EndpointResolverWithOptions from being used by the service client. ([#1514](https://github.com/aws/aws-sdk-go-v2/pull/1514))
* **Dependency Update**: Updated to the latest SDK module versions

# v1.17.0 (2021-11-30)

* **Feature**: API client updated

# v1.16.0 (2021-11-19)

* **Feature**: API client updated
* **Dependency Update**: Updated to the latest SDK module versions

# v1.15.0 (2021-11-12)

* **Feature**: Service clients now support custom endpoints that have an initial URI path defined.
* **Feature**: Waiters now have a `WaitForOutput` method, which can be used to retrieve the output of the successful wait operation. Thank you to [Andrew Haines](https://github.com/haines) for contributing this feature.

# v1.14.0 (2021-11-06)

* **Feature**: The SDK now supports configuration of FIPS and DualStack endpoints using environment variables, shared configuration, or programmatically.
* **Feature**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.13.0 (2021-10-21)

* **Feature**: Updated  to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.12.0 (2021-10-11)

* **Feature**: API client updated
* **Dependency Update**: Updated to the latest SDK module versions

# v1.11.0 (2021-09-24)

* **Feature**: API client updated

# v1.10.1 (2021-09-17)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.10.0 (2021-08-27)

* **Feature**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.9.1 (2021-08-19)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.9.0 (2021-08-12)

* **Feature**: API client updated

# v1.8.1 (2021-08-04)

* **Dependency Update**: Updated `github.com/aws/smithy-go` to latest version.
* **Dependency Update**: Updated to the latest SDK module versions

# v1.8.0 (2021-07-15)

* **Feature**: Updated service model to latest version.
* **Documentation**: Updated service model to latest revision.
* **Dependency Update**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.7.0 (2021-06-25)

* **Feature**: Updated `github.com/aws/smithy-go` to latest version
* **Dependency Update**: Updated to the latest SDK module versions

# v1.6.2 (2021-06-04)

* **Documentation**: Updated service client to latest API model.

# v1.6.1 (2021-05-20)

* **Dependency Update**: Updated to the latest SDK module versions

# v1.6.0 (2021-05-14)

* **Feature**: Constant has been added to modules to enable runtime version inspection for reporting.
* **Feature**: Updated to latest service API model.
* **Dependency Update**: Updated to the latest SDK module versions

