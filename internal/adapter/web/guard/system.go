package guard

type System struct {
	*Passport
}

func (s System) CanViewConfig() bool {
	return s.requireConfigSetup || s.can(viewConfig)
}

func (s System) CanUpdateConfig() bool {
	return s.requireConfigSetup || s.can(updateConfig)
}

func (s System) CanViewMetrics() bool {
	return s.isSuper
}
