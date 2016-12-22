# terraform-provider-cloudfoundry  [![Build Status](https://travis-ci.org/orange-cloudfoundry/terraform-provider-cloudfoundry.svg?branch=master)](https://travis-ci.org/orange-cloudfoundry/terraform-provider-cloudfoundry)

A terraform provider to manage a Cloud Foundry instance.

You can easily manage a Cloud Foundry instance with terraform file(s), you can actually manage:
- [Organizations](#organizations)
- [Spaces](#spaces)
- [Quotas](#quotas) (Space and Organization ones)
- [Security groups](#security-groups) (On space, staging or running)
- [Buildpacks](#buildpacks)
- [Service brokers](#service-brokers) ([Support gpg encryption on password](#enable-password-encryption))



## Motivations

A Cloud Foundry administrator would like to have always an expected Cloud Foundry instances. He always want to have the right buildpacks, the right orgs, the right spaces, the right service brokers available on his instance. 


Currently, there is no automatic or easy system to manage an entire cloud foundry instance. When a cloud foundry instance is used inside a company this type of system become more and more needed to ensure that everything will work over the time.


We choose a terraform provider to be able to manage a cloud foundry instance because it is already used in the community to create a cloud foundry but also because it provides an easy way to interact with different system. 


Terraform has also been experienced in this case of use case (e.g.: github provider https://www.terraform.io/docs/providers/github/index.html ). 

## Installations

**Requirements:** You need, of course, terraform (**>=0.7**) which is available here: https://www.terraform.io/downloads.html

### Automatic

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

## Resources

### Organizations

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

### Spaces

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
- **org_id**: (**Required**) Organization id created from resource [cloudfoundry_organization](#organizations).
- **allow_ssh**: *(Optional, default: `true`)* Set to `false` to remove ssh access on app instances inside this space.
- **sec_groups**: *(Optional, default: `null`)* This is a list of security groups id created from [cloudfoundry_sec_group](#security-groups), it will bind each security group on this space.
- **quota_id**: *(Optional, default: `null`)* Give a quota id (created from resource [cloudfoundry_quota](#quotas)) to set a quota on this space.

### Quotas

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
- **org_id**: *(Optional, default: `null`)* If set to an organization id created from resource [cloudfoundry_organization](#organizations), it will be considered as organization quota, else it will be a space quota.
- **total_memory**: *(Optional, default: `20G`)* Total amount of memory a space can have (e.g. 1024M, 1G, 10G).
- **total_instance_memory**: *(Optional, default: `-1`)* Maximum amount of memory an application instance can have (e.g. 1024M, 1G, 10G). -1 represents an unlimited amount.
- **routes**: *(Optional, default: `2000`)* Total number of routes that a space can have.
- **service_instances**: *(Optional, default: `200`)* Total number of service instances which can be created that a space can have.
- **app_instances**: *(Optional, default: `-1`)* Total number of application instances that a space can have. -1 represents an unlimited amount.
- **app_allow_paid_service_plans**: *(Optional, default: `true`)* Can provision instances of paid service plans.
- **reserved_route_ports**: *(Optional, default: `0`)* Maximum number of routes that may be created with reserved ports in a space.

### Security groups

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

### Buildpacks

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


### Service brokers

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
- **service_access**: (**Required**) Add service access as many as you need, service access make you service broker accessible on marketplace:
  - **service**: (**Required**) Service name from your service broker catalog to activate. **Note**: if there is only service in your service access it will enable all plan on all orgs on your Cloud Foundry.
  - **plan**: *(Optional, default: `null`)* Plan from your service broker catalog attached to this service to activate. **Note**: if no `org_id` is given it will enable this plan on all orgs.
  - **org_id**: *(Optional, default: `null`)* Org id created from resource [cloudfoundry_organization](#organizations) to activate this service. **Note**: if no `plan` is given it will all plans on this org.
  
**BUG FOUND**: if you set both `plan` and `org_id` in your `service_access` Cloud Foundry will enable all plans on this org. It's maybe only on the version of Cloud Foundry I am. Feedbacks are needed on other versions.

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
