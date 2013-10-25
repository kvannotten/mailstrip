// mailstrip is a small package to parse plain text email content.  The goal is
// to identify which fragments are quoted, part of a signature, or original
// body content.  We want to support both top and bottom posters, so no simple
// "REPLY ABOVE HERE" content is used.
//
// Beyond RFC 5322 (which is handled by the [Ruby mail gem][mail]), there
// aren't any real standards for how emails are created.  This attempts to
// parse out common conventions for things like replies:
//
//     this is some text
//
//     On <date>, <author> wrote:
//     > blah blah
//     > blah blah
//
// ... and signatures:
//
//     this is some text
//
//     --
//     Bob
//     http://homepage.com/~bob
//
// Each of these are parsed into Fragment objects.
//
// mailstrip also attempts to figure out which of these blocks should be hidden
// from users.
//
// [mail]: https://github.com/mikel/mail
package mailstrip

import (
	"bufio"
	"regexp"
	"strings"
	"unicode"
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
	multiLineReplyHeaderRegexp = regexp.MustCompile("(?m)^(On\\s(?:.+)wrote:)$")
	sigRegexp                  = regexp.MustCompile("(--|__|\\w-$)|(?m)(^(\\w+\\s*){1,3} " + reverseString("Sent from my") + "$)")
	quotedRegexp               = regexp.MustCompile("(>+)$")
	quoteHeaderRegexp          = regexp.MustCompile("(?m)^:etorw.*nO$")
)

func (p *parser) Parse(text string) (Email, error) {
	// Normalize line endings.
	text = strings.Replace(text, "\r\n", "\n", -1)

	// Check for multi-line reply headers. Some clients break up
	// the "On DATE, NAME <EMAIL> wrote:" line into multiple lines.
	// @TODO: email_reply_parser might be buggy here, this regexp could
	// match several times and we should probably handle each occurence.
	if m := multiLineReplyHeaderRegexp.FindStringSubmatch(text); len(m) == 2 {
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

	// Finish up the final fragment.  Finishing a fragment will detect any
	// attributes (hidden, signature, reply), and join each line into a
	// string.
	p.finishFragment()

	// Now that parsing is done, reverse the order.
	reverseFragments(p.fragments)

	// @TODO Write a test that exceeds the scanner buffer / triggers error
	err := scanner.Err()
	return Email(p.fragments), err
}

// scaneLine scans the given line of text and figures out which fragment it
// belongs to.
func (p *parser) scanLine(line string) {
	sigMatch := sigRegexp.MatchString(line)
	if !sigMatch {
		line = strings.TrimLeftFunc(line, unicode.IsSpace)
	}

	// We're looking for leading `>`'s to see if this line is part of a
	// quoted Fragment.
	isQuoted := quotedRegexp.MatchString(line)

	// Mark the current Fragment as a signature if the current line is empty
	// and the Fragment starts with a common signature indicator.
	if p.fragment != nil && line == "" {
		lastLine := p.fragment.lines[len(p.fragment.lines)-1]
		if sigRegexp.MatchString(lastLine) {
			p.fragment.signature = true
			p.finishFragment()
		}
	}

	// If the line matches the current fragment, add it.  Note that a common
	// reply header also counts as part of the quoted Fragment, even though
	// it doesn't start with `>`.
	if p.fragment != nil &&
		((p.fragment.quoted == isQuoted) ||
			(p.fragment.quoted && (p.quoteHeader(line) || line == ""))) {
		p.fragment.lines = append(p.fragment.lines, line)

		// Otherwise, finish the fragment and start a new one.
	} else {
		p.finishFragment()
		p.fragment = &Fragment{quoted: isQuoted, lines: []string{line}}
	}
}

// quoteHeader detects if a given line is a header above a quoted area.  It is
// only checked for lines preceding quoted regions. Returns true if the line is
// a valid header, or false.
// @TODO: does this have to be a function?
func (p *parser) quoteHeader(line string) bool {
	return quoteHeaderRegexp.MatchString(line)
}

// finishFragment builds the fragment string and reverses it, after all lines
// have been added.  It also checks to see if this Fragment is hidden.  The
// hidden Fragment check reads from the bottom to the top.
//
// Any quoted Fragments or signature Fragments are marked hidden if they are
// below any visible Fragments.  Visible Fragments are expected to contain
// original content by the author.  If they are below a quoted Fragment, then
// the Fragment should be visible to give context to the reply.
//
//     some original text (visible)
//
//     > do you have any two's? (quoted, visible)
//
//     Go fish! (visible)
//
//     > -- > Player 1 (quoted, hidden)
//
//     -- Player 2 (signature, hidden)
func (p *parser) finishFragment() {
	if p.fragment != nil {
		p.fragment.finish()
		if !p.foundVisible {
			if p.fragment.quoted || p.fragment.signature ||
				strings.TrimSpace(p.fragment.String()) == "" {
				p.fragment.hidden = true
			} else {
				p.foundVisible = true
			}
		}
		p.fragments = append(p.fragments, p.fragment)
	}
	p.fragment = nil
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
	lines     []string
	content   string
	hidden    bool
	signature bool
	quoted    bool
}

// finish builds the string content by joining the lines and reversing them.
func (f *Fragment) finish() {
	f.content = strings.Join(f.lines, "\n")
	f.lines = nil
	f.content = reverseString(f.content)
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
	return f.content
}
