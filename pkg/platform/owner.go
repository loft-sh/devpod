package platform

import (
	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"
)

func IsOwner(self *managementv1.Self, owner *storagev1.UserOrTeam) bool {
	if owner == nil {
		return false
	}

	if self.Status.User != nil && self.Status.User.Name == owner.User {
		return true
	}
	if self.Status.Team != nil && self.Status.Team.Name == owner.Team {
		return true
	}

	return false
}
