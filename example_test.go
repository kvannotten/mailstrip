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

	email, err := Parse(text)
	if err != nil {
		fmt.Printf("err: %#v\n", err)
	} else {
		fmt.Printf("result: %q\n", email.String())
	}
	// Output:
	// result: "Yeah, that works!"
}
