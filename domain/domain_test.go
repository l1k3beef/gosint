package domain

import (
	"testing"
	"time"
)

func TestDomainSearcher(t *testing.T) {
	opt := &GosintDomainOption{
		EnabledModuleMethod: make(map[string][]string),
		DomainSearcherOption: &DomainSearcherOption{
			Proxy: "http://localhost:8080",
		},
	}

	opt.EnabledModuleMethod["DomainSearcher"] = []string{"Baidu"}

	gd := CreateGosintDomain("hihonor.com", opt)
	gd.Start()

	for {
		time.Sleep(100 * time.Millisecond)
	}
}

func TestDomainProvider(t *testing.T) {
	opt := &GosintDomainOption{
		EnabledModuleMethod: make(map[string][]string),
		DomainProviderOption: &DomainProviderOption{
			Proxy: "http://localhost:8080",
		},
	}
	opt.EnabledModuleMethod["DomainProvider"] = []string{"CertSpotter"}

	gd := CreateGosintDomain("hihonor.com", opt)
	gd.Start()

	for {
		time.Sleep(100 * time.Millisecond)
	}
}
