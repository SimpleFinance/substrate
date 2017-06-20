package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"
)

const timeLayout = "2006-01-02T15:04:05Z"

func main() {
	// serve our provider
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() terraform.ResourceProvider {
			return &schema.Provider{
				DataSourcesMap: map[string]*schema.Resource{
					"tarball_file": resourceTarball(),
				},
			}
		},
	})
}

func resourceTarball() *schema.Resource {
	return &schema.Resource{
		Read: resourceTarballRead,
		Schema: map[string]*schema.Schema{
			"directory": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					stat, err := os.Stat(value)
					if err != nil {
						es = append(es, fmt.Errorf("%q: %q", k, err))
					} else if !stat.IsDir() {
						es = append(es, fmt.Errorf("%q %q is not a directory", k, value))
					}
					return
				},
				ForceNew:    true,
				Description: "directory of files to include in the archive",
			},
			"override_timestamp": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					_, err := time.Parse(timeLayout, value)
					if err != nil {
						es = append(es, fmt.Errorf("%q: %s", k, err))
					}
					return
				},
				Description: "override the timestamp of each file in the archive",
			},
			"override_owner": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "override the user ID (uid) of each file in the archive",
			},
			"override_group": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "override the group ID (gid) of each file in the archive",
			},
			"gzip_level": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     9,
				ForceNew:    true,
				Description: "gzip compression level (0-9)",
			},
			"contents_base64": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "tar.gz/tgz file contents encoded with base64",
			},
			"contents_size": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "tar.gz/tgz file size (before base64 encoding)",
			},
			"contents_base64_size": &schema.Schema{
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "tar.gz/tgz file size (after base64 encoding)",
			},
			"sha1": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA1 hash of tar.gz/tgz file (before base64 encoding)",
			},
			"sha256": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA256 hash of tar.gz/tgz file (before base64 encoding)",
			},
		},
	}
}

func buildTarball(d *schema.ResourceData) ([]byte, error) {

	rootDir := d.Get("directory").(string)

	var result bytes.Buffer
	gzipWriter, err := gzip.NewWriterLevel(&result, d.Get("gzip_level").(int))
	if err != nil {
		return nil, err
	}
	tarWriter := tar.NewWriter(gzipWriter)

	// if specified, choose an arbitrary fixed timestamp for all the files in the tarball
	overrideTimestamp := new(time.Time)
	if overrideTimestampParam, isSet := d.GetOk("override_timestamp"); isSet {
		*overrideTimestamp, err = time.Parse(timeLayout, overrideTimestampParam.(string))
	}

	// if specified, set the owner and group of each file in the tarball to root
	overrideOwner := new(int)
	if overrideOwnerParam, isSet := d.GetOk("override_owner"); isSet {
		*overrideOwner = overrideOwnerParam.(int)
	}
	overrideGroup := new(int)
	if overrideGroupParam, isSet := d.GetOk("override_group"); isSet {
		*overrideGroup = overrideGroupParam.(int)
	}

	walkFn := func(path string, info os.FileInfo, err error) error {
		if info.Mode().IsDir() {
			return nil
		}
		relPath := path[len(filepath.Clean(rootDir))+1:]
		if len(relPath) == 0 {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		fileHeader, err := tar.FileInfoHeader(info, relPath)
		if err != nil {
			return err
		}
		if overrideOwner != nil {
			fileHeader.Uid = *overrideOwner
		}
		if overrideGroup != nil {
			fileHeader.Gid = *overrideGroup
		}
		if overrideTimestamp != nil {
			fileHeader.ModTime = *overrideTimestamp
		}
		fileHeader.Name = relPath
		err = tarWriter.WriteHeader(fileHeader)
		if err != nil {
			return err
		}

		_, err = io.Copy(tarWriter, file)
		return err
	}

	err = filepath.Walk(rootDir, walkFn)
	if err != nil {
		return nil, err
	}

	tarWriter.Close()
	gzipWriter.Close()
	return result.Bytes(), nil
}

func sha1Hex(contents []byte) string {
	hashBytes := sha1.Sum(contents)
	return hex.EncodeToString(hashBytes[:])
}

func sha256Hex(contents []byte) string {
	hashBytes := sha256.Sum256(contents)
	return hex.EncodeToString(hashBytes[:])
}

func resourceTarballRead(d *schema.ResourceData, m interface{}) error {
	tarball, err := buildTarball(d)
	if err != nil {
		return fmt.Errorf("error building tarball: %s", err)
	}
	tarballBase64 := base64.StdEncoding.EncodeToString(tarball)

	d.SetId(sha256Hex(tarball))
	d.Set("contents_base64", tarballBase64)
	d.Set("contents_base64_size", len(tarballBase64))
	d.Set("contents_size", len(tarball))
	d.Set("sha1", sha1Hex(tarball))
	d.Set("sha256", sha256Hex(tarball))
	return nil
}
