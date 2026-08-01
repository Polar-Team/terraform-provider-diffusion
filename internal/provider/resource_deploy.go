package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Polar-Team/terraform-provider-diffusion/internal/deploy"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure DeployResource satisfies the resource.Resource interface.
var _ resource.Resource = &DeployResource{}

// DeployResource implements the diffusion_deploy resource.
type DeployResource struct {
	providerCfg *ProviderConfig
}

// DeployResourceModel maps the HCL schema to Go values.
type DeployResourceModel struct {
	RoleSources           []RoleSourceModel `tfsdk:"role_sources"`
	Playbook              types.String      `tfsdk:"playbook"`
	Hosts                 types.Map         `tfsdk:"hosts"`
	Groups                types.Map         `tfsdk:"groups"`
	Variables             types.Map         `tfsdk:"variables"`
	ExtraVars             types.Map         `tfsdk:"extra_vars"`
	SSHPrivateKeys        types.Map         `tfsdk:"ssh_private_keys"`
	SkipIfSucceededWithin types.String      `tfsdk:"skip_if_succeeded_within"`
	HostWaitInitialDelay  types.String      `tfsdk:"host_wait_initial_delay"`
	HostWaitInterval      types.String      `tfsdk:"host_wait_interval"`
	HostWaitTimeout       types.String      `tfsdk:"host_wait_timeout"`

	// Computed
	RunID             types.String `tfsdk:"run_id"`
	LastDeployed      types.String `tfsdk:"last_deployed"`
	MergedLockHash    types.String `tfsdk:"merged_lock_hash"`
	InventoryRendered types.String `tfsdk:"inventory_rendered"`
}

// RoleSourceModel maps a single role_sources list entry.
type RoleSourceModel struct {
	SCM     types.String `tfsdk:"scm"`
	Version types.String `tfsdk:"version"`
	URL     types.String `tfsdk:"url"`
	Galaxy  types.String `tfsdk:"galaxy"`
	Name    types.String `tfsdk:"name"`
	ApplyTo types.String `tfsdk:"apply_to"`
}

func NewDeployResource() resource.Resource {
	return &DeployResource{}
}

func (r *DeployResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_deploy"
}

