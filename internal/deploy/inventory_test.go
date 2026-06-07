package deploy

import (
	"strings"
	"testing"
)

func TestBuildInventory_Basic(t *testing.T) {
	hosts := []InventoryHost{
		{Name: "web01", Variables: map[string]string{"ansible_host": "1.2.3.4"}},
	}
	groups := []InventoryGroup{
		{Name: "webservers", Hosts: []string{"web01"}},
	}
	vars := map[string]string{"env": "prod"}

	data, err := BuildInventory(hosts, groups, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "web01") {
		t.Errorf("expected host 'web01' in output:\n%s", out)
	}
	if !strings.Contains(out, "webservers") {
		t.Errorf("expected group 'webservers' in output:\n%s", out)
	}
	if !strings.Contains(out, "ansible_host") {
		t.Errorf("expected 'ansible_host' in output:\n%s", out)
	}
}

func TestBuildInventory_EmptyHostName(t *testing.T) {
	hosts := []InventoryHost{
		{Name: "", Variables: map[string]string{}},
	}
	_, err := BuildInventory(hosts, nil, nil)
	if err == nil {
		t.Error("expected error for empty host name")
	}
}

func TestBuildInventory_EmptyGroupName(t *testing.T) {
	groups := []InventoryGroup{
		{Name: "", Hosts: []string{"web01"}},
	}
	_, err := BuildInventory(nil, groups, nil)
	if err == nil {
		t.Error("expected error for empty group name")
	}
}

func TestBuildInventory_NoGroupsNoVars(t *testing.T) {
	hosts := []InventoryHost{
		{Name: "server1", Variables: map[string]string{"ansible_user": "deploy"}},
	}

	data, err := BuildInventory(hosts, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "server1") {
		t.Errorf("expected host in output:\n%s", out)
	}
	if strings.Contains(out, "children") {
		t.Errorf("expected no 'children' key when groups are empty:\n%s", out)
	}
}
