package db

import "testing"

func DNSScriptCompile(t *testing.T) {
	for _, tc := range []struct {
		Script      string
		ShouldError bool
	}{
		{`test`, true},
		{`
async function handleQuery(query) {
    return {
        "authoritative": true,
        "rrs": [
            {
                name: query.name,
                ttl: 300,
                type: "TXT",
                value: "Hello World"
            }
        ]
    }
}
`, false},
		{`async function handleQuery(query) {}`, false},
	} {
		err := DNSScriptCompile(tc.Script, "test")
		if tc.ShouldError && err == nil {
			t.Fatal("should error but didn't")
		} else if !tc.ShouldError && err != nil {
			t.Fatal("shouldn't have errored but did " + err.Error())
		}
	}
}
