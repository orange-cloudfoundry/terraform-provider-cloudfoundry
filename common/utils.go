package common

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
	"time"
)

func IsWebURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}
func SchemaSetToStringList(set *schema.Set) []string {
	data := set.List()
	finalList := make([]string, len(data))
	for i, v := range data {
		finalList[i] = v.(string)
	}
	return finalList
}
func SchemaSetToIntList(set *schema.Set) []int {
	data := set.List()
	finalList := make([]int, len(data))
	for i, v := range data {
		finalList[i] = v.(int)
	}
	return finalList
}
func VarToStrPointer(data string) *string {
	if data == "" {
		return nil
	}
	return &data
}
func VarToIntPointer(data int) *int {
	if data == 0 {
		return nil
	}
	return &data
}
func Polling(pollingFunc func() (bool, error), waitTime time.Duration) error {

	for {
		finished, err := pollingFunc()
		if err != nil {
			return err
		}
		if finished {
			return nil
		}
		time.Sleep(waitTime)
	}
	return nil
}
func PollingWithTimeout(pollingFunc func() (bool, error), waitTime time.Duration, timeout time.Duration) error {
	stagingStartTime := time.Now()
	for {
		if time.Since(stagingStartTime) > timeout {
			return fmt.Errorf("Timeout reached")
		}
		finished, err := pollingFunc()
		if err != nil {
			return err
		}
		if finished {
			return nil
		}
		time.Sleep(waitTime)
	}
	return nil
}
