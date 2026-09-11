package notifications

import "testing"

func TestModerationNotificationEligibility(t *testing.T) {
	tests := map[string]struct {
		event ModerationEvent
		want  bool
	}{
		"report accepted":              {ModerationEvent{Kind: ModerationReportAccepted}, false},
		"no-action decision":           {ModerationEvent{Kind: ModerationDecision}, false},
		"consequential decision":       {ModerationEvent{Kind: ModerationDecision, ConsequenceChanged: true}, true},
		"effect reversal":              {ModerationEvent{Kind: ModerationEffectsChanged, ConsequenceChanged: true}, true},
		"effect reapplication":         {ModerationEvent{Kind: ModerationEffectsChanged, ConsequenceChanged: true}, true},
		"effect status only":           {ModerationEvent{Kind: ModerationEffectsChanged}, false},
		"appeal confirmed":             {ModerationEvent{Kind: ModerationAppealConfirmed}, false},
		"appeal upheld":                {ModerationEvent{Kind: ModerationAppealResolved}, false},
		"appeal changed consequences":  {ModerationEvent{Kind: ModerationAppealResolved, ConsequenceChanged: true}, true},
		"quiet strike expiry":          {ModerationEvent{Kind: ModerationStrikeExpired}, false},
		"restoring strike expiry":      {ModerationEvent{Kind: ModerationStrikeExpired, EnforcementChanged: true}, true},
		"severe restoration":           {ModerationEvent{Kind: ModerationSevereRestored, ConsequenceChanged: true}, true},
		"restoration without a change": {ModerationEvent{Kind: ModerationSevereRestored}, false},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := ShouldNotifyModeration(test.event); got != test.want {
				t.Errorf("ShouldNotifyModeration(%+v) = %v, want %v", test.event, got, test.want)
			}
		})
	}
	if ShouldNotifyModeration(ModerationEvent{Kind: ModerationEventKind("unknown"), ConsequenceChanged: true, EnforcementChanged: true}) {
		t.Fatal("unknown moderation event must fail closed")
	}
}
