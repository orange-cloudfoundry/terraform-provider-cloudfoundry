package common_test

import (
	. "github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/common"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Describe("IsWebURL", func() {
		Context("When the web url is valid", func() {
			It("should return true if it's an http url", func() {
				Expect(IsWebURL("http://test.com")).Should(BeTrue())
			})
			It("should return true if it's an https url", func() {
				Expect(IsWebURL("https://test.com")).Should(BeTrue())
			})
		})
		Context("When the web url is invalid", func() {
			It("should return false if it's not an http or https url", func() {
				Expect(IsWebURL("fprot://test.com")).Should(BeFalse())
			})
		})
	})
})
