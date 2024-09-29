package guard

type System struct {
	*Passport
}

func (s System) CanViewConfig() bool {
	return s.can(viewConfig)
}

func (s System) CanUpdateConfig() bool {
	return s.can(updateConfig)
}

func (s System) CanViewMetrics() bool {
	return s.can(viewMetrics)
}
