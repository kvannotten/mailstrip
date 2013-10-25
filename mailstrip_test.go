package mailstrip

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"
)

var tests = []struct {
	name      string             // name of the test, from email_reply_parser
	fixture   string             // fixture file to parse
	fragments []expectedFragment // expected fragments
}{
	{
		"test_reads_simple_body",
		"email_1_1",
		[]expectedFragment{
			{
				hidden:    false,
				quoted:    false,
				signature: false,
				text: equalsString(`Hi folks

What is the best way to clear a Riak bucket of all key, values after
running a test?
I am currently using the Java HTTP API.
`),
			},
			{
				hidden:    true,
				quoted:    false,
				signature: true,
				text:      nil,
			},
			{
				hidden:    true,
				quoted:    false,
				signature: true,
				text:      nil,
			},
		},
	},
}

func TestParse(t *testing.T) {
	for _, test := range tests {
		t.Log(test.name)
		text, err := loadFixture(test.fixture)
		if err != nil {
			t.Errorf("could not load fixture: %s", err)
			continue
		}

		parsed, err := Parse(text)
		if err != nil {

		}
		gotCount := len(parsed)
		expectedCount := len(test.fragments)
		if gotCount != expectedCount {
			t.Errorf("wrong fragment count: %d != %d", gotCount, expectedCount)
			continue
		}

		for i, fragment := range parsed {
			expectedFragment := test.fragments[i]
			t.Logf("fragment #%d", i)
			if hidden := fragment.Hidden(); hidden != expectedFragment.hidden {
				t.Errorf("Hidden(): %t != %t", hidden, expectedFragment.hidden)
			}

			if quoted := fragment.Quoted(); quoted != expectedFragment.quoted {
				t.Errorf("Quoted(): %d != %d", quoted, expectedFragment.quoted)
			}

			if signature := fragment.Signature(); signature != expectedFragment.signature {
				t.Errorf("Signature(): %d != %d", signature, expectedFragment.signature)
			}

			if s := fragment.String(); !expectedFragment.text.MatchString(s) {
				t.Errorf("String(): %q did not match: %#v", s, fragment.text)
			}
		}
	}
}

type expectedFragment struct {
	hidden    bool          // expected Hidden() value
	quoted    bool          // expected Quoted() value
	signature bool          // expected Signature() value
	text      stringMatcher // expected String() value
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
