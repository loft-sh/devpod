// Code generated by smithy-go-codegen DO NOT EDIT.

package ssm

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Modifies a task assigned to a maintenance window. You can't change the task
// type, but you can change the following values:
//   - TaskARN . For example, you can change a RUN_COMMAND task from
//     AWS-RunPowerShellScript to AWS-RunShellScript .
//   - ServiceRoleArn
//   - TaskInvocationParameters
//   - Priority
//   - MaxConcurrency
//   - MaxErrors
//
// One or more targets must be specified for maintenance window Run Command-type
// tasks. Depending on the task, targets are optional for other maintenance window
// task types (Automation, Lambda, and Step Functions). For more information about
// running tasks that don't specify targets, see Registering maintenance window
// tasks without targets (https://docs.aws.amazon.com/systems-manager/latest/userguide/maintenance-windows-targetless-tasks.html)
// in the Amazon Web Services Systems Manager User Guide. If the value for a
// parameter in UpdateMaintenanceWindowTask is null, then the corresponding field
// isn't modified. If you set Replace to true, then all fields required by the
// RegisterTaskWithMaintenanceWindow operation are required for this request.
// Optional fields that aren't specified are set to null. When you update a
// maintenance window task that has options specified in TaskInvocationParameters ,
// you must provide again all the TaskInvocationParameters values that you want to
// retain. The values you don't specify again are removed. For example, suppose
// that when you registered a Run Command task, you specified
// TaskInvocationParameters values for Comment , NotificationConfig , and
// OutputS3BucketName . If you update the maintenance window task and specify only
// a different OutputS3BucketName value, the values for Comment and
// NotificationConfig are removed.
func (c *Client) UpdateMaintenanceWindowTask(ctx context.Context, params *UpdateMaintenanceWindowTaskInput, optFns ...func(*Options)) (*UpdateMaintenanceWindowTaskOutput, error) {
	if params == nil {
		params = &UpdateMaintenanceWindowTaskInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "UpdateMaintenanceWindowTask", params, optFns, c.addOperationUpdateMaintenanceWindowTaskMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*UpdateMaintenanceWindowTaskOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type UpdateMaintenanceWindowTaskInput struct {

	// The maintenance window ID that contains the task to modify.
	//
	// This member is required.
	WindowId *string

	// The task ID to modify.
	//
	// This member is required.
	WindowTaskId *string

	// The CloudWatch alarm you want to apply to your maintenance window task.
	AlarmConfiguration *types.AlarmConfiguration

	// Indicates whether tasks should continue to run after the cutoff time specified
	// in the maintenance windows is reached.
	//   - CONTINUE_TASK : When the cutoff time is reached, any tasks that are running
	//   continue. The default value.
	//   - CANCEL_TASK :
	//   - For Automation, Lambda, Step Functions tasks: When the cutoff time is
	//   reached, any task invocations that are already running continue, but no new task
	//   invocations are started.
	//   - For Run Command tasks: When the cutoff time is reached, the system sends a
	//   CancelCommand operation that attempts to cancel the command associated with
	//   the task. However, there is no guarantee that the command will be terminated and
	//   the underlying process stopped. The status for tasks that are not completed
	//   is TIMED_OUT .
	CutoffBehavior types.MaintenanceWindowTaskCutoffBehavior

	// The new task description to specify.
	Description *string

	// The new logging location in Amazon S3 to specify. LoggingInfo has been
	// deprecated. To specify an Amazon Simple Storage Service (Amazon S3) bucket to
	// contain logs, instead use the OutputS3BucketName and OutputS3KeyPrefix options
	// in the TaskInvocationParameters structure. For information about how Amazon Web
	// Services Systems Manager handles these options for the supported maintenance
	// window task types, see MaintenanceWindowTaskInvocationParameters .
	LoggingInfo *types.LoggingInfo

	// The new MaxConcurrency value you want to specify. MaxConcurrency is the number
	// of targets that are allowed to run this task, in parallel. Although this element
	// is listed as "Required: No", a value can be omitted only when you are
	// registering or updating a targetless task (https://docs.aws.amazon.com/systems-manager/latest/userguide/maintenance-windows-targetless-tasks.html)
	// You must provide a value in all other cases. For maintenance window tasks
	// without a target specified, you can't supply a value for this option. Instead,
	// the system inserts a placeholder value of 1 . This value doesn't affect the
	// running of your task.
	MaxConcurrency *string

	// The new MaxErrors value to specify. MaxErrors is the maximum number of errors
	// that are allowed before the task stops being scheduled. Although this element is
	// listed as "Required: No", a value can be omitted only when you are registering
	// or updating a targetless task (https://docs.aws.amazon.com/systems-manager/latest/userguide/maintenance-windows-targetless-tasks.html)
	// You must provide a value in all other cases. For maintenance window tasks
	// without a target specified, you can't supply a value for this option. Instead,
	// the system inserts a placeholder value of 1 . This value doesn't affect the
	// running of your task.
	MaxErrors *string

	// The new task name to specify.
	Name *string

	// The new task priority to specify. The lower the number, the higher the
	// priority. Tasks that have the same priority are scheduled in parallel.
	Priority *int32

	// If True, then all fields that are required by the
	// RegisterTaskWithMaintenanceWindow operation are also required for this API
	// request. Optional fields that aren't specified are set to null.
	Replace *bool

	// The Amazon Resource Name (ARN) of the IAM service role for Amazon Web Services
	// Systems Manager to assume when running a maintenance window task. If you do not
	// specify a service role ARN, Systems Manager uses your account's service-linked
	// role. If no service-linked role for Systems Manager exists in your account, it
	// is created when you run RegisterTaskWithMaintenanceWindow . For more
	// information, see the following topics in the in the Amazon Web Services Systems
	// Manager User Guide:
	//   - Using service-linked roles for Systems Manager (https://docs.aws.amazon.com/systems-manager/latest/userguide/using-service-linked-roles.html#slr-permissions)
	//   - Should I use a service-linked role or a custom service role to run
	//   maintenance window tasks?  (https://docs.aws.amazon.com/systems-manager/latest/userguide/sysman-maintenance-permissions.html#maintenance-window-tasks-service-role)
	ServiceRoleArn *string

	// The targets (either managed nodes or tags) to modify. Managed nodes are
	// specified using the format Key=instanceids,Values=instanceID_1,instanceID_2 .
	// Tags are specified using the format Key=tag_name,Values=tag_value . One or more
	// targets must be specified for maintenance window Run Command-type tasks.
	// Depending on the task, targets are optional for other maintenance window task
	// types (Automation, Lambda, and Step Functions). For more information about
	// running tasks that don't specify targets, see Registering maintenance window
	// tasks without targets (https://docs.aws.amazon.com/systems-manager/latest/userguide/maintenance-windows-targetless-tasks.html)
	// in the Amazon Web Services Systems Manager User Guide.
	Targets []types.Target

	// The task ARN to modify.
	TaskArn *string

	// The parameters that the task should use during execution. Populate only the
	// fields that match the task type. All other fields should be empty. When you
	// update a maintenance window task that has options specified in
	// TaskInvocationParameters , you must provide again all the
	// TaskInvocationParameters values that you want to retain. The values you don't
	// specify again are removed. For example, suppose that when you registered a Run
	// Command task, you specified TaskInvocationParameters values for Comment ,
	// NotificationConfig , and OutputS3BucketName . If you update the maintenance
	// window task and specify only a different OutputS3BucketName value, the values
	// for Comment and NotificationConfig are removed.
	TaskInvocationParameters *types.MaintenanceWindowTaskInvocationParameters

	// The parameters to modify. TaskParameters has been deprecated. To specify
	// parameters to pass to a task when it runs, instead use the Parameters option in
	// the TaskInvocationParameters structure. For information about how Systems
	// Manager handles these options for the supported maintenance window task types,
	// see MaintenanceWindowTaskInvocationParameters . The map has the following
	// format: Key: string, between 1 and 255 characters Value: an array of strings,
	// each string is between 1 and 255 characters
	TaskParameters map[string]types.MaintenanceWindowTaskParameterValueExpression

	noSmithyDocumentSerde
}

type UpdateMaintenanceWindowTaskOutput struct {

	// The details for the CloudWatch alarm you applied to your maintenance window
	// task.
	AlarmConfiguration *types.AlarmConfiguration

	// The specification for whether tasks should continue to run after the cutoff
	// time specified in the maintenance windows is reached.
	CutoffBehavior types.MaintenanceWindowTaskCutoffBehavior

	// The updated task description.
	Description *string

	// The updated logging information in Amazon S3. LoggingInfo has been deprecated.
	// To specify an Amazon Simple Storage Service (Amazon S3) bucket to contain logs,
	// instead use the OutputS3BucketName and OutputS3KeyPrefix options in the
	// TaskInvocationParameters structure. For information about how Amazon Web
	// Services Systems Manager handles these options for the supported maintenance
	// window task types, see MaintenanceWindowTaskInvocationParameters .
	LoggingInfo *types.LoggingInfo

	// The updated MaxConcurrency value.
	MaxConcurrency *string

	// The updated MaxErrors value.
	MaxErrors *string

	// The updated task name.
	Name *string

	// The updated priority value.
	Priority int32

	// The Amazon Resource Name (ARN) of the Identity and Access Management (IAM)
	// service role to use to publish Amazon Simple Notification Service (Amazon SNS)
	// notifications for maintenance window Run Command tasks.
	ServiceRoleArn *string

	// The updated target values.
	Targets []types.Target

	// The updated task ARN value.
	TaskArn *string

	// The updated parameter values.
	TaskInvocationParameters *types.MaintenanceWindowTaskInvocationParameters

	// The updated parameter values. TaskParameters has been deprecated. To specify
	// parameters to pass to a task when it runs, instead use the Parameters option in
	// the TaskInvocationParameters structure. For information about how Systems
	// Manager handles these options for the supported maintenance window task types,
	// see MaintenanceWindowTaskInvocationParameters .
	TaskParameters map[string]types.MaintenanceWindowTaskParameterValueExpression

	// The ID of the maintenance window that was updated.
	WindowId *string

	// The task ID of the maintenance window that was updated.
	WindowTaskId *string

	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationUpdateMaintenanceWindowTaskMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsjson11_serializeOpUpdateMaintenanceWindowTask{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson11_deserializeOpUpdateMaintenanceWindowTask{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "UpdateMaintenanceWindowTask"); err != nil {
		return fmt.Errorf("add protocol finalizers: %v", err)
	}

	if err = addlegacyEndpointContextSetter(stack, options); err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddClientRequestIDMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddComputeContentLengthMiddleware(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = v4.AddComputePayloadSHA256Middleware(stack); err != nil {
		return err
	}
	if err = addRetryMiddlewares(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecordResponseTiming(stack); err != nil {
		return err
	}
	if err = addClientUserAgent(stack, options); err != nil {
		return err
	}
	if err = smithyhttp.AddErrorCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = addSetLegacyContextSigningOptionsMiddleware(stack); err != nil {
		return err
	}
	if err = addOpUpdateMaintenanceWindowTaskValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opUpdateMaintenanceWindowTask(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecursionDetection(stack); err != nil {
		return err
	}
	if err = addRequestIDRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = addResponseErrorMiddleware(stack); err != nil {
		return err
	}
	if err = addRequestResponseLogging(stack, options); err != nil {
		return err
	}
	if err = addDisableHTTPSMiddleware(stack, options); err != nil {
		return err
	}
	return nil
}

func newServiceMetadataMiddleware_opUpdateMaintenanceWindowTask(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "UpdateMaintenanceWindowTask",
	}
}
