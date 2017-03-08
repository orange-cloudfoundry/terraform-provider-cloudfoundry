package resources_test

import (
	. "github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/resources"

	"code.cloudfoundry.org/cli/cf/models"
	"errors"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/cf_client/fake_cf_client"
)

var _ = Describe("Buildpacks", func() {
	var resource *schema.Resource
	var fakeClient *fake_cf_client.FakeCfClient
	var meta interface{}
	var resourceData *schema.ResourceData
	BeforeEach(func() {
		resource = LoadCfResource(CfBuildpackResource{})
		fakeClient = fake_cf_client.NewFakeCfClient()
		meta = fakeClient.GetClient()
		resourceData = resource.Data(&terraform.InstanceState{})
	})
	Describe("Exists", func() {
		It("should return true and assign the buildpack guid to terraform id if the buildpack is found", func() {
			fakeClient.FakeBuildpack().FindByNameReturns(models.Buildpack{
				GUID: "1",
			}, nil)
			err := resourceData.Set("name", "aBuildpack")
			Expect(err).ToNot(HaveOccurred())

			found, err := resource.Exists(resourceData, meta)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(resourceData.Id()).To(BeEquivalentTo("1"))
		})
		It("should return false and no id is reaffected if the buildpack is not found", func() {
			fakeClient.FakeBuildpack().FindByNameReturns(models.Buildpack{}, errors.New("not found"))
			resourceData.Set("name", "aBuildpack")

			found, err := resource.Exists(resourceData, meta)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).To(BeFalse())
			Expect(resourceData.Id()).To(BeEmpty())
		})
	})
	Describe("Delete", func() {
		It("should call deletion if buildpack is not managed by system", func() {
			resourceData.Set("name", "aBuildpack")
			resourceData.Set("position", 1)
			resourceData.Set("path", "http://test.com/fake_buildpack.zip")
			resourceData.Set("locked", false)
			resourceData.Set("enabled", false)
			resourceData.SetId("1")

			err := resource.Delete(resourceData, meta)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeClient.FakeBuildpack().DeleteCallCount()).Should(Equal(1))
		})
		It("should not call deletion if buildpack is managed by system", func() {
			resourceData.Set("name", "aBuildpack")
			resourceData.Set("position", 1)
			resourceData.Set("path", "")
			resourceData.Set("locked", false)
			resourceData.Set("enabled", true)
			resourceData.SetId("1")

			err := resource.Delete(resourceData, meta)
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeClient.FakeBuildpack().DeleteCallCount()).Should(Equal(0))
		})
	})

	Describe("Create", func() {
		var bp models.Buildpack
		guid := "1"
		name := "aBuildpack"
		position := 1
		enabled := true
		locked := false
		BeforeEach(func() {
			bp = models.Buildpack{
				GUID:     guid,
				Enabled:  &enabled,
				Locked:   &locked,
				Name:     name,
				Position: &position,
			}
			fakeClient.FakeBuildpack().FindByNameReturns(bp, nil)
			resourceData.Set("name", name)
			resourceData.Set("position", position)
			resourceData.Set("path", "")
			resourceData.Set("locked", locked)
			resourceData.Set("enabled", enabled)
		})
		Context("when buildpack already exists in Cloud Foundry", func() {

			Context("and buildpack don't need to be updated", func() {
				It("should only set the id for the resource", func() {
					err := resource.Create(resourceData, meta)
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeClient.FakeBuildpack().CreateCallCount()).Should(Equal(0))
					Expect(fakeClient.FakeBuildpack().UpdateCallCount()).Should(Equal(0))
					Expect(fakeClient.FakeBuildpackBits().CreateBuildpackZipFileCallCount()).Should(Equal(0))
					Expect(fakeClient.FakeBuildpackBits().UploadBuildpackCallCount()).Should(Equal(0))
					Expect(resourceData.Id()).To(BeEquivalentTo(guid))
				})
			})
			Context("and buildpack need to be updated", func() {
				It("should only update which is not a buildpack zip file if it didn't change", func() {
					resourceData.Set("locked", !locked)
					resourceData.Set("enabled", !enabled)
					err := resource.Create(resourceData, meta)
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeClient.FakeBuildpack().CreateCallCount()).Should(Equal(0))
					Expect(fakeClient.FakeBuildpack().UpdateCallCount()).Should(Equal(1))
					Expect(fakeClient.FakeBuildpackBits().CreateBuildpackZipFileCallCount()).Should(Equal(0))
					Expect(fakeClient.FakeBuildpackBits().UploadBuildpackCallCount()).Should(Equal(0))
				})
				It("should also update buildpack zip file if it change", func() {
					fakeClient.FakeBuildpack().FindByNameReturns(bp, nil)
					resourceData.Set("path", "http://test.com/fake_buildpack.zip")
					resourceData.Set("locked", !locked)
					resourceData.Set("enabled", !enabled)
					err := resource.Create(resourceData, meta)
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeClient.FakeBuildpack().CreateCallCount()).Should(Equal(0))
					Expect(fakeClient.FakeBuildpack().UpdateCallCount()).Should(Equal(1))
					Expect(fakeClient.FakeBuildpackBits().CreateBuildpackZipFileCallCount()).Should(Equal(1))
					Expect(fakeClient.FakeBuildpackBits().UploadBuildpackCallCount()).Should(Equal(1))
				})
			})
		})
		Context("when buildpack doesn't exists in Cloud Foundry", func() {
			BeforeEach(func() {
				fakeClient.FakeBuildpack().FindByNameReturns(models.Buildpack{}, errors.New("not found"))
				fakeClient.FakeBuildpack().CreateReturns(bp, nil)
			})
			It("should create it in Cloud Foundry", func() {
				err := resource.Create(resourceData, meta)
				Expect(err).ToNot(HaveOccurred())
				resourceData.SetId(guid)

				Expect(fakeClient.FakeBuildpack().CreateCallCount()).Should(Equal(1))
			})
		})
	})

	Describe("Read", func() {
		name := "aBuildpack"
		guid := "1"
		BeforeEach(func() {
			resourceData.Set("name", name)
			resourceData.SetId(guid)
		})
		Context("When the buildpack doesn't exists anymore in Cloud Foundry", func() {
			It("should remove the id to remove reference inside terraform", func() {
				err := resource.Read(resourceData, meta)
				Expect(err).ToNot(HaveOccurred())

				Expect(resourceData.Id()).To(BeEmpty())
			})
		})
		Context("when the buildpack still exists in Cloud Foundry", func() {
			var bp models.Buildpack
			position := 1
			enabled := true
			locked := false
			path := "http://test.com/fake_buildpack.zip"
			BeforeEach(func() {
				bp = models.Buildpack{
					GUID:     guid,
					Enabled:  &enabled,
					Locked:   &locked,
					Name:     name,
					Position: &position,
					Filename: "other_buildpack.zip",
				}
				fakeClient.FakeBuildpack().ListBuildpacksStub = func(cb func(models.Buildpack) bool) error {
					cb(bp)
					return nil
				}
				resourceData.Set("name", name)
				resourceData.Set("position", position)
				resourceData.Set("locked", !locked)
				resourceData.Set("enabled", !enabled)
				resourceData.Set("path", path)
			})
			It("should set resource data with right values if not system managed buildpack", func() {
				err := resource.Read(resourceData, meta)
				Expect(err).ToNot(HaveOccurred())

				Expect(resourceData.Get("locked").(bool)).To(BeFalse())
				Expect(resourceData.Get("enabled").(bool)).To(BeTrue())
				Expect(resourceData.Get("path").(string)).To(Equal("other_buildpack.zip"))
			})
			It("should not set resource data except name if buildpack is a system managed buildpack", func() {
				resourceData.Set("position", 1)
				resourceData.Set("path", "")
				resourceData.Set("locked", false)
				resourceData.Set("enabled", true)
				newName := "new name"
				newEnabledValue := false
				newLockedValue := false
				bp.Enabled = &newEnabledValue
				bp.Locked = &newLockedValue
				bp.Name = newName

				err := resource.Read(resourceData, meta)
				Expect(err).ToNot(HaveOccurred())

				Expect(resourceData.Get("locked").(bool)).To(BeFalse())
				Expect(resourceData.Get("enabled").(bool)).To(BeTrue())
				Expect(resourceData.Get("path").(string)).To(BeEmpty())
				Expect(resourceData.Get("name").(string)).To(Equal(newName))
			})
		})
	})

})
