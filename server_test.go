package httpassert

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type helperT struct {
	testing.TB

	errors []string
}

func (t *helperT) Errorf(format string, args ...interface{}) {
	t.errors = append(t.errors, fmt.Sprintf(format, args...))
}

func (t *helperT) Helper() {}

func assertResponse(t *testing.T, code int, r *http.Response, err error) bool {
	return assertNoError(t, err) && assertStatusCode(t, code, r)
}

func assertNoError(t *testing.T, err error) bool {
	t.Helper()

	if err != nil {
		t.Errorf("Expected no error, got (%v)", err)
		return false
	}
	return true
}

func assertStatusCode(t *testing.T, code int, r *http.Response) bool {
	t.Helper()

	if code != r.StatusCode {
		t.Errorf("Expected status code of (%d), got (%d)", code, r.StatusCode)
		return false
	}
	return true
}

func assertExpectedCalls(t *testing.T, exp, act []string) {
	t.Helper()

	if !reflect.DeepEqual(exp, act) {
		t.Errorf("Expected errors\n\t%s\ngot\n\t%s", strings.Join(exp, "\n\t"), strings.Join(act, "\n\t"))
	}
}

func TestServer(t *testing.T) {
	var (
		ht     = new(helperT)
		u      string
		called int
	)
	s := New("testserver", &u)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
	})
	s.Expect(&ExpectedCall{Method: "GET", Path: "/endpoint", Calls: 1, Handler: h})
	s.Expect(&ExpectedCall{Method: "PATCH", Path: "/missed", Calls: 2})

	r, err := http.Get(u + "/endpoint")
	assertResponse(t, 200, r, err)
	r, err = http.Get(u + "/endpoint")
	assertResponse(t, 200, r, err)
	r, err = http.Post(u+"/endpoint", "", nil)
	assertResponse(t, 404, r, err)

	expCalled := 2
	if called != expCalled {
		t.Errorf("Expected called to be (%d), got (%d)", expCalled, called)
	}
	if s.Assert(ht) {
		t.Errorf("Expected s.Assert to not pass")
	}
	exp := []string{
		"Server(testserver) got (1) unexpected calls to GET /endpoint",
		"Server(testserver) expected (2) more calls to PATCH /missed",
		"Server(testserver) got (1) unexpected calls to POST /endpoint",
	}
	assertExpectedCalls(t, exp, ht.errors)

	t.Run("no handler", func(t *testing.T) {
		var (
			ht = new(helperT)
			u  string
		)
		s := New("testserver", &u)

		s.Expect(&ExpectedCall{Method: "GET", Path: "/", Calls: 1})

		r, err := http.Get(u)
		assertResponse(t, 404, r, err)
		if !s.Assert(ht) {
			t.Errorf("Expected s.Assert to pass")
		}
		assertExpectedCalls(t, nil, ht.errors)
	})
}
