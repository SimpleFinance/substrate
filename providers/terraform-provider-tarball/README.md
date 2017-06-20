# tarball_file

Provides a data source for creating `.tar.gz`/`.tgz` archives from local files (similar to the builtin `template_file` data source).

## Example Usage

```hcl
# create an archive of the ./foo directory (relative to the current module)
data "tarball_file" "mytarball" {
    directory = "${path.module}/foo"
    gzip_level = 9
    override_timestamp = "2015-12-31T18:34:13Z"
    override_owner = 0
    override_group = 0
}

# upload the tarball to s3://your_bucket_name/$(SHA256).tgz
resource "aws_s3_bucket_object" "mys3tarball" {
    bucket = "your_bucket_name"
    key = "${data.tarball_file.mytarball.sha256}.tgz"
    content = "${base64decode(data.tarball_file.mytarball.contents_base64)}"
}
```

## Argument Reference

The following arguments are supported:

* `directory` - (Required) The path to the directory of files to include in the archive.
* `override_timestamp` - (Optional) If set, override the modification timestamp of each archived file. The value should be in "YYYY-MM-DDThh:mm:ssZ" format in UTC/GMT only (for example, 2014-06-01T00:00:00Z ). Using this parameter prevents needless updates in the case where you only care about file contents. The default is to preserve the modification timestamps of the source files.
* `override_owner` / `override_group` - (Optional) If set, override the owner (UID) and/or group (GID) of each archived file. The value should be numeric. Use `0` to change the owner to root. The default is to keep the original owner and group data from the source files.
* `gzip_level` - (Optional) Set the (numeric) GZip compression level. Value should be 0 (no compression) through 9 (maxiumum compression). The default is 9.


## Attributes Reference

The following attributes are exported:

* `id` - Unique ID of the archive contents (changes whenever the file is modified).
* `contents_base64` - Base64 encoded, compressed archive file (use the builtin `base64decode` function to decode if needed).
* `contents_size` - Size of the archive file (before base64 encoding), in bytes.
* `contents_base64_size` - Size of the `contents_base64` value (after base64 encoding), in bytes.
* `sha1` / `sha256` - SHA1 / SHA256 hashes of the compressed archive file (before base64 encoding).
