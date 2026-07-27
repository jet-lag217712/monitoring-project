package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePamHelperPrefersDeployScript(t *testing.T) {
	deployDir := t.TempDir()
	script := filepath.Join(deployDir, "scripts", "manage-users.sh")
	if err := os.MkdirAll(filepath.Dir(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(script, []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := resolvePamHelper(deployDir)
	if err != nil {
		t.Fatal(err)
	}
	if got != script {
		t.Fatalf("helper=%q", got)
	}
}

func TestPamUserListWithStubScript(t *testing.T) {
	script := filepath.Join(t.TempDir(), "manage-users.sh")
	if err := os.WriteFile(script, []byte(`#!/usr/bin/env bash
set -euo pipefail
case "$1" in
  list) echo '[{"username":"alice","disabled":false}]' ;;
  *) exit 2 ;;
esac
`), 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := pamUserList(script)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "alice") {
		t.Fatalf("output=%q", out)
	}
}

func TestPamUserCreateWithStubScript(t *testing.T) {
	script := filepath.Join(t.TempDir(), "manage-users.sh")
	if err := os.WriteFile(script, []byte(`#!/usr/bin/env bash
set -euo pipefail
case "$1" in
  create) echo "created $2" ;;
  *) exit 2 ;;
esac
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := pamUserCreate(script, "alice", "secret"); err != nil {
		t.Fatal(err)
	}
}
