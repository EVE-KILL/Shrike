package cli

import (
	"slices"
	"testing"
)

// expandColonArgs is the whole compatibility layer between the Symfony-style
// names and the Cobra tree, so its edge cases are worth pinning down: a false
// positive here silently dispatches the wrong command.
func TestExpandColonArgs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "colon name is split into path segments",
			in:   []string{"db:migrate", "--apply"},
			want: []string{"db", "migrate", "--apply"},
		},
		{
			name: "deep colon name splits fully",
			in:   []string{"db:migrate:repair"},
			want: []string{"db", "migrate", "repair"},
		},
		{
			name: "space form is left alone",
			in:   []string{"db", "migrate", "--apply"},
			want: []string{"db", "migrate", "--apply"},
		},
		{
			name: "plain command is left alone",
			in:   []string{"doctor"},
			want: []string{"doctor"},
		},
		{
			name: "empty argv is left alone",
			in:   []string{},
			want: []string{},
		},
		{
			// The critical case: only argv[0] is eligible, so a colon inside a
			// later argument or flag value must survive untouched.
			name: "colon in a later argument is preserved",
			in:   []string{"search", "--filter", "type:ship"},
			want: []string{"search", "--filter", "type:ship"},
		},
		{
			name: "colon in a flag value on argv[0] position is not split",
			in:   []string{"--filter=type:ship"},
			want: []string{"--filter=type:ship"},
		},
		{
			// Malformed input should reach Cobra intact so the user gets an
			// "unknown command" naming what they actually typed.
			name: "double colon is not silently repaired",
			in:   []string{"db::migrate"},
			want: []string{"db::migrate"},
		},
		{
			name: "trailing colon is not silently repaired",
			in:   []string{"db:"},
			want: []string{"db:"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := expandColonArgs(tc.in)
			if !slices.Equal(got, tc.want) {
				t.Errorf("expandColonArgs(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestHasFlag(t *testing.T) {
	tests := []struct {
		args []string
		name string
		want bool
	}{
		{[]string{"doctor", "--json"}, "json", true},
		{[]string{"doctor", "--json=true"}, "json", true},
		{[]string{"doctor"}, "json", false},
		{[]string{"doctor", "--no-color"}, "no-color", true},
		// Must not match a different flag that merely shares a prefix.
		{[]string{"doctor", "--jsonify"}, "json", false},
	}

	for _, tc := range tests {
		if got := hasFlag(tc.args, tc.name); got != tc.want {
			t.Errorf("hasFlag(%q, %q) = %v, want %v", tc.args, tc.name, got, tc.want)
		}
	}
}

// The command tree must stay resolvable through both spellings. This walks
// every registered command and asserts its colon name dispatches to it.
func TestEveryCommandResolvesViaColonName(t *testing.T) {
	for _, leaf := range leafCommands(rootCmd) {
		name := canonicalName(leaf)
		args := expandColonArgs([]string{name})

		found, _, err := rootCmd.Find(args)
		if err != nil {
			t.Errorf("colon name %q did not resolve: %v", name, err)
			continue
		}
		if found != leaf {
			t.Errorf("colon name %q resolved to %q, want %q",
				name, canonicalName(found), name)
		}
	}
}
