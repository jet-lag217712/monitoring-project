package setup

import (
	"os"
	"testing"
)

func TestPamHasExistingUsers(t *testing.T) {
	script := t.TempDir() + "/manage-users.sh"
	if err := writeStubManageUsers(script, `#!/usr/bin/env bash
case "$1" in
  list) echo "admin (enabled)" ;;
  *) exit 2 ;;
esac
`); err != nil {
		t.Fatal(err)
	}
	has, body, err := pamHasExistingUsers(script)
	if err != nil {
		t.Fatal(err)
	}
	if !has || body == "" {
		t.Fatalf("has=%v body=%q", has, body)
	}
}

func TestPamHasExistingUsersEmpty(t *testing.T) {
	script := t.TempDir() + "/manage-users.sh"
	if err := writeStubManageUsers(script, `#!/usr/bin/env bash
case "$1" in
  list) echo "No appliance users listed." ;;
  *) exit 2 ;;
esac
`); err != nil {
		t.Fatal(err)
	}
	has, body, err := pamHasExistingUsers(script)
	if err != nil {
		t.Fatal(err)
	}
	if has || body != "" {
		t.Fatalf("has=%v body=%q", has, body)
	}
}

func writeStubManageUsers(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o755)
}
