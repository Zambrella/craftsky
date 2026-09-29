package imagesafety

func ParentDisplayEligible(states []State) bool {
	for _, state := range states {
		if !state.DisplayEligible() {
			return false
		}
	}
	return true
}
