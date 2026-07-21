package notifications

// Category is the closed notification category value used on the API wire and
// in private AppView persistence.
type Category string

const (
	Like           Category = "like"
	Follow         Category = "follow"
	Reply          Category = "reply"
	Mention        Category = "mention"
	Quote          Category = "quote"
	Repost         Category = "repost"
	EverythingElse Category = "everythingElse"
	InstagramMatch Category = "instagramMatch"
)

var categories = [...]Category{
	Like,
	Follow,
	Reply,
	Mention,
	Quote,
	Repost,
	EverythingElse,
	InstagramMatch,
}

// Categories returns the complete public registry in settings presentation
// order. The returned slice is independent so callers cannot mutate the
// registry.
func Categories() []Category {
	return append([]Category(nil), categories[:]...)
}

func (c Category) Valid() bool {
	for _, category := range categories {
		if c == category {
			return true
		}
	}
	return false
}

// HasProducer reports whether this implementation can currently create the
// category. EverythingElse is reserved for a future versioned event source.
func (c Category) HasProducer() bool {
	return c.Valid() && c != EverythingElse
}
