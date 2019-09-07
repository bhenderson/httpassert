package httpassert_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/bhenderson/httpassert"
)

var backendURL string

func Example() {
	var t *testing.T

	s := httpassert.New("my-server", &backendURL)

	s.Expect(&httpassert.ExpectedCall{
		Method: "GET",
		Path:   "/my/path",
		Calls:  1,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"hello": "world"}`))
		}),
	})

	defer httpassert.Assert(t)

	act := clientDo()
	exp := "world"

	if exp != act {
		t.Errorf("expected %q, got %q", exp, act)
	}
}

func clientDo() string {
	resp, _ := http.Get(backendURL)
	var b struct{ Hello string }
	json.NewDecoder(resp.Body).Decode(&b)
	return b.Hello
}
