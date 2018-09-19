package utils

import (
	"strings"
	"math/big"
)

// Number represents a number in a search query.  FHIR search params may define
// numbers to varying levels of precision, and the amount of precision affects
// the behavior of the query.  Number's value should only be interpreted in the
// context of the Precision supplied.  The Precision indicates the number of
// decimal places in the precision.
type Number struct {
	Value     *big.Rat
	Precision int
}

// String returns a string representation of the number, honoring the supplied
// precision.
func (n *Number) String() string {
	return n.Value.FloatString(n.Precision)
}

// RangeLowIncl represents the low end of a range to match against.  As
// the name suggests, the low end of the range is inclusive.
func (n *Number) RangeLowIncl() *big.Rat {
	return new(big.Rat).Sub(n.Value, n.rangeDelta())
}

// RangeHighExcl represents the high end of a range to match against.  As
// the name suggests, the high end of the range is exclusive.
func (n *Number) RangeHighExcl() *big.Rat {
	return new(big.Rat).Add(n.Value, n.rangeDelta())
}

// The FHIR spec defines equality for 100 to be the range [99.5, 100.5) so we
// must support min/max using rounding semantics. The basic algorithm for
// determining low/high is:
//   low  (inclusive) = n - 5 / 10^p
//   high (exclusive) = n + 5 / 10^p
// where n is the number and p is the count of the number's decimal places + 1.
//
// This function returns the delta ( 5 / 10^p )
func (n *Number) rangeDelta() *big.Rat {
	p := n.Precision + 1
	denomInt := new(big.Int).Exp(big.NewInt(int64(10)), big.NewInt(int64(p)), nil)
	denomRat, _ := new(big.Rat).SetString(denomInt.String())
	return new(big.Rat).Quo(new(big.Rat).SetInt64(5), denomRat)
}

// ParseNumber parses a numeric string into a Number object, maintaining the
// value and precision supplied.
func ParseNumber(numStr string) *Number {
	n := &Number{}

	numStr = strings.TrimSpace(numStr)
	n.Value, _ = new(big.Rat).SetString(numStr) // TODO: error handling
	i := strings.Index(numStr, ".")
	if i != -1 {
		n.Precision = len(numStr) - i - 1
	} else {
		n.Precision = 0
	}

	return n
}