func (r *DeployResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `**diffusion_deploy** deploys Ansible roles to remote hosts using the diffusion
molecule container.

It fetches ` + "`diffusion.lock`" + ` from each ` + "`role_sources`" + ` entry, merges dependency
constraints, and runs ` + "`ansible-playbook`" + ` inside the container. Roles and
collections are installed **inside the container** — nothing is downloaded to
the machine running Terraform.

**Playbook**: when ` + "`playbook`" + ` is omitted, a playbook is auto-generated that
applies each role to its ` + "`apply_to`" + ` hosts pattern (default: ` + "`\"all\"`" + `).

**State tracking**: ` + "`~/.diffusion/state`" + ` is written on every remote host after
each run. Use ` + "`skip_if_succeeded_within`" + ` to skip re-deployment when nothing
has changed.`,

		Attributes: map[string]schema.Attribute{
			"role_sources": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "List of remote role repositories to fetch `diffusion.lock` from.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"scm": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Source control type: `git` or `galaxy`.",
						},
						"version": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Version constraint or git ref (e.g. `>=2.0.0`, `main`, `v1.3.0`).",
						},
						"url": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Git repository URL. Required when `scm = \"git\"`.",
						},
						"galaxy": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Ansible Galaxy role name (`namespace.role_name`). Required when `scm = \"galaxy\"`.",
						},
						"name": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Role name override used in the auto-generated playbook.",
						},
						"apply_to": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Ansible hosts pattern for the auto-generated play. Defaults to `\"all\"`.",
						},
					},
				},
			},
			"playbook": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Path to an Ansible playbook. When omitted, a playbook is auto-generated from `role_sources`.",
			},
			"hosts": schema.MapNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Map of hostname → host variables.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"vars": schema.MapAttribute{
							Optional:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "Ansible connection variables (e.g. `ansible_host`, `ansible_user`).",
						},
					},
				},
			},
			"groups": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.ListType{ElemType: types.StringType},
				MarkdownDescription: "Map of group name → list of host names.",
			},
			"variables": schema.MapNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Map of group name → group variables. Use key `\"all\"` for global variables applied to the all group. Other keys set variables on the corresponding child group.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"vars": schema.MapAttribute{
							Optional:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "Variables for this group.",
						},
					},
				},
			},
			"extra_vars": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Extra variables passed to `ansible-playbook --extra-vars`.",
			},
			"ssh_private_keys": schema.MapAttribute{
				Optional:            true,
				Sensitive:           true,
				ElementType:         types.StringType,
				MarkdownDescription: "Map of named SSH private keys in PEM format (raw text). Each key is base64-encoded automatically before passing to diffusion. Example: `{ default = tls_private_key.ssh.private_key_openssh }`.",
				Validators: []validator.Map{
					MapKeyNoEquals(),
				},
			},
			"skip_if_succeeded_within": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Skip re-deploy if the last run succeeded within this period (Go duration, e.g. `\"24h\"`).",
			},
			"host_wait_initial_delay": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Override provider default initial delay before first host probe.",
			},
			"host_wait_interval": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Override provider default interval between host probes.",
			},
			"host_wait_timeout": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Override provider default hard deadline for host reachability.",
			},
			"run_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "SHA-256 hash of all deploy inputs. Stable for identical plans.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"last_deployed": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 timestamp of the last successful deploy.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"merged_lock_hash": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Hash of the merged `diffusion.lock` across all role sources.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"inventory_rendered": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The rendered Ansible YAML inventory (for inspection / debugging).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *DeployResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(*ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *ProviderConfig, got %T", req.ProviderData),
		)
		return
	}
	r.providerCfg = cfg
}

