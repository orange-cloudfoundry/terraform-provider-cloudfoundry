// Code generated by counterfeiter. DO NOT EDIT.
package bitsmanagerfakes

import (
	"sync"

	"github.com/orange-cloudfoundry/terraform-provider-cloudfoundry/bitsmanager"
)

type FakeBitsManager struct {
	UploadStub        func(appGuid string, path string) error
	uploadMutex       sync.RWMutex
	uploadArgsForCall []struct {
		appGuid string
		path    string
	}
	uploadReturns struct {
		result1 error
	}
	uploadReturnsOnCall map[int]struct {
		result1 error
	}
	IsDiffStub        func(path string, currentSha1 string) (isDiff bool, sha1 string, err error)
	isDiffMutex       sync.RWMutex
	isDiffArgsForCall []struct {
		path        string
		currentSha1 string
	}
	isDiffReturns struct {
		result1 bool
		result2 string
		result3 error
	}
	isDiffReturnsOnCall map[int]struct {
		result1 bool
		result2 string
		result3 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeBitsManager) Upload(appGuid string, path string) error {
	fake.uploadMutex.Lock()
	ret, specificReturn := fake.uploadReturnsOnCall[len(fake.uploadArgsForCall)]
	fake.uploadArgsForCall = append(fake.uploadArgsForCall, struct {
		appGuid string
		path    string
	}{appGuid, path})
	fake.recordInvocation("Upload", []interface{}{appGuid, path})
	fake.uploadMutex.Unlock()
	if fake.UploadStub != nil {
		return fake.UploadStub(appGuid, path)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.uploadReturns.result1
}

func (fake *FakeBitsManager) UploadCallCount() int {
	fake.uploadMutex.RLock()
	defer fake.uploadMutex.RUnlock()
	return len(fake.uploadArgsForCall)
}

func (fake *FakeBitsManager) UploadArgsForCall(i int) (string, string) {
	fake.uploadMutex.RLock()
	defer fake.uploadMutex.RUnlock()
	return fake.uploadArgsForCall[i].appGuid, fake.uploadArgsForCall[i].path
}

func (fake *FakeBitsManager) UploadReturns(result1 error) {
	fake.UploadStub = nil
	fake.uploadReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeBitsManager) UploadReturnsOnCall(i int, result1 error) {
	fake.UploadStub = nil
	if fake.uploadReturnsOnCall == nil {
		fake.uploadReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.uploadReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeBitsManager) IsDiff(path string, currentSha1 string) (isDiff bool, sha1 string, err error) {
	fake.isDiffMutex.Lock()
	ret, specificReturn := fake.isDiffReturnsOnCall[len(fake.isDiffArgsForCall)]
	fake.isDiffArgsForCall = append(fake.isDiffArgsForCall, struct {
		path        string
		currentSha1 string
	}{path, currentSha1})
	fake.recordInvocation("IsDiff", []interface{}{path, currentSha1})
	fake.isDiffMutex.Unlock()
	if fake.IsDiffStub != nil {
		return fake.IsDiffStub(path, currentSha1)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	return fake.isDiffReturns.result1, fake.isDiffReturns.result2, fake.isDiffReturns.result3
}

func (fake *FakeBitsManager) IsDiffCallCount() int {
	fake.isDiffMutex.RLock()
	defer fake.isDiffMutex.RUnlock()
	return len(fake.isDiffArgsForCall)
}

func (fake *FakeBitsManager) IsDiffArgsForCall(i int) (string, string) {
	fake.isDiffMutex.RLock()
	defer fake.isDiffMutex.RUnlock()
	return fake.isDiffArgsForCall[i].path, fake.isDiffArgsForCall[i].currentSha1
}

func (fake *FakeBitsManager) IsDiffReturns(result1 bool, result2 string, result3 error) {
	fake.IsDiffStub = nil
	fake.isDiffReturns = struct {
		result1 bool
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeBitsManager) IsDiffReturnsOnCall(i int, result1 bool, result2 string, result3 error) {
	fake.IsDiffStub = nil
	if fake.isDiffReturnsOnCall == nil {
		fake.isDiffReturnsOnCall = make(map[int]struct {
			result1 bool
			result2 string
			result3 error
		})
	}
	fake.isDiffReturnsOnCall[i] = struct {
		result1 bool
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeBitsManager) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.uploadMutex.RLock()
	defer fake.uploadMutex.RUnlock()
	fake.isDiffMutex.RLock()
	defer fake.isDiffMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeBitsManager) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ bitsmanager.BitsManager = new(FakeBitsManager)