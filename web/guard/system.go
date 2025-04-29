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

func (s System) CanBackup() bool {
	return s.can(backup)
}

func (s System) CanRestore() bool {
	return s.can(restore)
}

func (s System) CanViewMetrics() bool {
	return s.can(viewMetrics)
}
