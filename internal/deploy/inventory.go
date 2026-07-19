package deploy

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// InventoryHost represents a single host with its Ansible connection variables.
type InventoryHost struct {
	// Name is the Ansible inventory hostname (alias).
	Name string
	// Variables holds Ansible connection variables (ansible_host, ansible_user, etc.).
	Variables map[string]string
}

// InventoryGroup represents an Ansible host group.
type InventoryGroup struct {
	// Name is the group name.
	Name string
	// Hosts lists the inventory host names that belong to this group.
	Hosts []string
	// Variables holds group-level variables for this group.
	Variables map[string]string
}

// GroupVars maps group names (including "all") to their variables.
type GroupVars map[string]map[string]string

// BuildInventory generates a valid Ansible YAML inventory from the provided
// hosts, groups, and per-group variables. Returns the raw YAML bytes suitable
// for writing to a file or passing directly to ansible-playbook via -i.
//
// The groupVars parameter is a map of group name → variables. The special key
// "all" sets variables on the top-level all group. Other keys set variables on
// the corresponding child group.
//
// Output structure:
//
//	all:
//	  hosts:
//	    web01:
//	      ansible_host: "1.2.3.4"
//	  children:
//	    webservers:
//	      hosts:
//	        web01: {}
//	      vars:
//	        http_port: "80"
//	  vars:
//	    app_version: "1.0"
func BuildInventory(hosts []InventoryHost, groups []InventoryGroup, groupVars GroupVars) ([]byte, error) {
	hostsMap := make(map[string]any)
	for _, h := range hosts {
		if h.Name == "" {
			return nil, fmt.Errorf("inventory host has an empty name")
		}
		vars := make(map[string]any)
		keys := make([]string, 0, len(h.Variables))
		for k := range h.Variables {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			vars[k] = h.Variables[k]
		}
		hostsMap[h.Name] = vars
	}

	childrenMap := make(map[string]any)
	for _, g := range groups {
		if g.Name == "" {
			return nil, fmt.Errorf("inventory group has an empty name")
		}
		groupHosts := make(map[string]any)
		for _, hName := range g.Hosts {
			groupHosts[hName] = map[string]any{}
		}
		groupEntry := map[string]any{
			"hosts": groupHosts,
		}

		// Merge variables from the group struct itself.
		mergedVars := mergeGroupVars(g.Variables, groupVars[g.Name])
		if len(mergedVars) > 0 {
			groupEntry["vars"] = mergedVars
		}

		childrenMap[g.Name] = groupEntry
	}

	// Handle groupVars entries for groups not explicitly listed in groups slice.
	for name, vars := range groupVars {
		if name == "all" {
			continue
		}
		if _, exists := childrenMap[name]; !exists && len(vars) > 0 {
			groupEntry := map[string]any{
				"vars": toAnyMap(vars),
			}
			childrenMap[name] = groupEntry
		}
	}

	allGroup := map[string]any{
		"hosts": hostsMap,
	}
	if len(childrenMap) > 0 {
		allGroup["children"] = childrenMap
	}

	// Variables for the "all" group come from groupVars["all"].
	if allVars, ok := groupVars["all"]; ok && len(allVars) > 0 {
		allGroup["vars"] = toAnyMap(allVars)
	}

	inventory := map[string]any{
		"all": allGroup,
	}

	data, err := yaml.Marshal(inventory)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal inventory: %w", err)
	}

	return data, nil
}

// mergeGroupVars combines variables from the InventoryGroup struct and from the
// GroupVars map for the same group. Values in groupVarsEntry take precedence.
func mergeGroupVars(structVars map[string]string, groupVarsEntry map[string]string) map[string]any {
	if len(structVars) == 0 && len(groupVarsEntry) == 0 {
		return nil
	}
	merged := make(map[string]any)
	for k, v := range structVars {
		merged[k] = v
	}
	for k, v := range groupVarsEntry {
		merged[k] = v
	}
	return merged
}

// toAnyMap converts a map[string]string to map[string]any for YAML marshaling.
func toAnyMap(m map[string]string) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
