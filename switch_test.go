package main

import "testing"

func TestWithoutDropsOnlyTheNamedVariable(t *testing.T) {
	env := []string{"PATH=/bin", "ZMX_SESSION=supa-1", "ZMX_DIR=/tmp/zmx", "ZMX_SESSION_PREFIX=p"}
	got := without(env, "ZMX_SESSION")

	if len(got) != 3 {
		t.Fatalf("kept %d of %d entries: %q", len(got), len(env), got)
	}
	for _, kv := range got {
		if kv == "ZMX_SESSION=supa-1" {
			t.Error("ZMX_SESSION survived; the child would redirect its host session")
		}
	}
	if got[2] != "ZMX_SESSION_PREFIX=p" {
		t.Errorf("a variable sharing the prefix was dropped: %q", got)
	}
}
