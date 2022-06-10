package playerconn

import (
	"testing"
)

func TestParseConnTableDriven(t *testing.T) {
	var testCases = []struct {
		name   string
		input  string
		user   string
		pass   string
		status bool
	}{
		{"empty", "", "", "", false},
		{"no_args", "connect", "", "", false},
		{"blank_args", "connect     ", "", "", false},
		{"misspelled", "cronect user pass", "", "", false},
		{"simple", "connect user pass", "user", "pass", true},
		{"ws_padded", "  connect    user   pass    ", "user", "pass", true},
		{"special_chars", "connect Us3r_N4+e _Ab13+!-/~=", "Us3r_N4+e", "_Ab13+!-/~=", true},
		{"passphrase", "connect user simple pass phrase", "user", "simple pass phrase", true},
	}

	for _, params := range testCases {
		t.Run(params.name, func(t *testing.T) {
			user, pass, ok := parseConnect(params.input)
			if !(user == params.user && pass == params.pass && ok == params.status) {
				t.Errorf("got (%s, %s, %v), wanted (%s, %s, %v)", user, pass, ok,
					params.user, params.pass, params.status)
			}
		})
	}
}
