package settings

import "testing"

func TestUpdateHasChanges(t *testing.T) {
	siteName := "Lyra"
	enabled := true
	emptyMethods := []string{}

	cases := []struct {
		name   string
		update Update
		want   bool
	}{
		{name: "empty", update: Update{}, want: false},
		{name: "string pointer", update: Update{SiteName: &siteName}, want: true},
		{name: "bool pointer", update: Update{DebugEnabled: &enabled}, want: true},
		{name: "empty slice means explicit update", update: Update{EpayMethods: emptyMethods}, want: true},
		{name: "clear epay secret", update: Update{ClearEpayKey: true}, want: true},
		{name: "clear smtp password", update: Update{ClearSMTPPassword: true}, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.update.HasChanges(); got != tc.want {
				t.Fatalf("HasChanges() = %v, want %v", got, tc.want)
			}
		})
	}
}
