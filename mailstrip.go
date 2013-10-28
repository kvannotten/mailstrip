// mailstrip is a Go library that parses email text and strips it of
// signatures and reply quotes. It is a port of email_reply_parser,
// GitHub's library for parsing email replies.
//
// see https://github.com/github/email_reply_parser
package mailstrip

import (
	"bufio"
	"regexp"
	"strings"
	"unicode"
)

// Parse parses a plaintext email and returns the results. May return an error
// if the text contains a very long line (> bufio.MaxScanTokenSize, currently
// 64 * 1024).
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

// > I define UNIX as “30 definitions of regular expressions living under one
// > roof.”
// —Don Knuth
//
// Porting the Ruby regular expressions from email_reply_parser to Go required
// making the following changes:
//
// - Unlike most regexp flavors I'm familiar with, ^ and $ stand for beginning
//   and end of line respectively in Ruby. Getting the same behavior in Go
//   required enabling Go's multiline mode "(?m)" for these expressions.
// - Ruby's multiline mode "/m" is the same as Go's "(?s)" flag. Both are used
//   to make "." match "\n" characters.
var (
	multiLineReplyHeaderRegexp = regexp.MustCompile("(?sm)^(On\\s(?:.+)wrote:)$")
	sigRegexp                  = regexp.MustCompile("(--|__|(?m)\\w-$)|(?m)(^(\\w+\\s*){1,3} " + reverseString("Sent from my") + "$)")
	quotedRegexp               = regexp.MustCompile("(?m)(>+)$")
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

	// We might get a bufio.ErrTooLong here.
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

// Email contains the parsed contents of an email.
type Email []*Fragment

// String returns the non-Hidden() fragments of the Email.
func (e Email) String() string {
	results := []string{}
	for _, fragment := range e {
		if fragment.Hidden() {
			continue
		}

		results = append(results, fragment.String())
	}

	result := strings.Join(results, "\n")
	result = strings.TrimRightFunc(result, unicode.IsSpace)
	return result
}

// Fragment contains a parsed section of an email.
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

// Signature returns if the fragment is a signature or not.
func (f *Fragment) Signature() bool {
	return f.signature
}

// Signature returns if the fragment is a quote or not.
func (f *Fragment) Quoted() bool {
	return f.quoted
}

// Signature returns if the fragment is considered hidden or not.
func (f *Fragment) Hidden() bool {
	return f.hidden
}

// String returns the content of the fragment.
func (f *Fragment) String() string {
	return f.content
}
