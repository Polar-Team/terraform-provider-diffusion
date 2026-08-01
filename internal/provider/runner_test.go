package provider

import (
	"encoding/base64"
	"testing"

	"github.com/Polar-Team/terraform-provider-diffusion/internal/deploy"
)

func cfg(roleSources []RoleSourceEntry) DiffusionRunConfig {
	return DiffusionRunConfig{RoleSources: roleSources}
}

func hasArg(args []string, flag, value string) bool {
	for i, a := range args {
		if a == flag && i+1 < len(args) && args[i+1] == value {
			return true
		}
	}
	return false
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func TestBuildArgs_SubCommand(t *testing.T) {
	args := buildArgs(cfg(nil))
	if len(args) == 0 || args[0] != "deploy" {
		t.Errorf("expected first arg to be 'deploy', got %v", args)
	}
}

func TestBuildArgs_RoleSourceGalaxy(t *testing.T) {
	c := cfg([]RoleSourceEntry{
		{SCM: "galaxy", Version: ">=6.0.0", Galaxy: "geerlingguy.docker"},
	})
	args := buildArgs(c)
	if !hasArg(args, "--role-source", "scm=galaxy,version=>=6.0.0,galaxy=geerlingguy.docker") {
		t.Errorf("expected --role-source with galaxy spec, got %v", args)
	}
}

func TestBuildArgs_RoleSourceGit(t *testing.T) {
	c := cfg([]RoleSourceEntry{
		{SCM: "git", Version: "main", URL: "https://github.com/org/role.git"},
	})
	args := buildArgs(c)
	if !hasArg(args, "--role-source", "scm=git,version=main,url=https://github.com/org/role.git") {
		t.Errorf("expected --role-source with git spec, got %v", args)
	}
}

func TestBuildArgs_RoleSourceWithNameAndApplyTo(t *testing.T) {
	c := cfg([]RoleSourceEntry{
		{SCM: "galaxy", Version: ">=1.0.0", Galaxy: "ns.role", Name: "myrole", ApplyTo: "webservers"},
	})
	args := buildArgs(c)
	spec := "scm=galaxy,version=>=1.0.0,galaxy=ns.role,name=myrole,apply_to=webservers"
	if !hasArg(args, "--role-source", spec) {
		t.Errorf("expected --role-source %q, got %v", spec, args)
	}
}

func TestBuildArgs_PlaybookOmittedWhenEmpty(t *testing.T) {
	c := DiffusionRunConfig{Playbook: ""}
	args := buildArgs(c)
	if hasFlag(args, "--playbook") {
		t.Errorf("expected --playbook absent when empty, got %v", args)
	}
}

func TestBuildArgs_PlaybookIncludedWhenSet(t *testing.T) {
	c := DiffusionRunConfig{Playbook: "/path/to/site.yml"}
	args := buildArgs(c)
	if !hasArg(args, "--playbook", "/path/to/site.yml") {
		t.Errorf("expected --playbook /path/to/site.yml, got %v", args)
	}
}

func TestBuildArgs_HostWaitSettings(t *testing.T) {
	c := DiffusionRunConfig{
		HostWaitInitialDelay: "10s",
		HostWaitInterval:     "5s",
		HostWaitTimeout:      "2m",
	}
	args := buildArgs(c)
	if !hasArg(args, "--host-wait-initial-delay", "10s") {
		t.Errorf("expected --host-wait-initial-delay 10s, got %v", args)
	}
	if !hasArg(args, "--host-wait-interval", "5s") {
		t.Errorf("expected --host-wait-interval 5s, got %v", args)
	}
	if !hasArg(args, "--host-wait-timeout", "2m") {
		t.Errorf("expected --host-wait-timeout 2m, got %v", args)
	}
}

func TestBuildArgs_SkipPeriod(t *testing.T) {
	c := DiffusionRunConfig{SkipIfSucceededWithin: "24h"}
	args := buildArgs(c)
	if !hasArg(args, "--skip-period", "24h") {
		t.Errorf("expected --skip-period 24h, got %v", args)
	}
}

func TestBuildArgs_HostsAndGroups(t *testing.T) {
	c := DiffusionRunConfig{
		Hosts: []deploy.InventoryHost{
			{Name: "web01", Variables: map[string]string{"ansible_host": "1.2.3.4"}},
		},
		Groups: []deploy.InventoryGroup{
			{Name: "webservers", Hosts: []string{"web01"}},
		},
	}
	args := buildArgs(c)
	if !hasFlag(args, "--host") {
		t.Errorf("expected --host flag, got %v", args)
	}
	if !hasFlag(args, "--group") {
		t.Errorf("expected --group flag, got %v", args)
	}
}

func TestComputeRunID_Stability(t *testing.T) {
	c := DiffusionRunConfig{
		Playbook: "site.yml",
		RoleSources: []RoleSourceEntry{
			{SCM: "galaxy", Version: ">=6.0.0", Galaxy: "ns.role"},
		},
	}
	id1 := computeRunID(c)
	id2 := computeRunID(c)
	if id1 != id2 {
		t.Errorf("run ID not stable: %q vs %q", id1, id2)
	}
}

func TestComputeRunID_DiffersOnChange(t *testing.T) {
	base := DiffusionRunConfig{Playbook: "site.yml"}
	changed := DiffusionRunConfig{Playbook: "other.yml"}
	if computeRunID(base) == computeRunID(changed) {
		t.Error("expected different run IDs for different playbooks")
	}
}

func TestComputeRunID_Length(t *testing.T) {
	id := computeRunID(DiffusionRunConfig{})
	if len(id) != 16 {
		t.Errorf("expected run ID length 16, got %d", len(id))
	}
}

func TestBuildArgs_SSHKeyBase64Included(t *testing.T) {
	c := DiffusionRunConfig{SSHPrivateKeysBase64: map[string]string{"default": "Zm9vYmFy"}}
	args := buildArgs(c)
	if !hasArg(args, "--ssh-key-base64", "default=Zm9vYmFy") {
		t.Errorf("expected --ssh-key-base64 default=Zm9vYmFy, got %v", args)
	}
}

func TestBuildArgs_SSHKeyBase64OmittedWhenEmpty(t *testing.T) {
	c := DiffusionRunConfig{SSHPrivateKeysBase64: nil}
	args := buildArgs(c)
	if hasFlag(args, "--ssh-key-base64") {
		t.Errorf("expected --ssh-key-base64 absent when nil, got %v", args)
	}
}

func TestBuildArgs_SSHKeyBase64MultipleKeys(t *testing.T) {
	c := DiffusionRunConfig{SSHPrivateKeysBase64: map[string]string{
		"deploy": "a2V5MQ==",
		"backup": "a2V5Mg==",
	}}
	args := buildArgs(c)
	foundDeploy := false
	foundBackup := false
	for i, a := range args {
		if a == "--ssh-key-base64" && i+1 < len(args) {
			if args[i+1] == "deploy=a2V5MQ==" {
				foundDeploy = true
			}
			if args[i+1] == "backup=a2V5Mg==" {
				foundBackup = true
			}
		}
	}
	if !foundDeploy {
		t.Errorf("expected --ssh-key-base64 deploy=a2V5MQ==, got %v", args)
	}
	if !foundBackup {
		t.Errorf("expected --ssh-key-base64 backup=a2V5Mg==, got %v", args)
	}
}

func TestBase64EncodeIfSet(t *testing.T) {
	if got := base64EncodeIfSet(""); got != "" {
		t.Errorf("expected empty string for empty input, got %q", got)
	}

	pem := "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmU\n-----END OPENSSH PRIVATE KEY-----\n"
	got := base64EncodeIfSet(pem)
	want := base64.StdEncoding.EncodeToString([]byte(pem))
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}

	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if string(decoded) != pem {
		t.Errorf("round-trip mismatch: expected %q, got %q", pem, string(decoded))
	}
}

func TestRedactArgs_SSHKeyBase64(t *testing.T) {
	input := []string{"deploy", "--ssh-key-base64", "c2VjcmV0a2V5"}
	inputCopy := make([]string, len(input))
	copy(inputCopy, input)

	redacted := redactArgs(input)

	if !hasArg(redacted, "--ssh-key-base64", "***") {
		t.Errorf("expected --ssh-key-base64 value redacted to ***, got %v", redacted)
	}
	for _, a := range redacted {
		if a == "c2VjcmV0a2V5" {
			t.Errorf("raw value should not appear in redacted args, got %v", redacted)
		}
	}
	for i := range input {
		if input[i] != inputCopy[i] {
			t.Errorf("original input slice was mutated: expected %v, got %v", inputCopy, input)
		}
	}
}
