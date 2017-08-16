package resources_test

import (
	. "github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/resources"

	"code.cloudfoundry.org/cli/cf/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Describe("GetMissingSecGroup", func() {
		Context("When slices are equals", func() {
			It("should return an empty slice", func() {
				sliceSource := []models.SecurityGroupFields{
					models.SecurityGroupFields{GUID: "1"},
					models.SecurityGroupFields{GUID: "2"},
				}
				sliceToInspect := []models.SecurityGroupFields{
					models.SecurityGroupFields{GUID: "1"},
					models.SecurityGroupFields{GUID: "2"},
				}
				Expect(GetMissingSecGroup(sliceSource, sliceToInspect)).Should(BeEmpty())
			})
		})
		Context("When slices are differents", func() {
			It("should return an empty slice if the slice to inspect have more elements", func() {
				sliceSource := []models.SecurityGroupFields{
					models.SecurityGroupFields{GUID: "1"},
					models.SecurityGroupFields{GUID: "2"},
				}
				sliceToInspect := []models.SecurityGroupFields{
					models.SecurityGroupFields{GUID: "1"},
					models.SecurityGroupFields{GUID: "2"},
					models.SecurityGroupFields{GUID: "3"},
				}
				Expect(GetMissingSecGroup(sliceSource, sliceToInspect)).Should(BeEmpty())
			})
			It("should return a slice which contains the difference", func() {
				sliceSource := []models.SecurityGroupFields{
					models.SecurityGroupFields{GUID: "1"},
					models.SecurityGroupFields{GUID: "2"},
					models.SecurityGroupFields{GUID: "4"},
					models.SecurityGroupFields{GUID: "5"},
				}
				sliceToInspect := []models.SecurityGroupFields{
					models.SecurityGroupFields{GUID: "1"},
					models.SecurityGroupFields{GUID: "3"},
					models.SecurityGroupFields{GUID: "4"},
				}
				expectedSlice := []models.SecurityGroupFields{
					models.SecurityGroupFields{GUID: "2"},
					models.SecurityGroupFields{GUID: "5"},
				}
				Expect(GetMissingSecGroup(sliceSource, sliceToInspect)).Should(Equal(expectedSlice))
			})
		})
	})
})
