package mailstrip

import (
	"bufio"
	"regexp"
	"strings"
)

func Parse(text string) (Email, error) {
	p := &parser{}
	return p.Parse(text)
}

type parser struct {
	// This determines if any 'visible' Fragment has been found.  Once any
	// visible Fragment is found, stop looking for hidden ones.
	foundVisible bool
	// This instance variable points to the current Fragment.  If the matched
	// line fits, it should be added to this Fragment.  Otherwise, finish it and
	// start a new Fragment.
	fragment *Fragment
	// The fragments parsed so far
	fragments []*Fragment
}

// @TODO: figure out if we should use the POSIX regexp flavor of go for our
// regular expressions
var (
	multiLineReplyHeader = regexp.MustCompile("(?m)^(On\\s(?:.+)wrote:)$")
)

func (p *parser) Parse(text string) (Email, error) {
	// Normalize line endings.
	text = strings.Replace(text, "\r\n", "\n", -1)

	// Check for multi-line reply headers. Some clients break up
	// the "On DATE, NAME <EMAIL> wrote:" line into multiple lines.
	// @TODO: email_reply_parser might be buggy here, this regexp could
	// match several times and we should probably handle each occurence.
	if m := multiLineReplyHeader.FindStringSubmatch(text); len(m) == 2 {
		// Remove all new lines from the reply header.
		text = strings.Replace(text, m[1], strings.Replace(m[1], "\n", "", -1), -1)
	}

	// The text is reversed initially due to the way we check for hidden
	// fragments.
	text = reverseString(text)

	// Use the StringScanner to pull out each line of the email content.
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		p.scanLine(scanner.Text())
	}

	p.finishFragment()

	// Finish up the final fragment.  Finishing a fragment will detect any
	// attributes (hidden, signature, reply), and join each line into a
	// string.
	reverseFragments(p.fragments)

	// @TODO Write a test that exceeds the scanner buffer / triggers error
	err := scanner.Err()
	return Email(p.fragments), err
}

func (p *parser) scanLine(line string) {
	
}

func (p *parser) finishFragment() {
	
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func reverseFragments(f []*Fragment) {
	for i, j := 0, len(f)-1; i < j; i, j = i+1, j-1 {
		f[i], f[j] = f[j], f[i]
	}
}

type Email []*Fragment

// String returns the non-Hidden() fragments of the Email.
func (e Email) String() string {
	return ""
}

type Fragment struct {
	text      string
	hidden    bool
	signature bool
	quoted    bool
}

func (f *Fragment) Signature() bool {
	return f.signature
}

func (f *Fragment) Quoted() bool {
	return f.quoted
}

func (f *Fragment) Hidden() bool {
	return f.hidden
}

func (f *Fragment) String() string {
	return f.text
}
