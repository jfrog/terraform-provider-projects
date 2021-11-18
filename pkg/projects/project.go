package projects

import (
	"context"
	"regexp"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"strconv"
)

type Identifiable interface {
	Id() string
}

type AdminPrivileges struct {
	ManageMembers   bool `json:"manage_members"`
	ManageResources bool `json:"manage_resources"`
	IndexResources  bool `json:"index_resources"`
}

// Project GET {{ host }}/access/api/v1/projects/{{prjKey}}/
//GET {{ host }}/artifactory/api/repositories/?prjKey={{prjKey}}
type Project struct {
	Key                    string           `hcl:"key" json:"project_key"`
	DisplayName            string           `hcl:"display_name" json:"display_name"`
	Description            string           `hcl:"description" json:"description"`
	AdminPrivileges        *AdminPrivileges `hcl:"admin_privileges" json:"admin_privileges"`
	StorageQuota           int              `hcl:"max_storage_in_gigabytes" json:"storage_quota_bytes"`
	SoftLimit              bool             `hcl:"block_deployments_on_limit" json:"soft_limit"`
	QuotaEmailNotification bool             `hcl:"email_notification" json:"storage_quota_email_notification"`
}

func (p Project) Id() string {
	return p.Key
}

const projectsUrl = "/access/api/v1/projects/"

func verifyProject(id string, request *resty.Request) (*resty.Response, error) {
	return request.Head(projectsUrl + id)
}

func projectResource() *schema.Resource {

	var projectSchema = map[string]*schema.Schema{
		"key": {
			Type:     schema.TypeString,
			Required: true,
			ValidateDiagFunc: validation.ToDiagFunc(
				validation.StringMatch(regexp.MustCompile("^[a-z0-9]{3,6}$"), "key must be 3 - 6 lowercase alphanumeric characters"),
			),
			Description: "The Project Key is added as a prefix to resources created within a Project. This field is mandatory and supports only 3 - 6 lowercase alphanumeric characters. Must begin with a letter. For example: us1a.",
		},
		"display_name": {
			Required:         true,
			Type:             schema.TypeString,
			ValidateDiagFunc: validation.ToDiagFunc(validation.All(
				validation.StringIsNotEmpty,
				maxLength(32),
			)),
			Description:      "Also known as project name on the UI",
		},
		"description": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"admin_privileges": {
			Type:     schema.TypeSet,
			Required: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"manage_members": {
						Type:     schema.TypeBool,
						Required: true,
					},
					"manage_resources": {
						Type:     schema.TypeBool,
						Required: true,
					},
					"index_resources": {
						Type:     schema.TypeBool,
						Required: true,
					},
				},
			},
		},
		"max_storage_in_gigabytes": {
			Type:     schema.TypeInt,
			Optional: true,
			Default:  -1,
			ValidateDiagFunc: validation.ToDiagFunc(
				validation.Any(
					validation.IntAtLeast(1),
					validation.IntInSlice([]int{-1}),
				),
			),
			DiffSuppressFunc: func(key, old, new string, d *schema.ResourceData) bool {
				oldVal, err := strconv.Atoi(old)
				if err != nil {
					return false
				}
				newVal, err := strconv.Atoi(new)
				if err != nil {
					return false
				}
				// convert to bytes. The API says bytes, but the UI only allows GB (which makes more sense)
				oldVal = oldVal * 1024 * 1024 * 1024
				newVal = newVal * 1024 * 1024 * 1024
				return newVal == oldVal
			},
		},
		"block_deployments_on_limit": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
		"email_notification": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Alerts will be sent when reaching 75% and 95% of the storage quota. Serves as a notification only and is not a blocker",
		},
	}

	var unpackProject = func(data *schema.ResourceData) (interface{}, string, error) {
		d := &ResourceData{data}

		project := Project{
			Key:                    d.getString("key", false),
			DisplayName:            d.getString("display_name", false),
			Description:            d.getString("description", false),
			StorageQuota:           GigabytlesToBytes(d.getInt("max_storage_in_gigabytes", false)),
			SoftLimit:              d.getBool("block_deployments_on_limit", false),
			QuotaEmailNotification: d.getBool("email_notification", false),
		}

		if v, ok := d.GetOkExists("admin_privileges"); ok {
			privileges := v.(*schema.Set).List()
			if len(privileges) == 0 {
				return nil, "", nil
			}

			adminPrivileges := AdminPrivileges{}

			id := privileges[0].(map[string]interface{})

			adminPrivileges.ManageMembers = id["manage_members"].(bool)
			adminPrivileges.ManageResources = id["manage_resources"].(bool)
			adminPrivileges.IndexResources = id["index_resources"].(bool)

			project.AdminPrivileges = &adminPrivileges
		}

		return project, project.Id(), nil
	}

	var packProject = func(d *schema.ResourceData, project *Project) diag.Diagnostics {
		setValue := mkLens(d)

		setValue("key", project.Key)
		setValue("display_name", project.DisplayName)
		setValue("description", project.Description)
		setValue("max_storage_in_gigabytes", BytesToGigabytles(project.StorageQuota))
		setValue("block_deployments_on_limit", project.SoftLimit)
		setValue("email_notification", project.QuotaEmailNotification)

		if project.AdminPrivileges != nil {
			setValue("admin_privileges", []interface{}{
				map[string]bool{
					"manage_members":   project.AdminPrivileges.ManageMembers,
					"manage_resources": project.AdminPrivileges.ManageResources,
					"index_resources":  project.AdminPrivileges.IndexResources,
				},
			})
		}

		return nil
	}

	var readProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		project := Project{}
		_, err := m.(*resty.Client).R().SetResult(&project).Get(projectsUrl + data.Id())

		if err != nil {
			return diag.FromErr(err)
		}

		return packProject(data, &project)
	}

	var createProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		project, key, err := unpackProject(data)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = m.(*resty.Client).R().SetBody(project).Post(projectsUrl)
		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId(key)
		return readProject(ctx, data, m)
	}

	var updateProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		project, key, err := unpackProject(data)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = m.(*resty.Client).R().SetBody(project).Put(projectsUrl + data.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		data.SetId(key)
		return readProject(ctx, data, m)
	}

	var deleteProject = func(ctx context.Context, data *schema.ResourceData, m interface{}) diag.Diagnostics {
		_, err := m.(*resty.Client).R().Delete(projectsUrl + data.Id())
		if err != nil {
			return diag.FromErr(err)
		}

		return nil
	}

	return &schema.Resource{
		SchemaVersion: 1,
		CreateContext: createProject,
		ReadContext:   readProject,
		UpdateContext: updateProject,
		DeleteContext: deleteProject,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: projectSchema,
	}
}
