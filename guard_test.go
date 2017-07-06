package minions

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProtectAnonymous(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	var called200, called401, called403 bool
	fn200 := func(w http.ResponseWriter, r *http.Request) { called200 = true }
	fn401 := func(w http.ResponseWriter, r *http.Request) { called401 = true }
	fn403 := func(w http.ResponseWriter, r *http.Request) { called403 = true }

	NewGuard().
		UnauthorizedFn(fn401).
		ForbiddenFn(fn403).
		Protect(fn200, "anything")(rec, req)
	if called200 {
		t.Error("200 handler called")
	}
	if !called401 {
		t.Error("401 handler _NOT_ called")
	}
	if called403 {
		t.Error("403 handler called")
	}
}

func TestProtectUnauthorized(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	var called200, called401, called403 bool
	fn200 := func(w http.ResponseWriter, r *http.Request) { called200 = true }
	fn401 := func(w http.ResponseWriter, r *http.Request) { called401 = true }
	fn403 := func(w http.ResponseWriter, r *http.Request) { called403 = true }

	NewGuard().
		UnauthorizedFn(fn401).
		ForbiddenFn(fn403).
		PrincipalFn(func(r *http.Request) Principal {
			return TestUser{
				FnID:            func() string { return "test" },
				FnAuthenticated: func() bool { return false },
				FnHasAnyRole:    func(roles ...string) bool { return false },
			}
		}).
		Protect(fn200, "anything")(rec, req)
	if called200 {
		t.Error("200 handler called")
	}
	if !called401 {
		t.Error("401 handler _NOT_ called")
	}
	if called403 {
		t.Error("403 handler called")
	}
}

func TestForbidden(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	var called200, called401, called403 bool
	fn200 := func(w http.ResponseWriter, r *http.Request) { called200 = true }
	fn401 := func(w http.ResponseWriter, r *http.Request) { called401 = true }
	fn403 := func(w http.ResponseWriter, r *http.Request) { called403 = true }

	NewGuard().
		UnauthorizedFn(fn401).
		ForbiddenFn(fn403).
		PrincipalFn(func(r *http.Request) Principal {
			return TestUser{
				FnID:            func() string { return "test" },
				FnAuthenticated: func() bool { return true },
				FnHasAnyRole:    func(roles ...string) bool { return false },
			}
		}).
		Protect(fn200, "anything")(rec, req)
	if called200 {
		t.Error("200 handler called")
	}
	if called401 {
		t.Error("401 handler called")
	}
	if !called403 {
		t.Error("403 handler _NOT_ called")
	}
}

func TestAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	var called200, called401, called403 bool
	fn200 := func(w http.ResponseWriter, r *http.Request) { called200 = true }
	fn401 := func(w http.ResponseWriter, r *http.Request) { called401 = true }
	fn403 := func(w http.ResponseWriter, r *http.Request) { called403 = true }

	NewGuard().
		UnauthorizedFn(fn401).
		ForbiddenFn(fn403).
		PrincipalFn(func(r *http.Request) Principal {
			return TestUser{
				FnID:            func() string { return "test" },
				FnAuthenticated: func() bool { return true },
				FnHasAnyRole:    func(roles ...string) bool { return true },
			}
		}).
		Protect(fn200, "anything")(rec, req)
	if !called200 {
		t.Error("200 handler _NOT_ called")
	}
	if called401 {
		t.Error("401 handler called")
	}
	if called403 {
		t.Error("403 handler called")
	}
}

func TestDefaultUnauthorizedFn(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	NewGuard().
		PrincipalFn(func(r *http.Request) Principal {
			return TestUser{
				FnID:            func() string { return "test" },
				FnAuthenticated: func() bool { return false },
				FnHasAnyRole:    func(roles ...string) bool { return false },
			}
		}).
		Protect(func(w http.ResponseWriter, r *http.Request) {}, "anything")(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	if body := rec.Body.String(); body != "Unauthorized\n" {
		t.Errorf("expected body %q, got %q", "Unauthorized\n", body)
	}
}

func TestDefaultForbiddenFn(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	NewGuard().
		PrincipalFn(func(r *http.Request) Principal {
			return TestUser{
				FnID:            func() string { return "test" },
				FnAuthenticated: func() bool { return true },
				FnHasAnyRole:    func(roles ...string) bool { return false },
			}
		}).
		Protect(func(w http.ResponseWriter, r *http.Request) {}, "anything")(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
	if body := rec.Body.String(); body != "Forbidden\n" {
		t.Errorf("expected body %q, got %q", "Forbidden\n", body)
	}
}

type TestUser struct {
	FnID            func() string
	FnAuthenticated func() bool
	FnHasAnyRole    func(roles ...string) bool
}

func (u TestUser) Authenticated() bool             { return u.FnAuthenticated() }
func (u TestUser) ID() string                      { return u.FnID() }
func (u TestUser) HasAnyRole(roles ...string) bool { return u.FnHasAnyRole(roles...) }
