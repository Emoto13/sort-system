package state

func Contains(slice []ItemStatus, value ItemStatus) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
