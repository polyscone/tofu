package text

import (
	"math/rand"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errors"
	"github.com/polyscone/tofu/internal/pkg/gen"
)

type Desc string

func GenerateDesc() Desc {
	return Desc(optionalDescGenerator.Generate())
}

func NewDesc(desc string) (Desc, error) {
	if _, err := NewOptionalDesc(desc); err != nil {
		return "", errors.Tracef(err)
	}

	if strings.TrimSpace(desc) == "" {
		return "", errors.Tracef("cannot be empty")
	}

	return Desc(desc), nil
}

func (d Desc) String() string {
	return string(d)
}

func (d Desc) Equal(rhs Desc) bool {
	return d == rhs
}

func (d Desc) Generate(rand *rand.Rand) any {
	desc, err := NewDesc(GenerateOptionalDesc().String())
	for {
		if err == nil {
			return desc
		}

		desc, err = NewDesc(GenerateOptionalDesc().String())
	}
}

func (d Desc) Invalidate(rand *rand.Rand, value any) any {
	return Desc(errors.Must(gen.Pattern(`(|[^[[:print:]]\r\n]{1,1000})`)))
}
