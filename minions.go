package minions

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// JSON outputs the data encoded as JSON.
func JSON(w http.ResponseWriter, r *http.Request, code int, data interface{}) error {
	w.Header().Add("content-type", "application/json; charset=utf-8")
	w.WriteHeader(code)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")

	err := enc.Encode(data)
	return err
}

// Templates is a collection of HTML templates in the html/template format of the
// go stdlib. The templates are loaded and parsed on the first request, on every request
// when reload is enabled on explicetly when Load() is called.
type Templates struct {
	dir     string
	tpl     *template.Template
	funcmap template.FuncMap
	reload  bool
}

// NewTemplates creates a new template collection. The templates are loaded from dir
// on the first request or when Load() is called. When reload is true, the templates
// are reloaded on each request.
func NewTemplates(dir string, reload bool) *Templates {
	return &Templates{
		dir:     dir,
		funcmap: template.FuncMap{},
		reload:  reload,
	}
}

// Funcs adds the elements of the argument map to the template's function map.
// It panics if a value in the map is not a function with appropriate return
// type. However, it is legal to overwrite elements of the map. The return
// value is the template, so calls can be chained.
func (tpl *Templates) Funcs(funcmap template.FuncMap) *Templates {
	tpl.tpl.Funcs(template.FuncMap(funcmap))
	return tpl
}

// Load loads any template from the filesystem and adds the parsed template
// to the template instance.
func (tpl *Templates) Load() (*Templates, error) {
	tpl.tpl = template.New("").Funcs(tpl.funcmap)
	err := filepath.Walk(tpl.dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			tpl.tpl, err = tpl.tpl.
				New(strings.TrimPrefix(path, tpl.dir)).
				Parse(string(b))
			return err
		}
		return nil
	})

	return tpl, err
}

// HTML outputs a rendered HTML template to the client.
func (tpl *Templates) HTML(w http.ResponseWriter, r *http.Request, code int, name string, data interface{}) error {
	var err error

	// reload templates in debug mode
	if tpl.tpl == nil || tpl.reload {
		tpl, err = tpl.Load()
		if err != nil {
			return err
		}
	}

	w.Header().Add("content-type", "text/html; charset=utf-8")
	w.WriteHeader(code)

	err = tpl.tpl.ExecuteTemplate(w, name, data)
	if err != nil {
		return err
	}

	return nil
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

// BindingResult holds validation errors of the binding process from a HTML
// form to a Go struct.
type BindingResult map[string]string

// Valid returns whether the binding was successfull or not.
func (br BindingResult) Valid() bool {
	return len(br) == 0
}

// Fail marks the binding as failed and stores an error for the given field
// that caused the form binding to fail.
func (br BindingResult) Fail(field, err string) {
	br[field] = err
}

// Include copies all errors and state of a binding result
func (br BindingResult) Include(other BindingResult) {
	for field, err := range other {
		br.Fail(field, err)
	}
}

// V is a helper type to quickly build variable maps for templates.
type V map[string]interface{}

// MarshalJSON implements the json.Marshaler interface.
func (v V) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}(v))
}
