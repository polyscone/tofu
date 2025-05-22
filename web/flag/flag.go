package flag

import "strings"

type Provider struct {
	env string
}

func NewProvider(env string) *Provider {
	return &Provider{env: strings.ToLower(env)}
}

func (p *Provider) IsDevEnv() bool {
	return p.env == "dev"
}

func (p *Provider) IsTestEnv() bool {
	return p.env == "test"
}

func (p *Provider) IsLiveEnv() bool {
	return !p.IsDevEnv() && !p.IsTestEnv()
}
