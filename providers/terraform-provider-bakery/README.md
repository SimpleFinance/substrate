# bakery\_ebs\_snapshot

Provides a resource for creating AWS EBS snapshots by launching a source AMI and running some provisioning code. The resulting snapshots can be used with the builtin `aws_ami` resource to create a fully functioning custom AMI.

## Example Usage

```hcl
# create a snapshot by starting with ami-01234567 and running the "Hello World" script.
resource "bakery_ebs_snapshot" "base" {
    description = "My Custom Snapshot"
    region = "terah-west-1"
    builder_availability_zone = "terah-west-1a"
    builder_source_ami = "ami-01234567"
    builder_instance_tags = {
        "Name" = "snapshot builder instance"
    }
    tags = {
        "Name" = "my custom snapshot"
    }
    builder_user_data = <<USERDATA
#!/bin/sh
echo Hello World > /hello.txt
shutdown -h now
USERDATA
}

# create an AMI using the snapshot
resource "aws_ami" "base" {
    name = "custom-base-${bakery_ebs_snapshot.base.id}"
    description = "My Custom AMI"
    virtualization_type = "hvm"
    architecture = "x86_64"
    root_device_name = "/dev/xvda"

    # use the snapshot as the root volume
    ebs_block_device {
        device_name = "/dev/xvda"
        snapshot_id = "${bakery_ebs_snapshot.base.id}"
        volume_size = 8
        volume_type = "gp2"
        delete_on_termination = true
    }
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The AWS region in which to create the snapshot..
* `builder_source_ami` - (Required) The source AMI to with which to boot the temporary builder instance.
* `builder_user_data` - (Required) User data with which to start the builder instance. This should be a script or `#cloud-config` stanza that provisions the instance and then shuts it down.
* `builder_availability_zone` - (Required) The AWS AZ within which to start the temporary builder instance.
* `description` - (Optional) A description for the snapshot. Default is `""`.
* `tags` - (Optional) Tags to add to the snapshot. Default is no tags.
* `builder_instance_type` - (Optional) Type of EC2 instance to use as temporary builder instance. Default is `t2.medium`.
* `builder_iam_instance_profile` - (Optional) Name of an IAM Instance Profile to pass to the temporary builder instance. Default is not to pass a profile.
* `builder_instance_tags` - (Optional) Tags to add to the temporary builder instance. Default is no tags.


## Attributes Reference

The following attributes are exported:

* `id` - The AWS EBS snapshot ID (e.g., `snap-abcdefg0`).
