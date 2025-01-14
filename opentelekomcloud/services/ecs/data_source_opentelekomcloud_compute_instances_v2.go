package ecs

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/compute/v2/servers"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/cfg"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/common/fmterr"
	"github.com/opentelekomcloud/terraform-provider-opentelekomcloud/opentelekomcloud/helper/hashcode"
)

func DataSourceComputeInstancesV2() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceComputeInstancesV2Read,

		Schema: map[string]*schema.Schema{
			"instance_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"flavor_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"project_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"limit": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  100,
			},
			"status": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"flavor_name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"image_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"key_pair": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"instances": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"image_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"image_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"flavor_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"flavor_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"status": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"availability_zone": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"key_pair": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"project_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"security_group_ids": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"user_data": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"system_disk_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"public_ip": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"network": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"uuid": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"name": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"port": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"fixed_ip_v4": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"fixed_ip_v6": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"mac": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"tags": {
							Type:     schema.TypeMap,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func dataSourceComputeInstancesV2Read(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*cfg.Config)
	log.Print("[DEBUG] Creating compute client")
	client, err := config.ComputeV2Client(config.GetRegion(d))
	if err != nil {
		return fmterr.Errorf(errCreateV2Client, err)
	}

	opts := servers.ListOpts{
		Limit:    d.Get("limit").(int),
		Name:     d.Get("name").(string),
		Flavor:   d.Get("flavor_id").(string),
		Status:   d.Get("status").(string),
		TenantID: d.Get("project_id").(string),
	}
	var allServers []servers.Server
	allPages, err := servers.List(client, opts).AllPages()
	if err != nil {
		return fmterr.Errorf("unable to retrieve OpenTelekomCloud servers: %w", err)
	}
	allServers, err = servers.ExtractServers(allPages)
	if err != nil {
		return fmterr.Errorf("unable to retrieve OpenTelekomCloud servers: %w", err)
	}

	instances := make([]servers.Server, 0, len(allServers))
	ids := make([]string, 0, len(allServers))

	for _, server := range allServers {
		if serverId, ok := d.GetOk("instance_id"); ok && serverId != server.ID {
			continue
		}
		if flavorName, ok := d.GetOk("flavor_name"); ok && flavorName != server.Flavor["Name"] {
			continue
		}
		if imageId, ok := d.GetOk("image_id"); ok && imageId != server.Image["ID"] {
			continue
		}
		if keypair, ok := d.GetOk("key_pair"); ok && keypair != server.KeyName {
			continue
		}
		instances = append(instances, server)
		ids = append(ids, server.ID)
	}

	d.SetId(hashcode.Strings(ids))

	result := make([]map[string]interface{}, len(instances))
	for i, item := range instances {
		var secGrpNames []string
		for _, sg := range item.SecurityGroups {
			secGrpNames = append(secGrpNames, sg["name"].(string))
		}

		server := map[string]interface{}{
			"id":                 item.ID,
			"name":               item.Name,
			"image_id":           item.Image["id"],
			"flavor_id":          item.Flavor["id"],
			"status":             item.Status,
			"key_pair":           item.KeyName,
			"security_group_ids": secGrpNames,
			"project_id":         item.TenantID,
		}

		result[i] = server
	}

	if err := d.Set("instances", result); err != nil {
		return diag.Errorf("error setting cloud server list: %s", err)
	}

	return nil
}
