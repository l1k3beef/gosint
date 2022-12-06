package domain

import (
	"reflect"
	"sync"
	"time"
)

// GosintDomain 公开信息搜集子域名
type GosintDomain struct {
	wg                 sync.WaitGroup
	RunningModuleCount int
	StoppedSignal      chan struct{}

	*GosintDomainOption
}

// GosintDomainOption 用户配置项
type GosintDomainOption struct {
	Standardized     bool     `desc:"是否对扫描结果标准化处理"`
	LimitCachedSize  int      `desc:"限制缓存结果的数量"`
	LimitChannelSize int      `desc:"限制子模块交互通道的数量"`
	EnabledModule    []string `desc:"开启的子模块列表"`

	*DomainModuleOption
	*DomainSearcherOption
	*DomainGuesserOption
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

	gd.initSubdomainResultChan()
	gd.CachedResults = make(SubdomainResult)
	return
}

func (gd *GosintDomain) registerModule() {
	RegisteredModule["DomainSearcher"] = reflect.TypeOf(DomainSearcher{})
}

func (gd *GosintDomain) initSubdomainResultChan() {
	if gd.LimitChannelSize < 2 {
		gd.LimitChannelSize = 2
	}
	gd.SubdomainResultChan = make(chan SubdomainResult, gd.LimitChannelSize)
}

// Deduplicate 对内存中缓存的结果进行去重
func (gd *GosintDomain) Deduplicate(sr SubdomainResult) {
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
	if len(gd.EnabledModule) == 0 {
		Log.Warn("最少启用一个子域名探测模块")
		return false
	}

	for _, moduleName := range gd.EnabledModule {
		option := gd.GetDomainModuleOption(moduleName)
		if option == nil {
			Log.Warnf("找不到模块:%v对应的配置项", moduleName)
			return false
		}
	}

	Log.Info("用户配置项检查通过")
	return true
}

// CreateDomainModule 创建子域名搜集模块的工厂方法
func (gd *GosintDomain) CreateDomainModule(moduleName string) (m DomainModuler) {
	m = reflect.New(RegisteredModule[moduleName]).Interface().(DomainModuler)
	option := gd.GetDomainModuleOption("DomainModule")
	m.Parse(option)
	return
}

// GetDomainModuleOption 动态获取参数中的用户配置项
func (gd *GosintDomain) GetDomainModuleOption(moduleName string) (option interface{}) {
	ref := reflect.ValueOf(gd.GosintDomainOption)
	option = ref.FieldByName(moduleName + "Option").Interface()
	return
}

// Start 开始子域名搜集
func (gd *GosintDomain) Start() {
	Log.Info("子域名搜集开始")
	if !gd.PermissiveOptionCheck() {
		return
	}

	for _, moduleName := range gd.EnabledModule {

		option := gd.GetDomainModuleOption(moduleName)
		m := gd.CreateDomainModule(moduleName)
		go func() {
			defer gd.wg.Done()
			gd.wg.Add(1)
			m.Parse(option)
			m.Run()
		}()
	}

	go func() {
		defer func() {
			gd.Persistence()
			Log.Info("子域名搜集结束")
		}()

		select {
		case sr := <-gd.SubdomainResultChan:
			gd.Deduplicate(sr)
		case <-gd.StoppedSignal:
			return
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}()

	gd.wg.Wait()
	gd.StoppedSignal <- struct{}{}
}
