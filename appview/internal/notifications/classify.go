package notifications

type PostReasons struct {
	Reply   bool
	Quote   bool
	Mention bool
}

// ClassifyPostReason selects the one category emitted for one recipient.
func ClassifyPostReason(reasons PostReasons) (Category, bool) {
	switch {
	case reasons.Reply:
		return Reply, true
	case reasons.Quote:
		return Quote, true
	case reasons.Mention:
		return Mention, true
	default:
		return "", false
	}
}
