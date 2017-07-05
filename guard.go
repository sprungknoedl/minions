package minions

import "net/http"

// Guard enforces a role based security model on protected resources. Before
// a visitor can access a procted resource, he must be authenticated and have
// the required roles to access. Authentication is outside the scope of the
// guard, the principal is fetched using a provided PrincipalFn.
type Guard struct {
	unauthorized http.HandlerFunc
	forbidden    http.HandlerFunc
	principal    func(r *http.Request) Principal
}

// NewGuard creates a new guard. You wan't to overwrite at lest the PrincipalFn
// before the guard is of any use, otherwise the guard always returns an
// unauthenticated Anonymous user that is denied access to every protected
// resource.
func NewGuard() *Guard {
	return &Guard{
		unauthorized: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		},
		forbidden: func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Forbidden", http.StatusForbidden)
		},
		principal: func(r *http.Request) Principal {
			return Anonymous{}
		},
	}
}

// PrincipalFn overwrites the function used to fetch a principal object for
// a request. The return value is the guard, so calls can be chained.
func (g *Guard) PrincipalFn(fn func(r *http.Request) Principal) *Guard {
	g.principal = fn
	return g
}

// UnauthorizedFn overwrites the function used when a principal is missing or
// not authenticated for a request. The return value is the guard, so calls can be chained.
func (g *Guard) UnauthorizedFn(fn http.HandlerFunc) *Guard {
	g.unauthorized = fn
	return g
}

// ForbiddenFn overwrites the function used when a principal is not allowed for
// a request. The return value is the guard, so calls can be chained.
func (g *Guard) ForbiddenFn(fn http.HandlerFunc) *Guard {
	g.forbidden = fn
	return g
}

// Protect requires that the principal has at least one of the provided roles before
// the request is forwarded to the protected handler.
func (g *Guard) Protect(fn http.HandlerFunc, roles ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal := g.principal(r)
		if !principal.Authenticated() {
			g.unauthorized(w, r)
			return
		}

		if !principal.HasAnyRole(roles...) {
			g.forbidden(w, r)
			return
		}

		fn(w, r)
	}
}

// Principal is an entity that can be authenticated and verified.
type Principal interface {
	ID() string
	Authenticated() bool
	HasAnyRole(roles ...string) bool
}

// Anonymous implements the Principal interface for unauthenticated
// users and can be used as a fallback principal when none is set
// in the current session.
type Anonymous struct{}

// Authenticated returns always false, because Anonymous users are not
// authenticated.
func (a Anonymous) Authenticated() bool { return false }

// ID retunrs always the string `anonymous` as ID for unauthenticated
// users.
func (a Anonymous) ID() string { return "anonymous" }

// HasAnyRole returns always false for any role, because Anonymous users
// are not authenticated.
func (a Anonymous) HasAnyRole(roles ...string) bool { return false }
