package domain

type DomainGuesser struct {
	*DomainModule
}

type DomainGuesserOption struct {
	Enabled []string
}

// UseDict 使用高频率的字典进行子域名猜测
func (dg *DomainGuesser) UseDict() {
}

// UseCharacterTraversal 逐字符遍历
func (dg *DomainGuesser) UseCharacterTraversal() {

}

// UsePermutations 排列组合
func (dg *DomainGuesser) UsePermutations() {

}

// UseJoinRandom 添加随机词汇
func (dg *DomainGuesser) UseJoinRandom() {

}

// UseOpenAI 使用智能算法进行子域名猜测
func (dg *DomainGuesser) UseOpenAI() {

}

func (dg *DomainGuesser) doGuess() {

}

func (dg *DomainGuesser) Run() {
}
