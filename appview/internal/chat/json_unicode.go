package chat

import "unicode/utf8"

// ValidJSONUnicode rejects text that encoding/json would silently repair.
// JSON syntax is still checked by the caller's normal decoder. Escaped literal
// backslashes are skipped, so the six literal characters `\ud800` stay valid.
func ValidJSONUnicode(raw []byte) bool {
	if !utf8.Valid(raw) {
		return false
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' {
			continue
		}
		if i+1 >= len(raw) {
			return false
		}
		if raw[i+1] != 'u' {
			i++
			continue
		}
		unit, ok := jsonUnicodeUnit(raw[i:])
		if !ok || unit >= 0xdc00 && unit <= 0xdfff {
			return false
		}
		i += 5
		if unit >= 0xd800 && unit <= 0xdbff {
			low, ok := jsonUnicodeUnit(raw[i+1:])
			if !ok || low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 6
		}
	}
	return true
}

func jsonUnicodeUnit(raw []byte) (uint16, bool) {
	if len(raw) < 6 || raw[0] != '\\' || raw[1] != 'u' {
		return 0, false
	}
	var result uint16
	for _, char := range raw[2:6] {
		result <<= 4
		switch {
		case char >= '0' && char <= '9':
			result |= uint16(char - '0')
		case char >= 'a' && char <= 'f':
			result |= uint16(char-'a') + 10
		case char >= 'A' && char <= 'F':
			result |= uint16(char-'A') + 10
		default:
			return 0, false
		}
	}
	return result, true
}
