package domain

import (
	"reflect"
	"strings"
	"sync"
	"time"
)

// GosintDomain 公开信息搜集子域名
type GosintDomain struct {
	wg                 sync.WaitGroup `desc:"同步多个子模块结束"`
	RunningModuleCount int
	StoppedSignal      chan struct{}

	*GosintDomainOption
}

// GosintDomainOption 用户配置项
type GosintDomainOption struct {
	Standardized        bool                `desc:"是否对扫描结果标准化处理"`
	LimitCachedSize     int                 `desc:"限制缓存结果的数量"`
	LimitChannelSize    int                 `desc:"限制子模块交互通道的数量"`
	EnabledModuleMethod map[string][]string `desc:"开启的模块和对应的方法"`

	*DomainModuleOption
	*DomainSearcherOption
	*DomainGuesserOption
	*DomainProviderOption
}

// StandardSubdomainResult 标准化后的子域名搜集结果
type StandardSubdomainResult struct {
	DomainName string
	ResovledIP string
}

// SubdomainResult 简化的子域名搜集结果
type SubdomainResult map[string]struct{}

// CreateGosintDomain 用来创建GosintDomain实例的方法, 推荐使用
func CreateGosintDomain(root string, opt *GosintDomainOption) (gd *GosintDomain) {
	gd.registerModule()
	gd = &GosintDomain{}
	gd.GosintDomainOption = opt

	// 初始化DomainModuleOption的内容
	if gd.DomainModuleOption == nil {
		gd.DomainModuleOption = &DomainModuleOption{}
	}
	if root != "" {
		gd.RootDomain = root
	}
	gd.initSubdomainResultChan()
	gd.CachedResults = make(SubdomainResult)

	gd.StoppedSignal = make(chan struct{}, 1)
	return
}

func (gd *GosintDomain) registerModule() {
	RegisteredModule["DomainSearcher"] = reflect.TypeOf(DomainSearcher{})
	RegisteredModule["DomainProvider"] = reflect.TypeOf(DomainProvider{})
	RegisteredModule["DomainGuesser"] = reflect.TypeOf(DomainGuesser{})
}

func (gd *GosintDomain) initSubdomainResultChan() {
	if gd.LimitChannelSize < 2 {
		gd.LimitChannelSize = 2
	}
	gd.SubdomainResultChan = make(chan SubdomainResult, gd.LimitChannelSize)
}

// Process 对结果去重保存到内存中
func (gd *GosintDomain) Process(sr SubdomainResult) {
	for r := range sr {
		if _, ok := gd.CachedResults[r]; ok {
			continue
		}
		gd.CachedResults[r] = struct{}{}
	}
	if gd.LimitCachedSize > 0 && gd.LimitCachedSize <= len(gd.CachedResults) {
		gd.Persistence()
	}
}

// Standardize 标准化过程中补充子域名信息
func (gd *GosintDomain) Standardize(subdomain string) (ssr StandardSubdomainResult) {
	return
}

// Persistence 持久化保存结果
func (gd *GosintDomain) Persistence() {
}

// PermissiveOptionCheck 对用户配置项进行宽松的检查
func (gd *GosintDomain) PermissiveOptionCheck() (ok bool) {
	if len(gd.EnabledModuleMethod) == 0 {
		Log.Warn("最少启用一个子域名探测模块")
		return false
	}

	for moduleName, moduleMethod := range gd.EnabledModuleMethod {
		if len(moduleMethod) == 0 {
			Log.Warnf("将为模块%v启用所有方法", moduleName)
		}
		option := gd.GetDomainModuleOption(moduleName)
		if option == nil {
			Log.Warnf("找不到模块%v对应的配置项", moduleName)
			return false
		}
	}

	Log.Info("用户配置项检查通过")
	return true
}

// CreateDomainModule 创建子域名搜集模块的工厂方法, 子模块共用同一个DomainModuleOption
func (gd *GosintDomain) CreateDomainModule(moduleName string, moduleMethod []string) (m DomainModuler) {
	defer func() {
		if r := recover(); r != nil {
			Log.Warnf("创建模块%v失败, %v", moduleName, r)
			m = nil
		}
	}()
	m = reflect.New(RegisteredModule[moduleName]).Interface().(DomainModuler)
	dm := &DomainModule{
		DomainModuleOption: gd.GetDomainModuleOption("DomainModule").(*DomainModuleOption),
	}
	// 默认使用所有UseXXX的功能, 不包含继承的方法
	if len(moduleMethod) == 0 {
		ref := reflect.TypeOf(m)
		for i := 0; i < ref.NumMethod(); i++ {
			name := ref.Method(i).Name
			if strings.HasPrefix(name, "Use") {
				moduleMethod = append(moduleMethod, name[3:])
			}
		}
	}
	dm.EnabledMethod = moduleMethod
	reflect.ValueOf(m).Elem().FieldByName("DomainModule").Set(reflect.ValueOf(dm))
	return
}

// GetDomainModuleOption 动态获取参数中的用户配置项
func (gd *GosintDomain) GetDomainModuleOption(moduleName string) (option interface{}) {
	ref := reflect.ValueOf(gd.GosintDomainOption).Elem()
	option = ref.FieldByName(moduleName + "Option").Interface()
	return
}

// Start 开始子域名搜集
func (gd *GosintDomain) Start() {
	Log.Info("子域名搜集开始")
	if !gd.PermissiveOptionCheck() {
		return
	}

	for moduleName, moduleMethod := range gd.EnabledModuleMethod {

		option := gd.GetDomainModuleOption(moduleName)
		m := gd.CreateDomainModule(moduleName, moduleMethod)
		if m != nil {
			gd.wg.Add(1)
			go func() {
				defer gd.wg.Done()
				m.Parse(option)
				m.Run(m)
			}()
		}
	}

	go func() {
		defer func() {
			gd.Persistence()
			Log.Info("子域名搜集结束")
		}()
		for {
			select {
			case sr := <-gd.SubdomainResultChan:
				gd.Process(sr)
			case <-gd.StoppedSignal:
				return
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	gd.wg.Wait()
	gd.StoppedSignal <- struct{}{}
}
