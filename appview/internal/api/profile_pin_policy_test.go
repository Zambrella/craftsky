package api_test

import (
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestClassifyProfilePinSlot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		target  api.ProfilePinTargetShape
		want    api.ProfilePinSlot
		wantErr bool
	}{
		{name: "top-level standard", target: api.ProfilePinTargetShape{}, want: api.ProfilePinSlotStandard},
		{name: "top-level quote", target: api.ProfilePinTargetShape{HasQuote: true}, want: api.ProfilePinSlotStandard},
		{
			name: "top-level project",
			target: api.ProfilePinTargetShape{
				IsProject:                 true,
				HasProjectMaterialization: true,
			},
			want: api.ProfilePinSlotProject,
		},
		{name: "comment", target: api.ProfilePinTargetShape{HasReplyRoot: true, HasReplyParent: true}, wantErr: true},
		{name: "nested reply", target: api.ProfilePinTargetShape{HasReplyRoot: true, HasReplyParent: true}, wantErr: true},
		{name: "partial reply shape", target: api.ProfilePinTargetShape{HasReplyRoot: true}, wantErr: true},
		{
			name: "project with quote",
			target: api.ProfilePinTargetShape{
				IsProject:                 true,
				HasProjectMaterialization: true,
				HasQuote:                  true,
			},
			wantErr: true,
		},
		{name: "project flag without materialization", target: api.ProfilePinTargetShape{IsProject: true}, wantErr: true},
		{name: "project materialization without flag", target: api.ProfilePinTargetShape{HasProjectMaterialization: true}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := api.ClassifyProfilePinSlot(test.target)
			if test.wantErr {
				if err == nil {
					t.Fatalf("ClassifyProfilePinSlot(%+v) succeeded with %q", test.target, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ClassifyProfilePinSlot(%+v): %v", test.target, err)
			}
			if got != test.want {
				t.Fatalf("ClassifyProfilePinSlot(%+v) = %q, want %q", test.target, got, test.want)
			}
		})
	}
}
