package cmd

import "testing"

func TestSanitize(t *testing.T) {
	desc := "[rfe_id:27350][performance]Topology Manager [test_id:26932][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] should be enabled with the policy specified in profile"
	expected := "[performance]Topology Manager  should be enabled with the policy specified in profile"
	sanitized := sanitizeName(desc)
	if sanitized != expected {
		t.Errorf("Sanitized is %s", sanitized)
	}
}
