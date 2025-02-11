package platform

import (
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
)

func GetUserName(self *managementv1.Self) string {
	if self.Status.User != nil {
		return self.Status.User.Name
	}

	if self.Status.Team != nil {
		return self.Status.Team.Name
	}

	return self.Status.Subject
}
