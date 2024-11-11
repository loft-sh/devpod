package platform

func DisplayName(name string, displayName string) string {
	if displayName != "" {
		return displayName
	}

	return name
}