func (r *DeployResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DeployResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.runDeploy(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeployResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DeployResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeployResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DeployResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.runDeploy(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *DeployResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// No-op: diffusion_deploy is declarative; nothing to "undo".
}

func (r *DeployResource) runDeploy(ctx context.Context, data *DeployResourceModel) diag.Diagnostics {
	runCfg, diags := r.buildRunConfig(ctx, data)
	if diags.HasError() {
		return diags
	}

	result, err := RunDiffusionDeploy(ctx, runCfg)
	if err != nil {
		return diagError("Deploy failed", err.Error())
	}

	data.RunID = types.StringValue(result.RunID)
	data.LastDeployed = types.StringValue(result.LastDeployed)
	data.MergedLockHash = types.StringValue(result.MergedLockHash)
	data.InventoryRendered = types.StringValue(result.InventoryRendered)

	return nil
}

func (r *DeployResource) buildRunConfig(ctx context.Context, data *DeployResourceModel) (DiffusionRunConfig, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Role sources
	roleSources := make([]RoleSourceEntry, 0, len(data.RoleSources))
	for _, rs := range data.RoleSources {
		roleSources = append(roleSources, RoleSourceEntry{
			SCM:     rs.SCM.ValueString(),
			Version: rs.Version.ValueString(),
			URL:     valueOrEmpty(rs.URL),
			Galaxy:  valueOrEmpty(rs.Galaxy),
			Name:    valueOrEmpty(rs.Name),
			ApplyTo: valueOrEmpty(rs.ApplyTo),
		})
	}

	// Hosts
	hosts, d := extractHosts(ctx, data.Hosts)
	diags.Append(d...)
	if diags.HasError() {
		return DiffusionRunConfig{}, diags
	}

	// Groups
	groups, d := extractGroups(ctx, data.Groups)
	diags.Append(d...)
	if diags.HasError() {
		return DiffusionRunConfig{}, diags
	}

	// Variables (per-group)
	groupVars, d := extractGroupVars(ctx, data.Variables)
	diags.Append(d...)

	extraVars, d := extractStringMap(ctx, data.ExtraVars)
	diags.Append(d...)

	// SSH private keys (map of host → PEM, encoded to base64 by provider)
	sshKeys, d := extractStringMap(ctx, data.SSHPrivateKeys)
	diags.Append(d...)

	if diags.HasError() {
		return DiffusionRunConfig{}, diags
	}

	// Base64-encode each SSH key value.
	sshKeysBase64 := make(map[string]string, len(sshKeys))
	for host, pem := range sshKeys {
		sshKeysBase64[host] = base64.StdEncoding.EncodeToString([]byte(pem))
	}

	// Pre-render inventory for the computed attribute
	rendered, err := buildInventoryRendered(hosts, groups, groupVars)
	if err != nil {
		diags.AddError("Inventory render failed", err.Error())
		return DiffusionRunConfig{}, diags
	}

	cfg := DiffusionRunConfig{
		DiffusionBinary:       r.providerCfg.DiffusionBinary,
		RegistryServer:        r.providerCfg.RegistryServer,
		RegistryProvider:      r.providerCfg.RegistryProvider,
		ContainerName:         r.providerCfg.ContainerName,
		ContainerTag:          r.providerCfg.ContainerTag,
		VaultAddr:             r.providerCfg.VaultAddr,
		VaultToken:            r.providerCfg.VaultToken,
		ArtifactSources:       r.providerCfg.ArtifactSources,
		HostWaitInitialDelay:  coalesce(valueOrEmpty(data.HostWaitInitialDelay), r.providerCfg.HostWaitInitialDelay),
		HostWaitInterval:      coalesce(valueOrEmpty(data.HostWaitInterval), r.providerCfg.HostWaitInterval),
		HostWaitTimeout:       coalesce(valueOrEmpty(data.HostWaitTimeout), r.providerCfg.HostWaitTimeout),
		RoleSources:           roleSources,
		Playbook:              valueOrEmpty(data.Playbook),
		Hosts:                 hosts,
		Groups:                groups,
		GroupVars:             groupVars,
		ExtraVars:             extraVars,
		SSHPrivateKeysBase64:  base64EncodeMap(ctx, data.SSHPrivateKeys, &diags),
		SkipIfSucceededWithin: valueOrEmpty(data.SkipIfSucceededWithin),
		InventoryRendered:     rendered,
	}

	return cfg, diags
}

func buildInventoryRendered(hosts []deploy.InventoryHost, groups []deploy.InventoryGroup, groupVars deploy.GroupVars) (string, error) {
	if len(hosts) == 0 {
		return "", nil
	}
	data, err := deploy.BuildInventory(hosts, groups, groupVars)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func diagError(summary, detail string) diag.Diagnostics {
	var diags diag.Diagnostics
	diags.AddError(summary, detail)
	return diags
}

func valueOrEmpty(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

func coalesce(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// base64EncodeIfSet returns the base64-encoded version of s, or "" if s is empty.
// This allows the provider to accept raw PEM text and encode it transparently
// before passing to the diffusion CLI which expects base64.
func base64EncodeIfSet(s string) string {
	if s == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// base64EncodeMap extracts a types.Map of string values and returns a map
// where each value has been base64-encoded. Returns nil for null/unknown maps.
func base64EncodeMap(ctx context.Context, m types.Map, diags *diag.Diagnostics) map[string]string {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}
	elements := make(map[string]types.String, len(m.Elements()))
	d := m.ElementsAs(ctx, &elements, false)
	diags.Append(d...)
	if d.HasError() {
		return nil
	}
	result := make(map[string]string, len(elements))
	for k, v := range elements {
		if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
			result[k] = base64.StdEncoding.EncodeToString([]byte(v.ValueString()))
		}
	}
	return result
}
