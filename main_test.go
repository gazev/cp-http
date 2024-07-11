package main

import "testing"

type Test struct {
	href   string
	result bool
	out    string
}

var tests []Test

func TestSanitizeRelativeHref(t *testing.T) {
	tests = []Test{
		Test{href: "", result: false, out: ""},
		Test{href: "#", result: false, out: ""},
		Test{href: "?", result: false, out: ""},
		Test{href: "?sort=name&order=asc", result: false, out: ""},
		Test{href: "?sort=namedirfirst&order=asc", result: false, out: ""},
		Test{href: ".", result: false, out: ""},
		Test{href: "..", result: false, out: ""},
		Test{href: "../", result: false, out: ""},
		Test{href: "./", result: false, out: ""},
		Test{href: "../back", result: false, out: ""},
		Test{href: "../../back2", result: false, out: ""},
		Test{href: "../home/privacy", result: false, out: ""},
		Test{href: "mailto:mail@company.com", result: false, out: ""},
		Test{href: "http://google.com", result: false, out: ""},
		Test{href: "https://somedomain.xyz/ab", result: false, out: ""},

		Test{href: "./a", result: true, out: "a"},
		Test{href: "./ab", result: true, out: "ab"},
		Test{href: "./abc", result: true, out: "abc"},
		Test{href: "/a", result: true, out: "a"},
		Test{href: "a", result: true, out: "a"},
		Test{href: "ab", result: true, out: "ab"},
		Test{href: "abc", result: true, out: "abc"},
		Test{href: "abc/", result: true, out: "abc/"},
		Test{href: "./", result: false, out: ""},
		Test{href: "/", result: false, out: ""},
	}

	for _, test := range tests {
		out, valid := SanitizeRelativeHref(test.href)
		if valid != test.result || out != test.out {
			t.Errorf("failed test, expected '%s' to return `%s, %v`, got `%s, %v`\n", test.href, test.out, test.result, out, valid)
		}
	}
}
