package core_test

import (
	"fmt"

	"github.com/thedavidweng/money/internal/core"
)

func ExampleFormatMinorUnits() {
	fmt.Println(core.FormatMinorUnits(12345, "USD"))
	fmt.Println(core.FormatMinorUnits(-50, "USD"))
	fmt.Println(core.FormatMinorUnits(0, "USD"))
	// Output:
	// 123.45
	// -0.50
	// 0.00
}

func ExampleParseUnsignedDecimalMinorUnits() {
	minor, err := core.ParseUnsignedDecimalMinorUnits("123.45")
	fmt.Println(minor, err)

	minor, err = core.ParseUnsignedDecimalMinorUnits("1,000.50")
	fmt.Println(minor, err)

	minor, err = core.ParseUnsignedDecimalMinorUnits("42")
	fmt.Println(minor, err)
	// Output:
	// 12345 <nil>
	// 100050 <nil>
	// 4200 <nil>
}
