package main

import "testing"

func TestComparisonUpstreamsStayOnLoopback(t *testing.T) {
	for _, raw := range []string{"http://127.0.0.1:4200", "http://localhost:4450/"} {
		if _, err := localURL(raw); err != nil {
			t.Errorf("local origin rejected: %s", raw)
		}
	}
	for _, raw := range []string{"https://eve-kill.com", "http://100.111.3.8:4200", "http://user:secret@localhost:4200", "http://localhost:4200/api", "http://localhost:4200?target=remote", "//localhost:4200"} {
		if _, err := localURL(raw); err == nil {
			t.Errorf("nonlocal or decorated origin accepted: %s", raw)
		}
	}
	for _, entry := range cases {
		if len(entry.id) != 14 {
			t.Fatalf("invalid campaign fixture id %q", entry.id)
		}
	}
}
