package domain

type Claim string

func (c Claim) String() string {
	return string(c)
}
