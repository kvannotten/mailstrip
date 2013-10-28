package mailstrip

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

var tests = []struct {
	name    string    // name of the test, from email_reply_parser
	fixture string    // fixture file to parse
	checks  []checker // checks to perform
}{
	{
		"test_reads_simple_body",
		"email_1_1",
		[]checker{
			&attributeChecker{"Quoted", []bool{false, false, false}},
			&attributeChecker{"Signature", []bool{false, true, true}},
			&attributeChecker{"Hidden", []bool{false, true, true}},
			&contentChecker{0, equalsString(`Hi folks

What is the best way to clear a Riak bucket of all key, values after
running a test?
I am currently using the Java HTTP API.
`),
			},
		},
	},
	{
		"test_reads_top_post",
		"email_1_3",
		[]checker{
			&attributeChecker{"Quoted", []bool{false, false, true, false, false}},
			&attributeChecker{"Hidden", []bool{false, true, true, true, true}},
			&attributeChecker{"Signature", []bool{false, true, false, false, true}},
			&contentChecker{0, regexp.MustCompile("(?m)^Oh thanks.\n\nHaving")},
			&contentChecker{1, regexp.MustCompile("(?m)^-A")},
			&contentChecker{2, regexp.MustCompile("(?m)^On [^\\:]+\\:")},
			&contentChecker{4, regexp.MustCompile("^_")},
		},
	},
	{
		"test_reads_bottom_post",
		"email_1_2",
		[]checker{
			&attributeChecker{"Quoted", []bool{false, true, false, true, false, false}},
			&attributeChecker{"Signature", []bool{false, false, false, false, false, true}},
			&attributeChecker{"Hidden", []bool{false, false, false, true, true, true}},
			&contentChecker{0, equalsString("Hi,")},
			&contentChecker{1, regexp.MustCompile("(?m)^On [^\\:]+\\:")},
			&contentChecker{2, regexp.MustCompile("(?m)^You can list")},
			&contentChecker{3, regexp.MustCompile("(?m)^> ")},
			&contentChecker{5, regexp.MustCompile("(?m)^_")},
		},
	},
	{
		"test_recognizes_date_string_above_quote",
		"email_1_4",
		[]checker{
			&contentChecker{0, regexp.MustCompile("(?m)^Awesome")},
			&contentChecker{1, regexp.MustCompile("(?m)^On")},
			&contentChecker{1, regexp.MustCompile("Loader")},
		},
	},
	{
		"test_a_complex_body_with_only_one_fragment",
		"email_1_5",
		[]checker{fragmentCountChecker(1)},
	},
	{
		"test_reads_email_with_correct_signature",
		"correct_sig",
		[]checker{
			&attributeChecker{"Quoted", []bool{false, false}},
			&attributeChecker{"Signature", []bool{false, true}},
			&attributeChecker{"Hidden", []bool{false, true}},
			&contentChecker{1, regexp.MustCompile("(?m)^-- \nrick")},
		},
	},
	{
		"test_deals_with_multiline_reply_headers",
		"email_1_6",
		[]checker{
			&contentChecker{0, regexp.MustCompile("(?m)^I get")},
			&contentChecker{1, regexp.MustCompile("(?m)^On")},
			&contentChecker{1, regexp.MustCompile("Was this")},
		},
	},
}

func TestParse(t *testing.T) {
	for _, test := range tests {
		t.Logf("===== %s =====", test.name)
		text, err := loadFixture(test.fixture)
		if err != nil {
			t.Errorf("could not load fixture: %s", err)
			continue
		}

		parsed, err := Parse(text)
		if err != nil {
			t.Error(err)
			continue
		}

		for _, check := range test.checks {
			if err := check.Check(parsed); err != nil {
				t.Error(err)
			}
		}
	}
}

type checker interface {
	Check(email Email) error
}

type attributeChecker struct {
	attribute string
	values    []bool
}

func (c *attributeChecker) Check(email Email) error {
	expectedCount := len(c.values)
	gotCount := len(email)
	if gotCount != expectedCount {
		return fmt.Errorf("wrong fragment count: %d != %d", gotCount, expectedCount)
	}

	for i, fragment := range email {
		var val bool
		// could also use reflect, but seems overkill for this
		switch c.attribute {
		case "Hidden":
			val = fragment.Hidden()
		case "Quoted":
			val = fragment.Quoted()
		case "Signature":
			val = fragment.Signature()
		default:
			return fmt.Errorf("Unknown attribute: %s", c.attribute)
		}

		if val != c.values[i] {
			return fmt.Errorf("Invalid %s() value in fragment #%d: %t != %t", c.attribute, i, val, c.values[i])
		}
	}

	return nil
}

type contentChecker struct {
	fragmentId int
	content    stringMatcher
}

func (c *contentChecker) Check(email Email) error {
	fragment := email[c.fragmentId]
	content := fragment.String()
	if !c.content.MatchString(content) {
		return fmt.Errorf("String(): %q did not match %s", content, c.content)
	}
	return nil
}

type fragmentCountChecker int

func (c fragmentCountChecker) Check(email Email) error {
	expectedCount := int(c)
	gotCount := len(email)
	if gotCount != expectedCount {
		return fmt.Errorf("wrong fragment count: %d != %d", gotCount, expectedCount)
	}
	return nil
}

type stringMatcher interface {
	MatchString(string) bool
}

type equalsString string

func (s equalsString) MatchString(str string) bool {
	return str == string(s)
}

var (
	_, srcPath, _, _ = runtime.Caller(0)
	fixturesDir      = filepath.Join(filepath.Dir(srcPath), "fixtures")
)

func loadFixture(name string) (string, error) {
	fixturePath := filepath.Join(fixturesDir, name+".txt")
	data, err := ioutil.ReadFile(fixturePath)
	return string(data), err
}
