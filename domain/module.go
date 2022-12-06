package domain

import (
	"reflect"
	"sync"
)

type DomainModuler interface {
	Parse(option interface{})
	Run()
}

type DomainModule struct {
	wg            sync.WaitGroup
	EnabledMethod []string
	*DomainModuleOption
}

type DomainModuleOption struct {
	RootDomain          string
	CachedResults       SubdomainResult
	SubdomainResultChan chan SubdomainResult
}

var RegisteredModule = make(map[string]reflect.Type)

func (dm *DomainModule) Parse(option interface{}) {
	dm.DomainModuleOption = option.(*DomainModuleOption)
}

func (dm *DomainModule) Run() {
	ref := reflect.ValueOf(dm)
	for _, m := range dm.EnabledMethod {
		method := ref.MethodByName("Use" + m)
		if method.Kind() == reflect.Func {
			dm.wg.Add(1)
			go func() {
				defer dm.wg.Done()
				dm.wg.Add(1)
				method.Call(nil)
			}()
		}
	}
	dm.wg.Wait()
}
