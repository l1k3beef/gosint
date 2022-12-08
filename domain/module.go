package domain

import (
	"reflect"
	"sync"
)

type DomainModuler interface {
	Parse(option interface{})
	Run(this interface{})
}

type DomainModule struct {
	wg            sync.WaitGroup `desc:"同步多个方法结束"`
	EnabledMethod []string
	*DomainModuleOption
}

// DomainModuleOption 子模块交互信息
type DomainModuleOption struct {
	RootDomain          string
	CachedResults       SimpleDomainResult `desc:"子模块对缓存结果只有只读权限"`
	SubdomainResultChan chan SimpleDomainResult
}

var RegisteredModule = make(map[string]reflect.Type)

func (dm *DomainModule) Parse(option interface{}) {
	dm.DomainModuleOption = option.(*DomainModuleOption)
}

func (dm *DomainModule) Run(this interface{}) {
	ref := reflect.ValueOf(this)
	for _, m := range dm.EnabledMethod {
		method := ref.MethodByName("Use" + m)
		if method.Kind() == reflect.Func {
			dm.wg.Add(1)
			go func() {
				defer dm.wg.Done()
				method.Call(nil)
			}()
		} else {
			Log.Warnf("%v模块中找不到%v方法", ref.Type().Name(), "Use"+m)
		}
	}
	dm.wg.Wait()
}

func isValid() {

}
