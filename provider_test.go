package main_test

import (
	. "github.com/orange-cloudfoundry/terraform-provider-cloudfoundry"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/hashicorp/terraform/helper/schema"
)

var _ = Describe("Provider", func() {
	It("should validate", func() {
		Expect(Provider().(*schema.Provider).InternalValidate()).To(Succeed())
	})
})
