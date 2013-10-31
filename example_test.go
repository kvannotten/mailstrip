package mailstrip

import (
	"fmt"
)

func ExampleParse() {
	text :=
		`Yeah, that works!

-Bob

On 01/03/11 7:07 PM, Alice wrote:
> Hi Bob,
>
> can I push the latest release later tonight?
`

	email := Parse(text)
	fmt.Printf("result: %q\n", email.String())
	// Output:
	// result: "Yeah, that works!"
}
