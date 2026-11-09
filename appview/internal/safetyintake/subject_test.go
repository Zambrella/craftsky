package safetyintake

import "testing"

func TestCanonicalAppPostLinkIsAccepted(t *testing.T) {
	// Same encoded-DID route shape as the rejected operator intake.
	if !validCanonicalSubject("https://app.craftsky.social/posts/did%3Aplc%3Atestauthor/3testpost", KindAllegation) {
		t.Fatal("normal shared CraftSky post link rejected")
	}
}

func TestCanonicalSharedRecordSubjects(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"https://app.craftsky.social/posts/did%3Aplc%3Atestauthor/3testpost", "at://did:plc:testauthor/social.craftsky.feed.post/3testpost"},
		{" https://app.craftsky.social/profiles/did%3Aplc%3Atestauthor ", "at://did:plc:testauthor/app.bsky.actor.profile/self"},
		{"https://app.craftsky.social/events/did%3Aplc%3Atestauthor/3event", "at://did:plc:testauthor/social.craftsky.business.event/3event"},
		{"https://app.craftsky.social/posts/did%3Aplc%3Atestauthor/root?focus=at%3A%2F%2Fdid%3Aplc%3Areplier%2Fsocial.craftsky.feed.post%2Freply", "at://did:plc:replier/social.craftsky.feed.post/reply"},
		{"https://craftsky.social/posts/did:plc:testauthor/post", "at://did:plc:testauthor/social.craftsky.feed.post/post"},
		{"https://www.craftsky.social/post/legacy", "https://www.craftsky.social/post/legacy"},
		{"at://did:plc:testauthor/social.craftsky.feed.post/post", "at://did:plc:testauthor/social.craftsky.feed.post/post"},
	} {
		got, err := CanonicalSubject(tc.input, KindAllegation)
		if err != nil || got != tc.want {
			t.Errorf("%s: got %q error %v; want %q", tc.input, got, err, tc.want)
		}
	}
	for _, input := range []string{
		"http://app.craftsky.social/posts/did:plc:testauthor/post",
		"https://app.craftsky.social.attacker.invalid/posts/did:plc:testauthor/post",
		"https://user@app.craftsky.social/posts/did:plc:testauthor/post",
		"https://app.craftsky.social:8443/posts/did:plc:testauthor/post",
		"https://app.craftsky.social/posts/invalid/post",
		"https://app.craftsky.social/posts/did:plc:testauthor/a%2Fb",
		"https://app.craftsky.social/posts/did:plc:testauthor/post/extra",
		"https://app.craftsky.social/auth/complete?code=protected-canary",
		"https://app.craftsky.social/posts/did:plc:testauthor/root?focus=bad",
		"https://app.craftsky.social/posts/did:plc:testauthor/root?focus=at://did:plc:testauthor/app.bsky.actor.profile/self",
		"https://app.craftsky.social/posts/did:plc:testauthor/root?focus=a&focus=b",
	} {
		if _, err := CanonicalSubject(input, KindAllegation); err == nil {
			t.Errorf("accepted invalid subject %s", input)
		}
	}
}
