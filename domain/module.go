package domain

import (
	"reflect"
	"sync"
)

type DomainModuler interface {
	Init(interface{})
	Run()
}

type DomainModule struct {
	wg            sync.WaitGroup
	EnabledMethod []string
}

var RegisteredModule = make(map[string]reflect.Type)

func CreateDomainModule(moduleName string) (m DomainModuler) {
	m = reflect.New(RegisteredModule[moduleName]).Interface().(DomainModuler)
	return
}

func (dm *DomainModule) Run() {
	ref := reflect.ValueOf(dm)
	for _, m := range dm.EnabledMethod {
		method := ref.MethodByName("Use" + m)
		if method.Kind() == reflect.Func {
			dm.wg.Add(1)
			go func() {
				defer dm.wg.Done()
				method.Call(nil)

			}()
		}
	}
	dm.wg.Wait()
}
