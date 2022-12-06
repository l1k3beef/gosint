package domain

import (
	"testing"
	"time"
)

func TestDomainSearcher(t *testing.T) {
	opt := &GosintDomainOption{}
	gd := CreateGosintDomain("hihonor.com", opt)
	gd.Start()

	for {
		time.Sleep(100 * time.Millisecond)
	}
}
