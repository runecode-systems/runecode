//go:build !runecode_devseed

package brokerapi

import "fmt"

func (s *Service) seedDevManualInstanceControlContext() error {
	return fmt.Errorf("dev manual policy-context seeding unavailable in this build")
}
