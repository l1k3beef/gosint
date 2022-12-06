package domain

import (
	"reflect"
	"time"
)

type GosintDomain struct {
	GosintDomainArgument
	RunningModuleCount int
	*GosintDomainOption
}

// GosintDomainArgument 多线程通信所需要的变量
//
// CachedResults非线程安全, 仅提供给DomainGuesser只读权限
type GosintDomainArgument struct {
	RootDomain           string
	CachedResults        SubdomainResult
	SubdomainResultChan  chan SubdomainResult
	ModuleFinishedSignal chan struct{}
}

type GosintDomainOption struct {
	Standardized     bool     `desc:"是否对扫描结果标准化处理"`
	LimitCachedSize  int      `desc:"限制缓存结果的数量"`
	LimitChannelSize int      `desc:"限制子模块交互通道的数量"`
	EnabledModule    []string `desc:"开启的子模块列表"`

	*DomainSearcherOption
	*DomainGuesserOption
}

type StandardSubdomainResult struct {
	DomainName string
	ResovledIP string
}

type SubdomainResult map[string]struct{}

// CreateGosintDomain 用来创建GosintDomain实例的方法, 推荐使用
func CreateGosintDomain(root string, opt *GosintDomainOption) (gd *GosintDomain) {
	gd.registerModule()
	gd = &GosintDomain{}
	gd.GosintDomainOption = opt
	gd.RootDomain = root

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
		return false
	}

	return true
}

// Start 开始子域名搜集
func (gd *GosintDomain) Start() {
	if !gd.PermissiveOptionCheck() {
		return
	}

	for _, moduleName := range gd.EnabledModule {
		ref := reflect.ValueOf(gd.GosintDomainOption)
		option := ref.FieldByName(moduleName + "Option").Interface()
		m := CreateDomainModule(moduleName)
		go func() {
			m.Init(option)
			m.Run()
		}()
	}

	go func() {
		defer func() {
			gd.Persistence()
		}()

		select {
		case sr := <-gd.SubdomainResultChan:
			gd.Deduplicate(sr)
		case <-gd.ModuleFinishedSignal:
			if gd.RunningModuleCount--; gd.RunningModuleCount <= 0 {
				// 保证退出之前SubdomainResultChan没有内容
				return
			}
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}()
}
