package resources_test

import (
	. "github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/resources"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Quotas", func() {
	resource := NewCfQuotaResource()
	Context("Read", func() {
		It("empty", func() {
			resource.Schema()
			Expect(true).To(BeTrue())
		})
	})
})
