package wipe

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func s3HasEnvTag(tags []*s3.Tag, envName string) bool {
	for _, tag := range tags {
		if *tag.Key == "substrate:environment" && *tag.Value == envName {
			return true
		}
	}
	return false
}

type s3Bucket struct {
	svc     *s3.S3
	region  string
	name    string
	objects []string
}

func (r *s3Bucket) String() string {
	return fmt.Sprintf(
		"S3 bucket %s with %d %s in %s",
		r.name,
		len(r.objects),
		map[bool]string{true: "object", false: "objects"}[len(r.objects) == 1],
		r.region)
}

func (r *s3Bucket) Destroy() error {
	for _, object := range r.objects {
		_, err := r.svc.DeleteObject(&s3.DeleteObjectInput{
			Bucket: &r.name,
			Key:    aws.String(object),
		})
		if err != nil {
			return err
		}
	}

	_, err := r.svc.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: &r.name,
	})
	return err
}

func (r *s3Bucket) Priority() int {
	return 100
}

func discoverS3Resources(region string, envName string) []destroyableResource {
	result := []destroyableResource{}
	svc := s3.New(session.New(), &aws.Config{Region: aws.String(region)})

	fmt.Printf("scanning for S3 buckets in %s...\n", region)
	buckets, err := svc.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		panic(err)
	}
	for _, bucket := range buckets.Buckets {

		// we only want to deal with buckets in the chosen region
		location, err := svc.GetBucketLocation(&s3.GetBucketLocationInput{
			Bucket: bucket.Name,
		})
		if err != nil {
			panic(err)
		}
		if *location.LocationConstraint != region {
			continue
		}

		// get the tags on the bucket
		tags, err := svc.GetBucketTagging(&s3.GetBucketTaggingInput{
			Bucket: bucket.Name,
		})
		// handle a special case where a 404/NoSuchTagSet error really just means an empty list
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "NoSuchTagSet" {
				continue
			}
		}
		if err != nil {
			panic(err)
		}
		if !s3HasEnvTag(tags.TagSet, envName) {
			continue
		}

		// panic if the S3 bucket has versioning enabled (deleting these is trickier)
		versioning, err := svc.GetBucketVersioning(&s3.GetBucketVersioningInput{
			Bucket: bucket.Name,
		})
		if err != nil {
			panic(err)
		}
		if versioning.Status != nil {
			panic(fmt.Errorf(
				"need to wipe S3 bucket %s, but it has versioning enabled",
				*bucket.Name,
			))
		}

		// list out all the objects in the bucket, since we'll need to delete them first
		objects := []string{}
		err = svc.ListObjectsPages(
			&s3.ListObjectsInput{
				Bucket: bucket.Name,
			},
			func(page *s3.ListObjectsOutput, lastPage bool) bool {
				for _, object := range page.Contents {
					objects = append(objects, *object.Key)
				}
				return true
			})
		if err != nil {
			panic(err)
		}

		result = append(result, &s3Bucket{
			svc:     svc,
			region:  region,
			name:    *bucket.Name,
			objects: objects,
		})
	}
	return result
}
