package provider

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Polar-Team/terraform-provider-diffusion/internal/deploy"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// RoleSourceEntry is the provider-internal representation of a role source.
type RoleSourceEntry struct {
	SCM     string
	Version string
	URL     string
	Galaxy  string
	Name    string
	ApplyTo string
}

// DiffusionRunConfig holds the complete configuration for one diffusion deploy invocation.
type DiffusionRunConfig struct {
	// Provider-level
	DiffusionBinary  string
	RegistryServer   string
	RegistryProvider string
	ContainerName    string
	ContainerTag     string
	VaultAddr        string
	VaultToken       string
	ArtifactSources  []ArtifactSourceModel

	// Host wait
	HostWaitInitialDelay string
	HostWaitInterval     string
	HostWaitTimeout      string

	// Resource-level
	RoleSources           []RoleSourceEntry
	Playbook              string
	Hosts                 []deploy.InventoryHost
	Groups                []deploy.InventoryGroup
	GroupVars             deploy.GroupVars
	ExtraVars             map[string]string
	SSHPrivateKeyBase64   string
	SkipIfSucceededWithin string
	InventoryRendered     string
}

// DeployResult holds the computed values returned after a successful deploy.
type DeployResult struct {
	RunID             string
	LastDeployed      string
	MergedLockHash    string
	InventoryRendered string
}

// RunDiffusionDeploy builds the `diffusion deploy` CLI argument list and executes it.
func RunDiffusionDeploy(ctx context.Context, cfg DiffusionRunConfig) (DeployResult, error) {
	args := buildArgs(cfg)

	binary := cfg.DiffusionBinary
	if binary == "" {
		binary = "diffusion"
	}

	tflog.Info(ctx, "Executing diffusion deploy", map[string]interface{}{
		"binary": binary,
		"args":   redactArgs(args),
	})

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = &tflogWriter{ctx: ctx, level: "info"}
	cmd.Stderr = &tflogWriter{ctx: ctx, level: "warn"}

	env := os.Environ()
	if cfg.VaultToken != "" {
		env = append(env, "VAULT_TOKEN="+cfg.VaultToken)
	}
	if cfg.VaultAddr != "" {
		env = append(env, "VAULT_ADDR="+cfg.VaultAddr)
	}
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		return DeployResult{}, fmt.Errorf("diffusion deploy failed: %w", err)
	}

	return DeployResult{
		LastDeployed:      time.Now().UTC().Format(time.RFC3339),
		InventoryRendered: cfg.InventoryRendered,
		MergedLockHash:    "",
		RunID:             computeRunID(cfg),
	}, nil
}

func buildArgs(cfg DiffusionRunConfig) []string {
	args := []string{"deploy"}

	for _, rs := range cfg.RoleSources {
		spec := fmt.Sprintf("scm=%s,version=%s", rs.SCM, rs.Version)
		if rs.URL != "" {
			spec += ",url=" + rs.URL
		}
		if rs.Galaxy != "" {
			spec += ",galaxy=" + rs.Galaxy
		}
		if rs.Name != "" {
			spec += ",name=" + rs.Name
		}
		if rs.ApplyTo != "" {
			spec += ",apply_to=" + rs.ApplyTo
		}
		args = append(args, "--role-source", spec)
	}

	if cfg.Playbook != "" {
		args = append(args, "--playbook", cfg.Playbook)
	}

	for _, h := range cfg.Hosts {
		var parts []string
		for k, v := range h.Variables {
			parts = append(parts, k+"="+v)
		}
		args = append(args, "--host", h.Name+"="+strings.Join(parts, ","))
	}

	for _, g := range cfg.Groups {
		args = append(args, "--group", g.Name+"="+strings.Join(g.Hosts, ","))
	}

	for groupName, vars := range cfg.GroupVars {
		for k, v := range vars {
			args = append(args, "--var", groupName+"."+k+"="+v)
		}
	}

	for k, v := range cfg.ExtraVars {
		args = append(args, "--extra-var", k+"="+v)
	}

	if cfg.SkipIfSucceededWithin != "" {
		args = append(args, "--skip-period", cfg.SkipIfSucceededWithin)
	}

	if cfg.SSHPrivateKeyBase64 != "" {
		args = append(args, "--ssh-key-base64", cfg.SSHPrivateKeyBase64)
	}

	if cfg.HostWaitInitialDelay != "" {
		args = append(args, "--host-wait-initial-delay", cfg.HostWaitInitialDelay)
	}
	if cfg.HostWaitInterval != "" {
		args = append(args, "--host-wait-interval", cfg.HostWaitInterval)
	}
	if cfg.HostWaitTimeout != "" {
		args = append(args, "--host-wait-timeout", cfg.HostWaitTimeout)
	}

	return args
}

func computeRunID(cfg DiffusionRunConfig) string {
	h := sha256.New()
	if _, err := fmt.Fprintf(h, "playbook:%s\n", cfg.Playbook); err != nil {
		panic(err)
	}
	if _, err := fmt.Fprintf(h, "skip:%s\n", cfg.SkipIfSucceededWithin); err != nil {
		panic(err)
	}
	for _, rs := range cfg.RoleSources {
		if _, err := fmt.Fprintf(h, "role:%s:%s:%s:%s:%s:%s\n",
			rs.SCM, rs.Version, rs.URL, rs.Galaxy, rs.Name, rs.ApplyTo); err != nil {
			panic(err)
		}
	}
	for groupName, vars := range cfg.GroupVars {
		for k, v := range vars {
			if _, err := fmt.Fprintf(h, "var:%s.%s=%s\n", groupName, k, v); err != nil {
				panic(err)
			}
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func redactArgs(args []string) []string {
	redacted := make([]string, len(args))
	copy(redacted, args)
	for i, a := range redacted {
		lower := strings.ToLower(a)
		if strings.Contains(lower, "password") || strings.Contains(lower, "token") ||
			strings.Contains(lower, "ssh-key-base64") {
			if parts := strings.SplitN(a, "=", 2); len(parts) == 2 {
				redacted[i] = parts[0] + "=***"
			}
		}
		// Redact the value following --ssh-key-base64 flag (positional arg style).
		if i > 0 && strings.Contains(strings.ToLower(redacted[i-1]), "ssh-key-base64") && !strings.HasPrefix(a, "-") {
			redacted[i] = "***"
		}
	}
	return redacted
}

// tflogWriter routes output line-by-line to terraform-plugin-log.
type tflogWriter struct {
	ctx   context.Context
	level string
	buf   strings.Builder
}

func (w *tflogWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	s := w.buf.String()
	for {
		idx := strings.IndexByte(s, '\n')
		if idx < 0 {
			break
		}
		line := s[:idx]
		s = s[idx+1:]
		if w.level == "warn" {
			tflog.Warn(w.ctx, line)
		} else {
			tflog.Info(w.ctx, line)
		}
	}
	w.buf.Reset()
	w.buf.WriteString(s)
	return len(p), nil
}
