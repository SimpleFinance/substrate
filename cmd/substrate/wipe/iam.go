package wipe

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

type iamInstanceProfile struct {
	svc   *iam.IAM
	name  string
	id    string
	roles []string
}

func (r *iamInstanceProfile) String() string {
	return fmt.Sprintf(
		"IAM instance profile %s with %d %s (%s)",
		r.name,
		len(r.roles),
		map[bool]string{true: "role", false: "roles"}[len(r.roles) == 1],
		r.id)
}

func (r *iamInstanceProfile) Destroy() error {
	for _, role := range r.roles {
		_, err := r.svc.RemoveRoleFromInstanceProfile(&iam.RemoveRoleFromInstanceProfileInput{
			InstanceProfileName: &r.name,
			RoleName:            &role,
		})
		if err != nil {
			return err
		}
	}

	_, err := r.svc.DeleteInstanceProfile(&iam.DeleteInstanceProfileInput{
		InstanceProfileName: &r.name,
	})
	return err
}

func (r *iamInstanceProfile) Priority() int {
	return 450
}

type iamRole struct {
	svc      *iam.IAM
	name     string
	id       string
	policies []string
}

func (r *iamRole) String() string {
	return fmt.Sprintf(
		"IAM role %s with %d %s (%s)",
		r.name,
		len(r.policies),
		map[bool]string{true: "policy", false: "policies"}[len(r.policies) == 1],
		r.id)
}

func (r *iamRole) Destroy() error {
	for _, policy := range r.policies {
		_, err := r.svc.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
			PolicyName: &policy,
			RoleName:   &r.name,
		})
		if err != nil {
			return err
		}
	}
	_, err := r.svc.DeleteRole(&iam.DeleteRoleInput{
		RoleName: &r.name,
	})
	return err
}

func (r *iamRole) Priority() int {
	return 500
}

func discoverIAMResources(envName string) []destroyableResource {
	result := []destroyableResource{}
	svc := iam.New(session.New())

	fmt.Printf("scanning for IAM instance profiles...\n")
	err := svc.ListInstanceProfilesPages(&iam.ListInstanceProfilesInput{},
		func(page *iam.ListInstanceProfilesOutput, lastPage bool) bool {
			for _, profile := range page.InstanceProfiles {
				if strings.Contains(*profile.InstanceProfileName, envName) {
					p := &iamInstanceProfile{
						svc:  svc,
						name: *profile.InstanceProfileName,
						id:   *profile.InstanceProfileId,
					}
					for _, role := range profile.Roles {
						p.roles = append(p.roles, *role.RoleName)
					}
					result = append(result, p)
				}
			}
			return true
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("scanning for IAM roles...\n")
	err = svc.ListRolesPages(&iam.ListRolesInput{},
		func(page *iam.ListRolesOutput, lastPage bool) bool {
			for _, role := range page.Roles {
				if strings.Contains(*role.RoleName, envName) {
					r := &iamRole{
						svc:  svc,
						name: *role.RoleName,
						id:   *role.RoleId,
					}
					err = svc.ListRolePoliciesPages(
						&iam.ListRolePoliciesInput{
							RoleName: role.RoleName,
						},
						func(page *iam.ListRolePoliciesOutput, lastPage bool) bool {
							for _, policyName := range page.PolicyNames {
								r.policies = append(r.policies, *policyName)
							}
							return true
						})
					if err != nil {
						panic(err)
					}
					result = append(result, r)
				}
			}
			return true
		},
	)
	if err != nil {
		panic(err)
	}

	return result
}
