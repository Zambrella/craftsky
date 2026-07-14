package notifications

import "testing"

func TestClassifyPostReasonUsesCanonicalPerRecipientPrecedence(t *testing.T) {
	tests := []struct {
		name string
		in   PostReasons
		want Category
		ok   bool
	}{
		{"reply wins all", PostReasons{Reply: true, Quote: true, Mention: true}, Reply, true},
		{"quote wins mention", PostReasons{Quote: true, Mention: true}, Quote, true},
		{"mention alone", PostReasons{Mention: true}, Mention, true},
		{"no reason", PostReasons{}, "", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := ClassifyPostReason(test.in)
			if got != test.want || ok != test.ok {
				t.Fatalf("ClassifyPostReason(%+v) = %q,%v; want %q,%v", test.in, got, ok, test.want, test.ok)
			}
		})
	}
}
