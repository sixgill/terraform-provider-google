package google

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceVertexAIEndpoint() *schema.Resource {
	return &schema.Resource{
		Create: resourceVertexAIEndpointCreate,
		Read:   resourceVertexAIEndpointRead,
		Update: resourceVertexAIEndpointUpdate,
		Delete: resourceVertexAIEndpointDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(6 * time.Minute),
			Update: schema.DefaultTimeout(6 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"display_name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: `The user-defined name of the Endpoint. The name can be up to 128 characters long and can be consist of any UTF-8 characters.`,
			},
			"metadata_schema_uri": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: `Points to a YAML file stored on Google Cloud Storage describing additional information about the Endpoint. The schema is defined as an OpenAPI 3.0.2 Schema Object. The schema files that can be used here are found in gs://google-cloud-aiplatform/schema/endpoint/metadata/.`,
			},
			"encryption_spec": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Description: `Customer-managed encryption key spec for a Endpoint. If set, this Endpoint and all sub-resources of this Endpoint will be secured by this key.`,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"kms_key_name": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Description: `Required. The Cloud KMS resource identifier of the customer managed encryption key used to protect a resource. 
Has the form: projects/my-project/locations/my-region/keyRings/my-kr/cryptoKeys/my-key. The key needs to be in the same region as where the resource is created.`,
						},
					},
				},
			},
			"labels": {
				Type:        schema.TypeMap,
				Computed:    true,
				Optional:    true,
				Description: `A set of key/value label pairs to assign to this Workflow.`,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"region": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
				Description: `The region of the endpoint. eg us-central1`,
			},
			"create_time": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: `The timestamp of when the endpoint was created in RFC3339 UTC "Zulu" format, with nanosecond resolution and up to nine fractional digits.`,
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: `The resource name of the Endpoint. This value is set by Google.`,
			},
			"update_time": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: `The timestamp of when the endpoint was last updated in RFC3339 UTC "Zulu" format, with nanosecond resolution and up to nine fractional digits.`,
			},
			"project": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
		UseJSONNumber: true,
	}
}

func resourceVertexAIEndpointCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}

	obj := make(map[string]interface{})
	displayNameProp, err := expandVertexAIEndpointDisplayName(d.Get("display_name"), d, config)
	if err != nil {
		return err
	} else if v, ok := d.GetOkExists("display_name"); !isEmptyValue(reflect.ValueOf(displayNameProp)) && (ok || !reflect.DeepEqual(v, displayNameProp)) {
		obj["displayName"] = displayNameProp
	}
	labelsProp, err := expandVertexAIEndpointLabels(d.Get("labels"), d, config)
	if err != nil {
		return err
	} else if v, ok := d.GetOkExists("labels"); !isEmptyValue(reflect.ValueOf(labelsProp)) && (ok || !reflect.DeepEqual(v, labelsProp)) {
		obj["labels"] = labelsProp
	}
	encryptionSpecProp, err := expandVertexAIEndpointEncryptionSpec(d.Get("encryption_spec"), d, config)
	if err != nil {
		return err
	} else if v, ok := d.GetOkExists("encryption_spec"); !isEmptyValue(reflect.ValueOf(encryptionSpecProp)) && (ok || !reflect.DeepEqual(v, encryptionSpecProp)) {
		obj["encryptionSpec"] = encryptionSpecProp
	}
	metadataSchemaUriProp, err := expandVertexAIEndpointMetadataSchemaUri(d.Get("metadata_schema_uri"), d, config)
	if err != nil {
		return err
	} else if v, ok := d.GetOkExists("metadata_schema_uri"); !isEmptyValue(reflect.ValueOf(metadataSchemaUriProp)) && (ok || !reflect.DeepEqual(v, metadataSchemaUriProp)) {
		obj["metadataSchemaUri"] = metadataSchemaUriProp
	}

	url, err := replaceVars(d, config, "{{VertexAIBasePath}}projects/{{project}}/locations/{{region}}/endpoints")
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating new Endpoint: %#v", obj)
	billingProject := ""

	project, err := getProject(d, config)
	if err != nil {
		return fmt.Errorf("Error fetching project for Endpoint: %s", err)
	}
	billingProject = project

	// err == nil indicates that the billing_project value was found
	if bp, err := getBillingProject(d, config); err == nil {
		billingProject = bp
	}

	res, err := sendRequestWithTimeout(config, "POST", billingProject, url, userAgent, obj, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return fmt.Errorf("Error creating Endpoint: %s", err)
	}

	// Store the ID now
	id, err := replaceVars(d, config, "{{name}}")
	if err != nil {
		return fmt.Errorf("Error constructing id: %s", err)
	}
	d.SetId(id)

	// Use the resource in the operation response to populate
	// identity fields and d.Id() before read
	var opRes map[string]interface{}
	err = vertexAIOperationWaitTimeWithResponse(
		config, res, &opRes, project, "Creating Endpoint", userAgent,
		d.Timeout(schema.TimeoutCreate))
	if err != nil {
		// The resource didn't actually create
		d.SetId("")
		return fmt.Errorf("Error waiting to create Endpoint: %s", err)
	}

	if err := d.Set("name", flattenVertexAIEndpointName(opRes["name"], d, config)); err != nil {
		return err
	}

	// This may have caused the ID to update - update it if so.
	id, err = replaceVars(d, config, "{{name}}")
	if err != nil {
		return fmt.Errorf("Error constructing id: %s", err)
	}
	d.SetId(id)

	log.Printf("[DEBUG] Finished creating Endpoint %q: %#v", d.Id(), res)

	return resourceVertexAIEndpointRead(d, meta)
}

func resourceVertexAIEndpointRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}

	url, err := replaceVars(d, config, "{{VertexAIBasePath}}{{name}}")
	if err != nil {
		return err
	}

	billingProject := ""

	project, err := getProject(d, config)
	if err != nil {
		return fmt.Errorf("Error fetching project for Endpoint: %s", err)
	}
	billingProject = project

	// err == nil indicates that the billing_project value was found
	if bp, err := getBillingProject(d, config); err == nil {
		billingProject = bp
	}

	res, err := sendRequest(config, "GET", billingProject, url, userAgent, nil)
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("VertexAIEndpoint %q", d.Id()))
	}

	if err := d.Set("project", project); err != nil {
		return fmt.Errorf("Error reading Endpoint: %s", err)
	}

	if err := d.Set("name", flattenVertexAIEndpointName(res["name"], d, config)); err != nil {
		return fmt.Errorf("Error reading Endpoint: %s", err)
	}
	if err := d.Set("display_name", flattenVertexAIEndpointDisplayName(res["displayName"], d, config)); err != nil {
		return fmt.Errorf("Error reading Endpoint: %s", err)
	}
	if err := d.Set("create_time", flattenVertexAIEndpointCreateTime(res["createTime"], d, config)); err != nil {
		return fmt.Errorf("Error reading Endpoint: %s", err)
	}
	if err := d.Set("update_time", flattenVertexAIEndpointUpdateTime(res["updateTime"], d, config)); err != nil {
		return fmt.Errorf("Error reading Endpoint: %s", err)
	}
	if err := d.Set("labels", flattenVertexAIEndpointLabels(res["labels"], d, config)); err != nil {
		return fmt.Errorf("Error reading Endpoint: %s", err)
	}
	if err := d.Set("encryption_spec", flattenVertexAIEndpointEncryptionSpec(res["encryptionSpec"], d, config)); err != nil {
		return fmt.Errorf("Error reading Endpoint: %s", err)
	}
	if err := d.Set("metadata_schema_uri", flattenVertexAIEndpointMetadataSchemaUri(res["metadataSchemaUri"], d, config)); err != nil {
		return fmt.Errorf("Error reading Endpoint: %s", err)
	}

	return nil
}

func resourceVertexAIEndpointUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}

	billingProject := ""

	project, err := getProject(d, config)
	if err != nil {
		return fmt.Errorf("Error fetching project for Endpoint: %s", err)
	}
	billingProject = project

	obj := make(map[string]interface{})
	displayNameProp, err := expandVertexAIEndpointDisplayName(d.Get("display_name"), d, config)
	if err != nil {
		return err
	} else if v, ok := d.GetOkExists("display_name"); !isEmptyValue(reflect.ValueOf(v)) && (ok || !reflect.DeepEqual(v, displayNameProp)) {
		obj["displayName"] = displayNameProp
	}
	labelsProp, err := expandVertexAIEndpointLabels(d.Get("labels"), d, config)
	if err != nil {
		return err
	} else if v, ok := d.GetOkExists("labels"); !isEmptyValue(reflect.ValueOf(v)) && (ok || !reflect.DeepEqual(v, labelsProp)) {
		obj["labels"] = labelsProp
	}

	url, err := replaceVars(d, config, "{{VertexAIBasePath}}{{name}}")
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Updating Endpoint %q: %#v", d.Id(), obj)
	updateMask := []string{}

	if d.HasChange("display_name") {
		updateMask = append(updateMask, "displayName")
	}

	if d.HasChange("labels") {
		updateMask = append(updateMask, "labels")
	}
	// updateMask is a URL parameter but not present in the schema, so replaceVars
	// won't set it
	url, err = addQueryParams(url, map[string]string{"updateMask": strings.Join(updateMask, ",")})
	if err != nil {
		return err
	}

	// err == nil indicates that the billing_project value was found
	if bp, err := getBillingProject(d, config); err == nil {
		billingProject = bp
	}

	res, err := sendRequestWithTimeout(config, "PATCH", billingProject, url, userAgent, obj, d.Timeout(schema.TimeoutUpdate))

	if err != nil {
		return fmt.Errorf("Error updating Endpoint %q: %s", d.Id(), err)
	} else {
		log.Printf("[DEBUG] Finished updating Endpoint %q: %#v", d.Id(), res)
	}

	err = vertexAIOperationWaitTime(
		config, res, project, "Updating Endpoint", userAgent,
		d.Timeout(schema.TimeoutUpdate))

	if err != nil {
		return err
	}

	return resourceVertexAIEndpointRead(d, meta)
}

func resourceVertexAIEndpointDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	userAgent, err := generateUserAgentString(d, config.userAgent)
	if err != nil {
		return err
	}

	billingProject := ""

	project, err := getProject(d, config)
	if err != nil {
		return fmt.Errorf("Error fetching project for Endpoint: %s", err)
	}
	billingProject = project

	url, err := replaceVars(d, config, "{{VertexAIBasePath}}{{name}}")
	if err != nil {
		return err
	}

	var obj map[string]interface{}
	log.Printf("[DEBUG] Deleting Endpoint %q", d.Id())

	// err == nil indicates that the billing_project value was found
	if bp, err := getBillingProject(d, config); err == nil {
		billingProject = bp
	}

	res, err := sendRequestWithTimeout(config, "DELETE", billingProject, url, userAgent, obj, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return handleNotFoundError(err, d, "Endpoint")
	}

	err = vertexAIOperationWaitTime(
		config, res, project, "Deleting Endpoint", userAgent,
		d.Timeout(schema.TimeoutDelete))

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Finished deleting Endpoint %q: %#v", d.Id(), res)
	return nil
}

func flattenVertexAIEndpointName(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	return v
}

func flattenVertexAIEndpointDisplayName(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	return v
}

func flattenVertexAIEndpointCreateTime(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	return v
}

func flattenVertexAIEndpointUpdateTime(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	return v
}

func flattenVertexAIEndpointLabels(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	return v
}

func flattenVertexAIEndpointEncryptionSpec(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	if v == nil {
		return nil
	}
	original := v.(map[string]interface{})
	if len(original) == 0 {
		return nil
	}
	transformed := make(map[string]interface{})
	transformed["kms_key_name"] =
		flattenVertexAIEndpointEncryptionSpecKmsKeyName(original["kmsKeyName"], d, config)
	return []interface{}{transformed}
}
func flattenVertexAIEndpointEncryptionSpecKmsKeyName(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	return v
}

func flattenVertexAIEndpointMetadataSchemaUri(v interface{}, d *schema.ResourceData, config *Config) interface{} {
	return v
}

func expandVertexAIEndpointDisplayName(v interface{}, d TerraformResourceData, config *Config) (interface{}, error) {
	return v, nil
}

func expandVertexAIEndpointLabels(v interface{}, d TerraformResourceData, config *Config) (map[string]string, error) {
	if v == nil {
		return map[string]string{}, nil
	}
	m := make(map[string]string)
	for k, val := range v.(map[string]interface{}) {
		m[k] = val.(string)
	}
	return m, nil
}

func expandVertexAIEndpointEncryptionSpec(v interface{}, d TerraformResourceData, config *Config) (interface{}, error) {
	l := v.([]interface{})
	if len(l) == 0 || l[0] == nil {
		return nil, nil
	}
	raw := l[0]
	original := raw.(map[string]interface{})
	transformed := make(map[string]interface{})

	transformedKmsKeyName, err := expandVertexAIEndpointEncryptionSpecKmsKeyName(original["kms_key_name"], d, config)
	if err != nil {
		return nil, err
	} else if val := reflect.ValueOf(transformedKmsKeyName); val.IsValid() && !isEmptyValue(val) {
		transformed["kmsKeyName"] = transformedKmsKeyName
	}

	return transformed, nil
}

func expandVertexAIEndpointEncryptionSpecKmsKeyName(v interface{}, d TerraformResourceData, config *Config) (interface{}, error) {
	return v, nil
}

func expandVertexAIEndpointMetadataSchemaUri(v interface{}, d TerraformResourceData, config *Config) (interface{}, error) {
	return v, nil
}
