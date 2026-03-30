package firewall

import "testing"

func TestManagedTableSpec(t *testing.T) {
	t.Parallel()

	family, tableName, ok := managedTableSpec(`
# cloudfirewall generated artifact

table inet cloudfirewall {
  chain input {
  }
}
`)
	if !ok {
		t.Fatal("expected managed table spec to be detected")
	}
	if family != "inet" {
		t.Fatalf("unexpected family: got %q want %q", family, "inet")
	}
	if tableName != "cloudfirewall" {
		t.Fatalf("unexpected table name: got %q want %q", tableName, "cloudfirewall")
	}
}

func TestManagedTableSpecRejectsNonTableInput(t *testing.T) {
	t.Parallel()

	if _, _, ok := managedTableSpec("# comment only\n\n"); ok {
		t.Fatal("expected managed table spec detection to fail")
	}
}
