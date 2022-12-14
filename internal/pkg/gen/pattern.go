package gen

import (
	"fmt"
	"math/rand"
	"regexp/syntax"
	"strings"
	"sync"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

const codepointsRangeEnd = 0x10ffff

var generators = struct {
	mu   sync.Mutex
	data map[string]*PatternGenerator
}{data: make(map[string]*PatternGenerator)}

// Pattern is a convenience function that creates a new generator based on the
// given regular expression pattern, and then uses that generator to create and
// return a new random string.
// Generators are cached in memory so multiple calls for the same pattern will
// use the same generator.
// This function is safe for concurrent use.
func Pattern(pattern string) (string, error) {
	generators.mu.Lock()
	defer generators.mu.Unlock()

	if pg := generators.data[pattern]; pg != nil {
		return pg.Generate(), nil
	}

	pg, err := NewPatternGenerator(pattern)
	if err != nil {
		return "", errors.Tracef(err)
	}

	generators.data[pattern] = pg

	return pg.Generate(), nil
}

// PatternGenerator implements a string generator that generates strings based
// on a regular expression.
// The regular expression it is created with is cached internally, so for each
// new pattern a new generator is required.
// By default a pattern generator is safe for concurrent use, unless the source
// of randomness is set to a source that is not safe for concurrent use.
type PatternGenerator struct {
	re   *syntax.Regexp
	rand *rand.Rand
}

// NewPatternGenerator create a new pattern generator based on the given
// regular expression pattern.
func NewPatternGenerator(pattern string) (*PatternGenerator, error) {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil, errors.Tracef(err)
	}

	pg := &PatternGenerator{
		re:   re,
		rand: defaultRand,
	}

	return pg, nil
}

func (pg *PatternGenerator) generate(re *syntax.Regexp, sb *strings.Builder, limit int) {
	switch op := re.Op; op {
	case syntax.OpBeginLine,
		syntax.OpBeginText,
		syntax.OpEmptyMatch,
		syntax.OpEndLine,
		syntax.OpEndText,
		syntax.OpNoMatch,
		syntax.OpNoWordBoundary,
		syntax.OpWordBoundary:
		// All of these ops don't do anything useful for string generation, so
		// we handle them in order to not panic but leave them as no-ops

	case syntax.OpAlternate:
		i := pg.rand.Intn(len(re.Sub))

		pg.generate(re.Sub[i], sb, limit)

	case syntax.OpAnyChar:
		sb.WriteRune(rune(pg.rand.Intn(codepointsRangeEnd)))

	case syntax.OpAnyCharNotNL:
		r := rune(pg.rand.Intn(codepointsRangeEnd))
		for r == '\n' {
			r = rune(pg.rand.Intn(codepointsRangeEnd))
		}

		sb.WriteRune(r)

	case syntax.OpCapture:
		pg.generate(re.Sub0[0], sb, limit)

	case syntax.OpCharClass:
		// It's possible to write a regular expression that leaves the range of
		// applicable runes empty, for example [^\x00-\x{10FFFF}], so we need
		// to check for that here to prevent a panic
		if len(re.Rune) > 0 {
			// We can't just choose a rune range pair at random and then choose
			// a rune from that range because each run range pair may describe
			// vastly different quantities of code points
			//
			// To randomly choose a rune in all of the available ranges as
			// fairly as possible we can map each range into the half-open range
			// [0, n), where n is the total number of possible code points
			// available in the sum of all ranges
			//
			// This mapping is possible because character class rune range pairs
			// are sorted from low to high, merged, deduplicated, etc. by the Go
			// standard library when parsing a regular expression
			//
			// For example, if we have two rune ranges, [40, 50], and [70, 80],
			// then we can map the m to a half-open range as follows:
			//
			//   Mapping: 0 ------- 9 10 -------- 19 20
			//   Ranges:  [40     50] [70        80] --
			//
			// That is, [40, 50] maps to [0, 10), and [70, 80] maps to [10, 20)
			// We can then choose a random number in the mapping and figure out
			// which rune that applies to
			var mapping int
			for i := 0; i < len(re.Rune); i += 2 {
				start, end := int(re.Rune[i]), int(re.Rune[i+1])

				mapping += 1 + end - start
			}

			// Here we choose a random number within the mapping that represents
			// the rune we will add to the string
			//
			// To figure out which rune range pair the number n falls into we
			// can think of n as a number representing a level in a bucket
			//
			// For each rune range pair we get the distance between them
			// If the "level" in the "bucket" is lower than the distance then
			// we've found the range that n maps to
			// If it's greater than or equal to the distance then we subtract
			// the distance from n and check the next range in the same fashion
			//
			// This works because the rune range pairs are sorted from
			// lowest to highest
			n := pg.rand.Intn(mapping)
			for i := 0; i < len(re.Rune); i += 2 {
				start, end := int(re.Rune[i]), int(re.Rune[i+1])
				distance := 1 + end - start

				if n < distance {
					sb.WriteRune(rune(start + n))

					break
				}

				n -= distance
			}
		}

	case syntax.OpConcat:
		for _, sub := range re.Sub {
			pg.generate(sub, sb, limit)
		}

	case syntax.OpLiteral:
		for _, r := range re.Rune {
			sb.WriteRune(r)
		}

	case syntax.OpRepeat:
		max := re.Max
		if max < 0 {
			if limit < re.Min {
				max = re.Min
			} else {
				max = limit
			}
		}
		if re.Min != max {
			max = re.Min + pg.rand.Intn(1+max-re.Min)
		}

		for i := 0; i < max; i++ {
			for _, sub := range re.Sub {
				pg.generate(sub, sb, limit)
			}
		}

	case syntax.OpStar, syntax.OpPlus:
		var min int
		if op == syntax.OpPlus {
			min = 1
		}

		// Since star and plus match any number of characters we choose a random
		// number of runes to generate up to an internal limit, using
		// min as a base, where min for star is 0 and for plus is 1
		n := min + pg.rand.Intn(1+limit)
		for i := 0; i < n; i++ {
			pg.generate(re.Sub0[0], sb, limit)
		}

	case syntax.OpQuest:
		// Quest is the ? optional operator, so we use the first bit of a random
		// int to decide whether or not to include it or not
		if pg.rand.Int()&1 == 0 {
			pg.generate(re.Sub0[0], sb, limit)
		}

	default:
		panic(fmt.Sprintf("unhandled op %v", op))
	}
}

// Generate will create a new random string from the regular expression pattern
// provided at the time of the generator's creation with a limit of 10.
//
// For details see the documentation for GenerateLimit.
func (pg *PatternGenerator) Generate() string {
	return pg.GenerateLimit(10)
}

// Generate will create a new random string from the regular expression pattern
// provided at the time of the generator's creation.
//
// The quantifiers * and + will generate at most n runes, where n is the given
// limit number.
// In a range of characters such as {5,} where the upper limit is unbounded it
// will be set to the given limit, unless the lower bound is higher, in which
// case the lower and upper bound will be the same.
// The limit must be at least 1.
//
// For some regular expressions a string that matches exactly cannot
// be generated.
// In these cases the closest approximation is generated instead.
// For example, "a^a" cannot be matched, but will produce the string "aa".
func (pg *PatternGenerator) GenerateLimit(limit int) string {
	if limit < 1 {
		panic("limit must be at least 1")
	}

	var sb strings.Builder

	pg.generate(pg.re, &sb, limit)

	return sb.String()
}
