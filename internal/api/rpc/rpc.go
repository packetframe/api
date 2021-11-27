package rpc

import (
	"net/http"
	"net/url"
)

var Server string

// Call makes a HTTP GET request to a RPC endpoint
func Call(path string, args map[string]string) error {
	if Server == "" {
		return nil
	}

	u, err := url.Parse(Server)
	if err != nil {
		return err
	}
	u.Path = path

	if args != nil {
		q := u.Query()
		for k, v := range args {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	_, err = http.Get(u.String())
	return err
}
