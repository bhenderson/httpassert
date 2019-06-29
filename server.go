package httpassert

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// NotFound can be rewritten to return a different status code or other behavior
var NotFound http.Handler = http.HandlerFunc(http.NotFound)

var testServers []*Server

// Assert is a package level convenience method to check if all Servers
// created have been validated.
func Assert(t testing.TB) bool {
	t.Helper()

	pass := true
	for _, s := range testServers {
		pass = s.Assert(t) && pass
	}
	return pass
}

// Server is a mocking http server that keeps track of intended and unintended
// calls. This allows for checking that http calls were made correctly and that
// no other calls were made unintentionally.
type Server struct {
	Name          string
	Server        *httptest.Server
	ExpectedCalls []ExpectedCall

	m sync.Mutex
}

// New creates a new Server using httptest, starts listening and writes the address to url.
func New(name string, url *string) *Server {
	s := new(Server)

	hs := httptest.NewServer(s)
	*url = hs.URL

	s.Name = name
	s.Server = hs

	// register
	testServers = append(testServers, s)
	return s
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for i := range s.ExpectedCalls {
		if s.ExpectedCalls[i].Match(r) {
			s.ExpectedCalls[i].ServeHTTP(w, r)
			return
		}
	}
	ec := ExpectedCall{
		Method: r.Method,
		Path:   r.URL.Path,
	}
	ec.ServeHTTP(w, r)
	s.Expect(ec)
}

func (s *Server) Assert(t testing.TB) bool {
	t.Helper()
	pass := true

	for _, ec := range s.ExpectedCalls {
		if ec.Calls < 0 {
			t.Errorf(
				"Server(%s) got (%d) unexpected calls to %s %s",
				s.Name, -ec.Calls, ec.Method, ec.Path,
			)
			pass = false
		}
		if ec.Calls > 0 {
			t.Errorf(
				"Server(%s) expected (%d) more calls to %s %s",
				s.Name, ec.Calls, ec.Method, ec.Path,
			)
			pass = false
		}
	}
	return pass
}

// Close closes the listener
func (s *Server) Close() {
	s.Server.Close()
}

func (s *Server) Expect(ec ExpectedCall) {
	s.m.Lock()
	defer s.m.Unlock()

	s.ExpectedCalls = append(s.ExpectedCalls, ec)
}

func NewCall(m, p string, h http.Handler) ExpectedCall {
	return ExpectedCall{
		Method:  m,
		Path:    p,
		Handler: h,
	}
}

type ExpectedCall struct {
	Method  string
	Path    string
	Handler http.Handler
	Calls   int

	m sync.Mutex
}

// Match matches on r.Method and r.URL.Path prefix. More extensive matching can be done in Handler.
func (ec *ExpectedCall) Match(r *http.Request) bool {
	return ec.Method == r.Method && strings.HasPrefix(r.URL.Path, ec.Path)
}

func (ec *ExpectedCall) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := ec.Handler
	if h == nil {
		h = NotFound
	}
	h.ServeHTTP(w, r)
	ec.Increment(-1)
}

// Increment allows changing Calls in a thread-safe way.
// use negative numbers to decrement.
func (ec *ExpectedCall) Increment(i int) {
	ec.m.Lock()
	defer ec.m.Unlock()

	ec.Calls += i
}
