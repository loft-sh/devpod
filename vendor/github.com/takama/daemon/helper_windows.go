// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by
// license that can be found in the LICENSE file.

package daemon

// SystemError contains error description and corresponded action helper to fix it
type SystemError struct {
	Title       string
	Description string
	Action      string
}

var (
	// WinErrCode - List of system errors from Microsoft source:
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms681385(v=vs.85).aspx
	WinErrCode = map[int]SystemError{
		5: SystemError{
			Title:       "ERROR_ACCESS_DENIED",
			Description: "Access denied.",
			Action:      "Administrator access is needed to install a service.",
		},
		1051: SystemError{
			Title:       "ERROR_DEPENDENT_SERVICES_RUNNING",
			Description: "A stop control has been sent to a service that other running services are dependent on.",
		},
		1052: SystemError{
			Title:       "ERROR_INVALID_SERVICE_CONTROL",
			Description: "The requested control is not valid for this service.",
		},
		1053: SystemError{
			Title:       "ERROR_SERVICE_REQUEST_TIMEOUT",
			Description: "The service did not respond to the start or control request in a timely fashion.",
		},
		1054: SystemError{
			Title:       "ERROR_SERVICE_NO_THREAD",
			Description: "A thread could not be created for the service.",
		},
		1055: SystemError{
			Title:       "ERROR_SERVICE_DATABASE_LOCKED",
			Description: "The service database is locked.",
		},
		1056: SystemError{
			Title:       "ERROR_SERVICE_ALREADY_RUNNING",
			Description: "An instance of the service is already running.",
		},
		1057: SystemError{
			Title:       "ERROR_INVALID_SERVICE_ACCOUNT",
			Description: "The account name is invalid or does not exist, or the password is invalid for the account name specified.",
		},
		1058: SystemError{
			Title:       "ERROR_SERVICE_DISABLED",
			Description: "The service cannot be started, either because it is disabled or because it has no enabled devices associated with it.",
		},
		1060: SystemError{
			Title:       "ERROR_SERVICE_DOES_NOT_EXIST",
			Description: "The specified service does not exist as an installed service.",
		},
		1061: SystemError{
			Title:       "ERROR_SERVICE_CANNOT_ACCEPT_CTRL",
			Description: "The service cannot accept control messages at this time.",
		},
		1062: SystemError{
			Title:       "ERROR_SERVICE_NOT_ACTIVE",
			Description: "The service has not been started.",
		},
		1063: SystemError{
			Title:       "ERROR_FAILED_SERVICE_CONTROLLER_CONNECT",
			Description: "The service process could not connect to the service controller.",
		},
		1064: SystemError{
			Title:       "ERROR_EXCEPTION_IN_SERVICE",
			Description: "An exception occurred in the service when handling the control request.",
		},
		1066: SystemError{
			Title:       "ERROR_SERVICE_SPECIFIC_ERROR",
			Description: "The service has returned a service-specific error code.",
		},
		1068: SystemError{
			Title:       "ERROR_SERVICE_DEPENDENCY_FAIL",
			Description: "The dependency service or group failed to start.",
		},
		1069: SystemError{
			Title:       "ERROR_SERVICE_LOGON_FAILED",
			Description: "The service did not start due to a logon failure.",
		},
		1070: SystemError{
			Title:       "ERROR_SERVICE_START_HANG",
			Description: "After starting, the service hung in a start-pending state.",
		},
		1071: SystemError{
			Title:       "ERROR_INVALID_SERVICE_LOCK",
			Description: "The specified service database lock is invalid.",
		},
		1072: SystemError{
			Title:       "ERROR_SERVICE_MARKED_FOR_DELETE",
			Description: "The specified service has been marked for deletion.",
		},
		1073: SystemError{
			Title:       "ERROR_SERVICE_EXISTS",
			Description: "The specified service already exists.",
		},
		1075: SystemError{
			Title:       "ERROR_SERVICE_DEPENDENCY_DELETED",
			Description: "The dependency service does not exist or has been marked for deletion.",
		},
		1077: SystemError{
			Title:       "ERROR_SERVICE_NEVER_STARTED",
			Description: "No attempts to start the service have been made since the last boot.",
		},
		1078: SystemError{
			Title:       "ERROR_DUPLICATE_SERVICE_NAME",
			Description: "The name is already in use as either a service name or a service display name.",
		},
		1079: SystemError{
			Title:       "ERROR_DIFFERENT_SERVICE_ACCOUNT",
			Description: "The account specified for this service is different from the account specified for other services running in the same process.",
		},
		1083: SystemError{
			Title:       "ERROR_SERVICE_NOT_IN_EXE",
			Description: "The executable program that this service is configured to run in does not implement the service.",
		},
		1084: SystemError{
			Title:       "ERROR_NOT_SAFEBOOT_SERVICE",
			Description: "This service cannot be started in Safe Mode.",
		},
	}
)
