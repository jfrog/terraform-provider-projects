package projects

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccProjectMember(t *testing.T) {
	name := "tftestprojects" + randSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(randSeq(6))

	username1 := "user1"
	email1 := username1 + "@tempurl.org"
	username2 := "user2"
	email2 := username2 + "@tempurl.org"
	developeRole := "developer"
	contributorRole := "contributor"

	params := map[string]interface{}{
		"name":            name,
		"project_key":     projectKey,
		"username1":       username1,
		"username2":       username2,
		"developeRole":    developeRole,
		"contributorRole": contributorRole,
	}

	initialConfig := executeTemplate("TestAccProjectUser", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			member {
				name = "{{ .username1 }}"
				roles = ["{{ .developeRole }}"]
			}
		}
	`, params)

	addUserConfig := executeTemplate("TestAccProjectMember", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			member {
				name = "{{ .username1 }}"
				roles = ["{{ .developeRole }}", "{{ .contributorRole }}"]
			}

			member {
				name = "{{ .username2 }}"
				roles = ["{{ .contributorRole }}"]
			}
		}
	`, params)

	noUserConfig := executeTemplate("TestAccProjectMember", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}
		}
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createTestUser(t, projectKey, username1, email1)
			createTestUser(t, projectKey, username2, email2)
		},
		CheckDestroy: verifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			deleteTestUser(t, projectKey, username1)
			deleteTestUser(t, projectKey, username2)
			resp, err := verifyProject(id, request)

			return resp, err
		}),
		ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "member.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "member.0.name", username1),
					resource.TestCheckResourceAttr(resourceName, "member.0.roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "member.0.roles.0", developeRole),
				),
			},
			{
				Config: addUserConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "member.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "member.0.name", username1),
					resource.TestCheckResourceAttr(resourceName, "member.0.roles.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "member.0.roles.0", contributorRole),
					resource.TestCheckResourceAttr(resourceName, "member.0.roles.1", developeRole),
					resource.TestCheckResourceAttr(resourceName, "member.1.name", username2),
					resource.TestCheckResourceAttr(resourceName, "member.1.roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "member.1.roles.0", contributorRole),
				),
			},
			{
				Config: noUserConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckNoResourceAttr(resourceName, "member"),
				),
			},
		},
	})
}

func TestAccProjectGroup(t *testing.T) {
	name := "tftestprojects" + randSeq(10)
	resourceName := "project." + name
	projectKey := strings.ToLower(randSeq(6))

	group1 := "group1"
	group2 := "group2"
	developeRole := "developer"
	contributorRole := "contributor"

	params := map[string]interface{}{
		"name":            name,
		"project_key":     projectKey,
		"group1":          group1,
		"group2":          group2,
		"developeRole":    developeRole,
		"contributorRole": contributorRole,
	}

	initialConfig := executeTemplate("TestAccProjectGroup", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			group {
				name = "{{ .group1 }}"
				roles = ["{{ .developeRole }}"]
			}
		}
	`, params)

	addGroupConfig := executeTemplate("TestAccProjectGroup", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}

			group {
				name = "{{ .group1 }}"
				roles = ["{{ .developeRole }}", "{{ .contributorRole }}"]
			}

			group {
				name = "{{ .group2 }}"
				roles = ["{{ .contributorRole }}"]
			}
		}
	`, params)

	noGroupConfig := executeTemplate("TestAccProjectGroup", `
		resource "project" "{{ .name }}" {
			key = "{{ .project_key }}"
			display_name = "{{ .name }}"
			description = "test description"
			admin_privileges {
				manage_members = true
				manage_resources = true
				index_resources = true
			}
		}
	`, params)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			createTestGroup(t, projectKey, group1)
			createTestGroup(t, projectKey, group2)
		},
		CheckDestroy: verifyDeleted(resourceName, func(id string, request *resty.Request) (*resty.Response, error) {
			deleteTestGroup(t, projectKey, group1)
			deleteTestGroup(t, projectKey, group2)
			resp, err := verifyProject(id, request)

			return resp, err
		}),
		ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config: initialConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "group.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "group.0.name", group1),
					resource.TestCheckResourceAttr(resourceName, "group.0.roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "group.0.roles.0", developeRole),
				),
			},
			{
				Config: addGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckResourceAttr(resourceName, "group.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "group.0.name", group1),
					resource.TestCheckResourceAttr(resourceName, "group.0.roles.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "group.0.roles.0", contributorRole),
					resource.TestCheckResourceAttr(resourceName, "group.0.roles.1", developeRole),
					resource.TestCheckResourceAttr(resourceName, "group.1.name", group2),
					resource.TestCheckResourceAttr(resourceName, "group.1.roles.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "group.1.roles.0", contributorRole),
				),
			},
			{
				Config: noGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "key", fmt.Sprintf("%s", params["project_key"])),
					resource.TestCheckResourceAttr(resourceName, "display_name", name),
					resource.TestCheckResourceAttr(resourceName, "description", "test description"),
					resource.TestCheckNoResourceAttr(resourceName, "group"),
				),
			},
		},
	})
}
