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
	vars := GroupVars{"all": {"env": "prod"}}

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
	if !strings.Contains(out, "env") {
		t.Errorf("expected 'env' variable in output:\n%s", out)
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

func TestBuildInventory_PerGroupVariables(t *testing.T) {
	hosts := []InventoryHost{
		{Name: "web01", Variables: map[string]string{"ansible_host": "10.0.0.1"}},
		{Name: "db01", Variables: map[string]string{"ansible_host": "10.0.0.2"}},
	}
	groups := []InventoryGroup{
		{Name: "webservers", Hosts: []string{"web01"}},
		{Name: "databases", Hosts: []string{"db01"}},
	}
	vars := GroupVars{
		"all":        {"env": "staging", "region": "us-east-1"},
		"webservers": {"http_port": "80", "https_port": "443"},
		"databases":  {"db_port": "5432"},
	}

	data, err := BuildInventory(hosts, groups, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(data)

	// all group vars
	if !strings.Contains(out, "env") {
		t.Errorf("expected 'env' in all group vars:\n%s", out)
	}
	if !strings.Contains(out, "region") {
		t.Errorf("expected 'region' in all group vars:\n%s", out)
	}

	// webservers group vars
	if !strings.Contains(out, "http_port") {
		t.Errorf("expected 'http_port' in webservers group vars:\n%s", out)
	}
	if !strings.Contains(out, "https_port") {
		t.Errorf("expected 'https_port' in webservers group vars:\n%s", out)
	}

	// databases group vars
	if !strings.Contains(out, "db_port") {
		t.Errorf("expected 'db_port' in databases group vars:\n%s", out)
	}
}

func TestBuildInventory_GroupVarsOnlyAll(t *testing.T) {
	hosts := []InventoryHost{
		{Name: "app01", Variables: map[string]string{"ansible_host": "192.168.1.10"}},
	}
	vars := GroupVars{
		"all": {"deploy_user": "admin", "log_level": "info"},
	}

	data, err := BuildInventory(hosts, nil, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "deploy_user") {
		t.Errorf("expected 'deploy_user' in all vars:\n%s", out)
	}
	if !strings.Contains(out, "log_level") {
		t.Errorf("expected 'log_level' in all vars:\n%s", out)
	}
	if strings.Contains(out, "children") {
		t.Errorf("expected no 'children' key when no groups defined:\n%s", out)
	}
}

func TestBuildInventory_GroupVarsWithoutExplicitGroup(t *testing.T) {
	// GroupVars references a group name that is not in the groups slice.
	// The group should still appear in children with its vars but no hosts.
	hosts := []InventoryHost{
		{Name: "app01", Variables: map[string]string{"ansible_host": "10.0.0.5"}},
	}
	vars := GroupVars{
		"monitoring": {"prometheus_port": "9090"},
	}

	data, err := BuildInventory(hosts, nil, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "monitoring") {
		t.Errorf("expected 'monitoring' group in children:\n%s", out)
	}
	if !strings.Contains(out, "prometheus_port") {
		t.Errorf("expected 'prometheus_port' in monitoring vars:\n%s", out)
	}
}

func TestBuildInventory_GroupStructVarsMergedWithGroupVars(t *testing.T) {
	// Variables defined on the InventoryGroup struct should merge with GroupVars.
	hosts := []InventoryHost{
		{Name: "web01", Variables: map[string]string{"ansible_host": "10.0.0.1"}},
	}
	groups := []InventoryGroup{
		{
			Name:      "webservers",
			Hosts:     []string{"web01"},
			Variables: map[string]string{"from_struct": "yes"},
		},
	}
	vars := GroupVars{
		"webservers": {"from_groupvars": "also_yes"},
	}

	data, err := BuildInventory(hosts, groups, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "from_struct") {
		t.Errorf("expected 'from_struct' in webservers vars:\n%s", out)
	}
	if !strings.Contains(out, "from_groupvars") {
		t.Errorf("expected 'from_groupvars' in webservers vars:\n%s", out)
	}
}

func TestBuildInventory_EmptyGroupVars(t *testing.T) {
	hosts := []InventoryHost{
		{Name: "web01", Variables: map[string]string{"ansible_host": "10.0.0.1"}},
	}
	groups := []InventoryGroup{
		{Name: "webservers", Hosts: []string{"web01"}},
	}

	// Empty GroupVars - no vars section should appear
	data, err := BuildInventory(hosts, groups, GroupVars{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(data)
	if strings.Contains(out, "vars:") {
		t.Errorf("expected no 'vars:' key when group vars are empty:\n%s", out)
	}
}
