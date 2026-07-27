package instagrammeta

func validProviderID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for _, b := range []byte(value) {
		if (b < 'a' || b > 'z') &&
			(b < 'A' || b > 'Z') &&
			(b < '0' || b > '9') &&
			b != '_' && b != '-' && b != '.' {
			return false
		}
	}
	return true
}

func validMessageID(value string) bool {
	if len(value) == 0 || len(value) > 512 {
		return false
	}
	for _, b := range []byte(value) {
		if b < 0x21 || b > 0x7e {
			return false
		}
	}
	return true
}
