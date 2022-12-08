package model

import "gorm.io/gorm"

// DomainStatus 域名解析状态
type DomainStatus int

const (
	DomainUnkown   = iota // 不清楚状况
	DomainHistory         // 历史可以解析
	DomainInner           // 内网可以解析
	DomainExternal        // 外网可以解析
	DomianSpecial         // 特定DNS解析
)

// DomainResult 子域名扫描结果
type DomainResult struct {
	gorm.Model

	DomainName string       `desc:"域名" gorm:"unique"`
	Status     DomainStatus `desc:"域名有效状态"`
	ResovledIP string       `desc:"解析IP"`
	TaskName   string       `desc:"关联任务" gorm:"not null"`
}

func SaveDomainResults(drs []*DomainResult) (err error) {
	err = db.Save(drs).Error
	return
}
