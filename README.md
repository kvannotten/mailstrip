# mailstrip

mailstrip is a [Go][2] library that parses email text and strips it of
signatures and reply quotes. It is a port of [email\_reply\_parser][1], GitHub's
library for parsing email replies.

## Differences to email_reply_parser

Most of mailstrip is a line-by-line port of email\_reply\_parser and it passes
all tests from the email\_reply\_parser test suite.

Additionally mailstrip detects forwarded fragments and considers them to be
visible text, see d321c10543f77c0beaacb40b04511e619f0652c6.

## Documentation

The API documentation can be found here:
http://godoc.org/github.com/ThomsonReutersEikon/mailstrip

## License

MIT License. See LICENSE file.

[1]: https://github.com/github/email_reply_parser
[2]: http://golang.org/
