# terraform-provider-cloudfoundry  [![Build Status](https://travis-ci.org/orange-cloudfoundry/terraform-provider-cloudfoundry.svg?branch=master)](https://travis-ci.org/orange-cloudfoundry/terraform-provider-cloudfoundry)

This is a work in progress, meaning the syntax may change in the future, and the implementation is being hardened. The [design proposal document](https://docs.google.com/document/d/1d5XUPu08wLNTdCLYz-Fi--ogFZdtn3f_BcR-gzW6AXM/edit#) provides more background on the intended use-cases, and the next potential resources to be added. Feedback and contributions are welcomed.

This terraformp provider supports the use-case of managing a Cloud Foundry instance, with current support for:
- [Organizations](#organizations)
- [Spaces](#spaces)
- [Quotas](#quotas) (Space and Organization ones)
- [Security groups](#security-groups) (On space, staging or running)
- [Buildpacks](#buildpacks)
- [Feature flags](#feature-flags)
- [Services](#services)
- [Domains](#domains)
- [Routes](#routes)
- [Isolation Segments](#isolation-segments)
- [Stacks](#stacks)
- [Environment Variable Group](#environment-variable-group)
- [Applications](#applications)
- [Service brokers](#service-brokers) ([Support gpg encryption on password](#enable-password-encryption))

## Installations

**Requirements:** You need, of course, terraform (**>=0.8**) which is available here: https://www.terraform.io/downloads.html

### Automatic

To install a specific version, set PROVIDER_CLOUDFOUNDRY_VERSION before executing the following command

```bash
$ export PROVIDER_CLOUDFOUNDRY_VERSION="v0.8.1"
```

#### via curl

```bash
$ sh -c "$(curl -fsSL https://raw.github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/master/bin/install.sh)"
```

#### via wget

```bash
$ sh -c "$(wget https://raw.github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/master/bin/install.sh -O -)"
```

### Manually

1. Get the build for your system in releases: https://github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/releases/latest
2. Create a `providers` directory inside terraform user folder: `mkdir -p ~/.terraform.d/providers`
3. Move the provider previously downloaded in this folder: `mv /path/to/download/directory/terraform-provider-cloudfoundry ~/.terraform.d/providers`
4. Ensure provider is executable: `chmod +x ~/.terraform.d/providers/terraform-provider-cloudfoundry`
5. add `providers` path to your `.terraformrc`:
```bash
cat <<EOF > ~/.terraformrc
providers {
    cloudfoundry = "/full/path/to/.terraform.d/providers/terraform-provider-cloudfoundry"
}
EOF
```

6. you can now performs any terraform action on Cloud Foundry resources

## provider configuration

```tf
provider "cloudfoundry" {
  api_endpoint = "https://api.of.your.cloudfoundry.com"
  username = "user"
  password = "mypassword"
  skip_ssl_validation = true
  enc_private_key = "${file("secring_b64.gpg")}"
  enc_passphrase = "mypassphrase"
  verbose = false
  user_access_token = "bearer key"
  user_refresh_token = "bearer key"
}
```

- **name**: (**Required**, *Env Var: `CF_API`*) Your Cloud Foundry api url.
- **username**: *(Optional, default: `null`, Env Var: `CF_USERNAME`)* The username of an admin user. (Optional if you use an access token)
- **password**: *(Optional, default: `null`, Env Var: `CF_PASSWORD`)* The password of an admin user. (Optional if you use an access token)
- **skip_ssl_validation**: *(Optional, default: `false`)* Set to true to skip verification of the API endpoint. Not recommended!.
- **enc_private_key**: *(Optional, default: `null`, Env Var: `CF_ENC_PRIVATE_KEY`)* A GPG private key(s) generate from `gpg --export-secret-key -a <real name>` . Need a passphrase with `enc_passphrase`..
- **enc_passphrase**: *(Optional, default: `null`, Env Var: `CF_ENC_PASSPHRASE`)* The passphrase for your gpg key.
- **verbose**: *(Optional, default: `null`)* Set to true to see requests sent to Cloud Foundry. (Use `TF_LOG=1` to see them)
- **user_access_token**: *(Optional, default: `null`, Env Var: `CF_TOKEN`)* The OAuth token used to connect to a Cloud Foundry. (Optional if you use 'username' and 'password')
- **user_refresh_token**: *(Optional, default: `null`)* The OAuth refresh token used to refresh your token.

## Resources and Data sources

----

### Organizations

#### Resource

```tf
resource "cloudfoundry_organization" "org_mysuperorg" {
  name = "mysuperorg"
  is_system_domain = true
  quota_id = "${cloudfoundry_quota.quota_mysuperquota.id}"
}
```

- **name**: (**Required**) Name of your organization.
- **is_system_domain**: *(Optional, default: `false`)* set it to true only if this organization is a system_domain organization, it will prevent deletion on Cloud Foundry.
- **quota_id**: *(Optional, default: `null`)* Give a quota id (created from resource [cloudfoundry_quota](#quotas)) to set a quota on this org.

#### Data source

**Note**: every parameters from resource which are not used here are marked as computed and will be filled.

```tf
data "cloudfoundry_organization" "org_mysuperorg" {
  name = "mysuperorg"
}
```

- **name**: (**Required**) Name of your organization.

----

### Spaces

#### Resource

```tf
resource "cloudfoundry_space" "space_mysuperspace" {
    name = "mysuperspace"
    org_id = "${cloudfoundry_organization.org_mysuperorg.id}"
    quota_id = "${cloudfoundry_quota.quota_mysuperquota.id}"
    sec_groups = ["${cloudfoundry_sec_group.sec_group_mysupersecgroup.id}"]
    allow_ssh = true
}
```

- **name**: (**Required**) Name of your space.
- **org_id**: (**Required**) Organization id created from resource or data source [cloudfoundry_organization](#organizations).
- **allow_ssh**: *(Optional, default: `true`)* Set to `false` to remove ssh access on app instances inside this space.
- **sec_groups**: *(Optional, default: `null`)* This is a list of security groups id created from [cloudfoundry_sec_group](#security-groups), it will bind each security group on this space.
- **quota_id**: *(Optional, default: `null`)* Give a quota id (created from resource [cloudfoundry_quota](#quotas)) to set a quota on this space.

#### Data source

**Note**: every parameters from resource which are not used here are marked as computed and will be filled.

```tf
data "cloudfoundry_space" "space_mysuperspace" {
    name = "mysuperspace"
    org_id = "${cloudfoundry_organization.org_mysuperorg.id}"
}
```

- **name**: (**Required**) Name of your space.
- **org_id**: (**Required**) Organization id created from resource or data source [cloudfoundry_organization](#organizations).

----

### Quotas

#### Resource

**Note**: There is two kinds of quotas inside Cloud Foundry: a space's quota, an organization's quota. This resource is able to find what kind of quota you defined. If you omit *`org_id`* the resource will consider this 
quota as an organization's quota. With it will consider it's a space's quota.

```tf
resource "cloudfoundry_quota" "quota_for_ahalet" {
  name = "quotaAhalet"
  org_id = "${cloudfoundry_organization.org_mysuperorg.id}"
  total_memory = "10G"
  instance_memory = "1G"
  routes = 200
  service_instances = 10
  app_instances = -1
  allow_paid_service_plans = true
  reserved_route_ports = 0
}
```

- **name**: (**Required**) Name of your quota.
- **org_id**: *(Optional, default: `null`)* If set to an organization id created from resource or data source [cloudfoundry_organization](#organizations), it will be considered as organization quota, else it will be a space quota.
- **total_memory**: *(Optional, default: `20G`)* Total amount of memory a space can have (e.g. 1024M, 1G, 10G).
- **total_instance_memory**: *(Optional, default: `-1`)* Maximum amount of memory an application instance can have (e.g. 1024M, 1G, 10G). -1 represents an unlimited amount.
- **routes**: *(Optional, default: `2000`)* Total number of routes that a space can have.
- **service_instances**: *(Optional, default: `200`)* Total number of service instances which can be created that a space can have.
- **app_instances**: *(Optional, default: `-1`)* Total number of application instances that a space can have. -1 represents an unlimited amount.
- **app_allow_paid_service_plans**: *(Optional, default: `true`)* Can provision instances of paid service plans.
- **reserved_route_ports**: *(Optional, default: `0`)* Maximum number of routes that may be created with reserved ports in a space.

#### Data source

**Note**: every parameters from resource which are not used here are marked as computed and will be filled.

```tf
data "cloudfoundry_quota" "quota_for_ahalet" {
  name = "quotaAhalet"
  org_id = "${cloudfoundry_organization.org_mysuperorg.id}"
}
```

- **name**: (**Required**) Name of your quota.
- **org_id**: *(Optional, default: `null`)* If set to an organization id created from resource or data source [cloudfoundry_organization](#organizations), it will be considered as organization quota, else it will be a space quota.

----

### Security groups

#### Resource

```tf
resource "cloudfoundry_sec_group" "sec_group_mysupersecgroup" {
  name = "mysupersecgroup"
  on_staging = false
  on_running = false
  rules {
    protocol = "tcp"
    destination = "10.0.0.2"
    ports = "65000"
    log = false
    description = "my description"
  }
  rules {
    protocol = "icmp"
    destination = "192.0.2.0-192.0.1-4"
    type = 3
    code = 1
  }
  rules {
      protocol = "all"
      destination = "10.0.0.0/24"
      log = true
    }
}
```

- **name**: (**Required**) Name of your security group.
- **on_staging**: *(Optional, default: `false`)* Set to true to apply this security group during staging an app.
- **on_running**: *(Optional, default: `false`)* Set to true to apply this security group during running an app.
- **rules**: *(Optional, default: `null`)* Add rules as many as you need: 
  - **protocol**: (**Required**) The protocol to use, it can be `tcp`, `udp`, `icmp`, or `all`
  - **destination**: *(Optional, default: `null`)* A single IP address, an IP address range (e.g. 192.0.2.0-192.0.1-4), or a CIDR block to allow network access to.
  - **ports**: *(Optional, default: `null`)* A single port, multiple comma-separated ports, or a single range of ports that can receive traffic, e.g. `"443"`, `"80,8080,8081"`, `"8080-8081"`. Required when `protocol` is `tcp` or `udp`. 
  - **code**: *(Optional, default: `null`)* ICMP code. Required when `protocol` is `icmp`. 
  - **type**: *(Optional, default: `null`)* ICMP type. Required when `protocol` is `icmp`. 
  - **log**: *(Optional, default: `false`)* Set to `true` to enable logging. For more information about how to configure system logs to be sent to a syslog drain, see [Using Log Management Services](https://docs.cloudfoundry.org/devguide/services/log-management.html) topic.
  - **description**: *(Optional, default: `null`)* This is an optional field that contains useful text for operators to manage security group rules. This field is available in Cloud Foundry v238 and later.

#### Data source

**Note**: every parameters from resource which are not used here are marked as computed and will be filled.

```tf
data "cloudfoundry_sec_group" "sec_group_mysupersecgroup" {
  name = "mysupersecgroup"
}
```

- **name**: (**Required**) Name of your security group.

----

### Buildpacks

#### Resource

```tf
resource "cloudfoundry_buildpack" "buildpack_mysuperbuildpack" {
  name = "mysuperbuildpack"
  path = "https://github.com/cloudfoundry/staticfile-buildpack/releases/download/v1.3.13/staticfile_buildpack-cached-v1.3.13.zip"
  position = 13
  locked = false
  enabled = false
}
```

- **name**: (**Required**) Name of your buildpack. **Note**: if there is only name inside your buildpack the provider will consider your buildpack as a system managed buildpack (e.g.: `php_buildpack`, `java_buildpack`), so if you remove it from your tf file it will not be removed from your Cloud Foundry.
- **path**: *(Optional, default: `null`)* Path should be a zip file, a url to a zip file, or a local directory which contains your buildpack code.
- **position**: *(Optional, default: `null`)* Position is a positive integer, sets priority, and is sorted from lowest to highest.
- **enabled**: *(Optional, default: `true`)* Set to `false` to disable the buildpack to be used for staging.
- **locked**: *(Optional, default: `false`)* Set to `true` to lock the buildpack to prevent updates.

#### Data source

**Note**: every parameters from resource which are not used here are marked as computed and will be filled.

```tf
resource "cloudfoundry_buildpack" "buildpack_mysuperbuildpack" {
  name = "mysuperbuildpack"
}
```

- **name**: (**Required**) Name of your buildpack.

----

### Feature flags

#### Resource

```tf
resource "cloudfoundry_feature_flags" "feature_flags" {
  diego_docker = true
  custom_flag {
    name = "my_flag"
    enabled = true
  }
}
```
List of default feature flags:
- **user_org_creation**: *(Optional, default: `false`)*
- **private_domain_creation**: *(Optional, default: `true`)*
- **app_bits_upload**: *(Optional, default: `true`)*
- **app_scaling**: *(Optional, default: `true`)*
- **route_creation**: *(Optional, default: `true`)*
- **service_instance_creation**: *(Optional, default: `true`)*
- **diego_docker**: *(Optional, default: `false`)*
- **set_roles_by_username**: *(Optional, default: `true`)*
- **unset_roles_by_username**: *(Optional, default: `true`)*
- **task_creation**: *(Optional, default: `false`)*
- **env_var_visibility**: *(Optional, default: `true`)*
- **space_scoped_private_broker_creation**: *(Optional, default: `true`)*
- **space_developer_env_var_visibility**: *(Optional, default: `true`)*

Custom flags made for feature flags not in the default resource:
- **custom_flag**: *(Optional, default: `null`)* Add cutom feature flags as many as you need: 
  - **name**: (**Required**) Name of the feature
  - **enabled**: (**Required**) Set to `true` to enable the feature in your cloud foundry.

#### Data source

**Feature flags cannot be used as data source**

----

### Services

#### Resource

Service from marketplace:

```tf
resource "cloudfoundry_service" "svc_db" {
  name = "my-db"
  space_id = "${cloudfoundry_space.space_mysuperspace.id}"
  service = "p-mysql"
  plan = "100mb"
  params = "{ \"my-param\": 1}"
  update_params = "{ \"my-param\": 1}"
  tags = [ "tag1", "tag2" ]
}
```

An user provided service:

```tf
resource "cloudfoundry_service" "svc_ups" {
  name = "my-ups"
  space_id = "${cloudfoundry_space.space_mysuperspace.id}"
  user_provided = true
  params = "{ \"my-credential\": 1}"
  route_service_url = "http://my.route.com"
  syslog_drain_url = "http://my.syslog.com"
  tags = [ "tag1", "tag2" ]
}
```

- **name**: (**Required**) Name of your service.
- **space_id**: (**Required**) Space id created from resource or data source [cloudfoundry_space](#spaces) to register service inside.
- **user_provided**: *(Optional, default: `false`)* Set to `true` to create an user provided service. **Note**: `service` and `plan` params will not be used.
- **params**: *(Optional, default: `null`)* Must be json, if it's an user provided service it will be credential for your service instead it will be params sent to service broker when creating service.
- **update_params**: *(Optional, default: `null`)* Must be json, Params sent to service broker when updating service.
- **tags**: *(Optional, default: `null`)* list of tags for your service.
- **service**: (**Required when not user provided service**) name of service from marketplace.
- **plan**: (**Required when not user provided service**) name of the plan to use.
- **route_service_url**: *(Optional, default: `null`)* Only works for user provided, an url to create a [route service](https://docs.cloudfoundry.org/services/route-services.html)
- **syslog_drain_url**: *(Optional, default: `null`)* Only works for user provided, an url to drain logs as a service on an app.

#### Data source

**Note**: every parameters from resource which are not used here are marked as computed and will be filled, except:
- `params`
- `update_params`

```tf
data "cloudfoundry_service" "svc_ups" {
  name = "my-ups"
  space_id = "${cloudfoundry_space.space_mysuperspace.id}"
}
```

- **name**: (**Required**) Name of your service.
- **space_id**: (**Required**) Space id created from resource or data source [cloudfoundry_space](#spaces) to register service inside.

----

### Domains

#### Resource

```tf
resource "cloudfoundry_domain" "domain_mydomain" {
  name = "my.domain.com"
  org_owner_id = "${cloudfoundry_organization.org_mysuperorg.id}"
  router_group = "default-router"
  orgs_shared_id = ["${cloudfoundry_organization.org_mysecondorg.id}"]
  shared = false
}
```

- **name**: (**Required**) Your domain name.
- **org_owner_id**: (**Required if not shared**) Organization id created from resource or data source which owned the domain [cloudfoundry_organization](#organizations).
- **orgs_shared_id**: *(Optional, default: `null`)* Set of organization id which can have access to domain. **Note**: Only can used when not a shared domain
- **router_group**: *(Optional, default: `null`)* Routes for this domain will be configured only on the specified router group. **Note**: Only when when it's a shared domain
- **shared**: *(Optional, default: `false`)* If `True` this domain will be a shared domain.

#### Data source

**Note**: every parameters from resource which are not used here are marked as computed and will be filled.

```tf
data "cloudfoundry_domain" "domain_mydomain" {
  name = "my.domain.com"
  org_owner_id = "${cloudfoundry_organization.org_mysuperorg.id}"
  first = false
}
```

- **name**: *(Optional if `first` param set to `true`, default: `null`)* Your domain name.
- **first**: *(Optional, default: `null`)* If set to `true` parameter `name` become unnecessary and will give the first domain found in your Cloud Foundry (it will be the first shared domain if `org_owner_id` is not set).
- **org_owner_id**: (**Required if not shared**) Organization id created from resource or data source which owned the domain [cloudfoundry_organization](#organizations).

----

### Routes

#### Resource

```tf
resource "cloudfoundry_route" "route_superroute" {
  hostname = "superroute"
  space_id = "${cloudfoundry_space.space_mysuperspace.id}"
  domain_id = "${cloudfoundry_domain.domain_mydomain.id}"
  port = -1
  path = ""
  service_id = "${cloudfoundry_service.svc_ups.id}"
  service_params = "{ \"my-param\": 1}"
}
```

- **name**: (**Required**) Your hostname.
- **domain_id**: (**Required**) Domain id created from resource or data source [domains](#domains).
- **space_id**: (**Required**) Space id created from resource or data source [cloudfoundry_space](#spaces) to register route inside.
- **port**: *(Optional, default: `-1`)* Set a port for your route (only works with a tcp domain). **Note**: If `0` a random port will be chose
- **path**: *(Optional, default: `null`)* Set a path for your route (only works with a http(s) domain).
- **service_id**: *(Optional, default: `null`)* Set a service id created from resource or data source [services](#services) this will bind a route service on your route. **Note**: It obviously needs a service which is a route service.
- **service_params**: *(Optional, default: `null`)*  Must be in json, set params to send to service when binding on it.
- **protocol**: *(Optional, default: `null`)*  This parameter is only for uri computed parameter it permits to override 
  the protocol when generating uri (generated uri will use always `https` protocol when it's an http route, you can found useful to force in `http`).
- **uri**: *(Computed)*  This is an uri generated by the resource, you can use this for service brokers resource for example. **Note**: It autodetects when it's an http route or a tcp route.

#### Data source

**Note**: every parameters from resource which are not used here are marked as computed and will be filled, except:
- `service_params`

```tf
resource "cloudfoundry_route" "route_superroute" {
  hostname = "superroute"
  domain_id = "${cloudfoundry_domain.domain_mydomain.id}"
  port = -1
  path = ""
}
```

- **name**: (**Required**) Your hostname.
- **domain_id**: (**Required**) Domain id created from resource or data source [domains](#domains).
- **port**: *(Optional, default: `-1`)* Set a port for your route (only works with a tcp domain). **Note**: If `0` a random port will be chose
- **path**: *(Optional, default: `null`)* Set a path for your route (only works with a http(s) domain).
- **protocol**: *(Optional, default: `null`)*  This parameter is only for uri computed parameter it permits to override 
  the protocol when generating uri (generated uri will use always `https` protocol when it's an http route, you can found useful to force in `http`).

----

### Isolation segments

**IMPORTANT NOTE**:
- Isolation segments are in development on cloud foundry and only available with cloud controller api V3.
- My actual Cloud Foundry doesn't have isolation segment and resource could **not be tested**
- **Use at your own risk, there is no warranty**

#### Resource

```tf
resource "cloudfoundry_isolation_segment" "my_isolation_segment" {
  name = "isolation_segment_name_set_in_cf_deployment"
  orgs_id = ["${cloudfoundry_organization.org_mysuperorg.id}"]
}
```

- **name**: (**Required**) Isolation segment that you have set on your cloud foundry deployment.
- **orgs_id**: (**Required**) *(Optional, default: `null`)* You can pass a list of organization created from resource or data source [cloudfoundry_organization](#organizations), this will put those organizations in the isolation segment.

#### Data source

**Note**: every parameters from resource which are not used here are marked as computed and will be filled.

```tf
data "cloudfoundry_isolation_segment" "my_isolation_segment" {
  name = "isolation_segment_name_set_in_cf_deployment"
}
```

- **name**: (**Required**) Isolation segment that you have set on your cloud foundry deployment.

----

### Stacks

#### Resource

**Stacks cannot be used as a resource**

#### Data source

**Note**: every parameters from resource which are not used here are marked as computed and will be filled.

```tf
data "cloudfoundry_stack" "my_stack" {
  name = "cflinuxfs2"
  first = false
}
```

- **name**: *(Optional if `first` param set to `true`, default: `null`)* Name of the stack.
- **first**: *(Optional, default: `null`)* If set to `true` parameter `name` become unnecessary and will give the first stack found in your Cloud Foundry.

----

### Environment Variable Group

#### Resource

```tf
resource "cloudfoundry_env_var_group" "env_var_group" {
  env_var {
    key = "myvar1"
    value = "myvalue1"
    running = true
    staging = true
  }
  env_var {
    key = "myvar2"
    value = "myvalue1"
    running = true
    staging = true
  }
}
```

- **env_var**: (**Required**) Add any variable you want to environment variable group:
  - **key**: (**Required**) Env var key.
  - **value**: (**Required**) Env var value.
  - **running**: (**Required**) if set to `true` this env var will be use on all running app.
  - **staging**: (**Required**) if set to `true` this env var will be use during staging step when creating an app.

#### Data source

**Environment Variable Group cannot be used as a data source**

----

### Service brokers

#### Resource

```tf
resource "cloudfoundry_service_broker" "service_broker_mysuperbroker" {
  name = "mysuperbroker"
  url = "http://url.of.my.service.broker.com"
  username = "user"
  password = "mypassword"
  service_access {
    service = "service_name_from_service_broker_catalog"
    plan = "plan_from_service_broker_catalog"
    org_id = "${cloudfoundry_organization.org_mysuperorg.id}"
  }
  service_access {
    service = "service_name_from_service_broker_catalog"
    plan = "plan2_from_service_broker_catalog"
    org_id = "${cloudfoundry_organization.org_mysuperorg.id}"
  }
  #...
}
```

- **name**: (**Required**) Name of your service broker.
- **url**: (**Required**) URL to access to your service broker.
- **username**: *(Optional, default: `null`)* Username to authenticate to your service broker.
- **password**: *(Optional, default: `null`)* Password to authenticate to your service broker. **Note**: you can pass a base 64 encrypted gpg message if you [enabled password encryption](#enable-password-encryption).
- **catalog_sha1**: *(Computed)* Do not modify yourself, this permits to detect a change in the service broker catalog.
- **service_access**: (**Required**) Add service access as many as you need, service access make you service broker accessible on marketplace:
  - **service**: (**Required**) Service name from your service broker catalog to activate. **Note**: if there is only service in your service access it will enable all plan on all orgs on your Cloud Foundry.
  - **plan**: *(Optional, default: `null`)* Plan from your service broker catalog attached to this service to activate. **Note**: if no `org_id` is given it will enable this plan on all orgs.
  - **org_id**: *(Optional, default: `null`)* Org id created from resource or data source [cloudfoundry_organization](#organizations) to activate this service. **Note**: if no `plan` is given it will all plans on this org.
  
**BUG FOUND**: if you set both `plan` and `org_id` in your `service_access` Cloud Foundry will enable all plans on this org. It's maybe only on the version of Cloud Foundry I am. Feedbacks are needed on other versions.

#### Data source

**Note**: every parameters from resource which are not used here are marked as computed and will be filled, except:
- `username`
- `password`

```tf
resource "cloudfoundry_service_broker" "service_broker_mysuperbroker" {
  name = "mysuperbroker"
}
```

- **name**: (**Required**) Name of your service broker.

### Applications

This resource is used in order to deploy and update an application. It can see changes between code you have locally and code you have in your cloud foundry to do the update fastly (It compares a checksum from a chunk of data between local and remotely)

**By default, when updating, your app will never shutdown**. It always use blue-green deployment when app bits changed, rename or scale number of instances instantly and do blue-green restage in all others modification.

As a terraform resource, creating an app give you more control but can also be more painful than using the cli. 
To be painless, [terraform modules](https://www.terraform.io/docs/modules/index.html) can be use to deploy you app like you could do with a `manifest.yml` file. 
This can be found on https://github.com/orange-cloudfoundry/terraform-cloudfoundry-modules

#### Resource

```tf
resource "cloudfoundry_app" "myapp" {
  name = "myapp"
  stack_id = "${data.cloudfoundry_stack.my_stack.id}"
  space_id = "${data.cloudfoundry_space.space_mysuperspace.id}"
  started = true
  instances = 2
  memory = "64M"
  disk_quota = "1G"
  command = ""
  path = "/path/to/folder"
  diego = true
  buildpack = "php_buildpack"
  health_check_type = "port"
  health_check_http_endpoint = ""
  health_check_timeout = ""
  docker_image = ""
  enable_ssh = false
  ports = [8080]
  routes = ["${cloudfoundry_route.route_superroute.id}"]
  services = ["${cloudfoundry_service.svc_db.id}"]
  env_var {
    key = "MY_ENV_KEY"
    value = "myvalue"
  } # you can have, of course, multiple env_var
}
```

- **name**: (**Required**) Name of your application.
- **space_id**: (**Required**) Space id created from resource or data source [spaces](#spaces).
- **stack_id**: (**Required**) Stack id retrieve from data source [Stacks](#stacks).
- **path**: (**Required**) Path to a folder which contains application code or url to a zip/jar file
- **started**: *(Optional, default: `true`)* State of your application (should be start or not).
- **instances**: *(Optional, default: `1`)*  The number of instances of the app to run.
- **memory**: *(Optional, default: `1G`)* The amount of memory each instance should have.
- **disk_quota**: *(Optional, default: `1G`)* The maximum amount of disk available to an instance of an app.
- **command**: *(Optional, default: `NULL`)* The command to start an app after it is staged.
- **diego**: *(Optional, default: `true`)* Use diego to stage and to run when available (Diego should be always available because DEA is not supported anymore).
- **buildpack**: *(Optional, default: `NULL`)* Buildpack to build the app. 3 options: a) Blank means autodetection; b) A Git Url pointing to a buildpack; c) Name of an installed buildpack.
- **health_check_type**: *(Optional, default: `port`)* Type of health check to perform. Others values are: 
  - http (Diego only)
  - port
  - process
  - none
- **health_check_http_endpoint**: *(Optional, default: `NULL`)* Endpoint called to determine if the app is healthy. (Can  be use only when check type is http)
- **health_check_timeout**: *(Optional, default: `NULL`)* Timeout in seconds for health checking of an staged app when starting up.
- **docker_image**: *(Optional, default: `NULL`)* Name of the Docker image containing the app. The "diego_docker" feature flag must be enabled in order to create Docker image apps.
- **enable_ssh**: *(Optional, default: `false`)* Enable SSHing into the app. Supported for Diego only.
- **ports**: *(Optional, default: `8080` when diego is set to `true`)* List of ports on which application may listen. Overwrites previously configured ports. 
  Ports must be in range 1024-65535. Supported for Diego only. (**Note**: This is a copy of the default behaviour of cloud foundry cli, it always create a default port to 8080 when using diego backend)
- **routes**: *(Optional, default: `NULL`)* List of route guid retrieve from resource or data source [routes](#routes) to attach routes to your app.  
- **services**: *(Optional, default: `NULL`)* List of service guid retrieve from resource or data source [services](#services) to bind services to your app.
- **env_var**: *(Optional, default: `NULL`)* Add any variable you want to the app environment:
  - **key**: (**Required**) Env var key.
  - **value**: (**Required**) Env var value.
- **no_blue_green_restage**: *(Optional, default: `false`)* If set to `true` no blue green restage will be performed (it will restart the app).
- **no_blue_green_deploy**: *(Optional, default: `false`)* If set to `true` no blue green deployment will be performed.

#### Data source

**Note**: every parameters from resource which are not used here are marked as computed and will be filled.

```tf
resource "cloudfoundry_service_broker" "myapp" {
  name = "mysuperbroker"
  space_id = "${data.cloudfoundry_space.space_mysuperspace.id}"
}
```

- **name**: (**Required**) Name of your service broker. If `space_id` it will try to find the first matching app found in all spaces you have access to.
- **space_id**: *(Optional, default: `null`)* Space id created from resource or data source [spaces](#spaces).

## Enable password encryption

You can use gpg encryption to encrypt your service broker password.

### Create a private key for the provider

**Requirements**: you will need to have `gpg` on your system.

1. run `gpg --gen-key`, next steps will assume that you put `cloudfoudry` as real name. (Do not forget to remember your passphrase!)
2. go on your terraform folder config in command line
3. run `gpg --export-secret-key -a cloudfoudry > private.key`
4. inside provider configuration put those two key/value pairs (you can also copy content of `private.key` and `export CF_ENC_PRIVATE_KEY=content_of_private.key && export CF_ENC_PASSPHRASE=your_passphrase_that_you_remembered:)`): 
```tf
provider "cloudfoundry" {
  enc_private_key = "${file("private.key")}"
  enc_passphrase = "your_passphrase_that_you_remembered:)"
}
```
5. create the public key with `gpg --export -a cloudfoudry > public.key`
6. Share the public key to the rest of your team to let them encrypt password with it (see [Encrypt password](#encrypt-password))
7. you're done

### Encrypt password

1. Get the public key previously created (`public.key`)
2. Import the key with `gpg --import public.key`
3. generate the encrypted password with commands `echo "mypassword" | gpg --encrypt --armor -r cloudfoudry > encrypted_pass.key`
4. Retrieve it from your resource, e.g.:
```tf
resource "cloudfoundry_service_broker" "service_broker_mysuperbroker" {
  name = "mysuperbroker"
  url = "http://url.of.my.service.broker.com"
  username = "user"
  password = "${file("encrypted_pass.key")}"
  service_access {
    service = "service_name_from_service_broker_catalog"
  }
}
```
5. you're done
