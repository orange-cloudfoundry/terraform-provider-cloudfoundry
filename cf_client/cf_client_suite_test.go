package cf_client_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCfClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CfClient Suite")
}
