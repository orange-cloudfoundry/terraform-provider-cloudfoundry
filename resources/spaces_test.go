package resources_test

import (
	. "github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/resources"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Spaces", func() {
	resource := CfSpaceResource{}
	Context("Read", func() {
		It("empty", func() {
			resource.Schema()
			Expect(true).To(BeTrue())
		})
	})
})
