package minions

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"html/template"
	"io"
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

// XML outputs the data encoded as XML.
func XML(w http.ResponseWriter, r *http.Request, code int, data interface{}) error {
	w.Header().Add("content-type", "application/xml; charset=utf-8")
	w.WriteHeader(code)

	enc := xml.NewEncoder(w)
	enc.Indent("", "\t")

	err := enc.Encode(data)
	return err
}

// Templates is a collection of HTML templates in the html/template format of the
// go stdlib. The templates are loaded and parsed on the first request, on every request
// when reload is enabled on explicitly when Load() is called.
type Templates struct {
	dir       string
	templates *template.Template
	funcmap   template.FuncMap
	reload    bool
}

// NewTemplates creates a new template collection. The templates are loaded from dir
// on the first request or when Load() is called. When reload is true, the templates
// are reloaded on each request.
func NewTemplates(dir string, reload bool) Templates {
	return Templates{
		dir: dir,
		funcmap: template.FuncMap{
			"div": func(dividend, divisor int) float64 {
				return float64(dividend) / float64(divisor)
			},
			"dict": func(values ...interface{}) (map[string]interface{}, error) {
				if len(values)%2 != 0 {
					return nil, errors.New("invalid dict call")
				}
				dict := make(map[string]interface{}, len(values)/2)
				for i := 0; i < len(values); i += 2 {
					key, ok := values[i].(string)
					if !ok {
						return nil, errors.New("dict keys must be strings")
					}
					dict[key] = values[i+1]
				}
				return dict, nil
			},
		},
		reload: reload,
	}
}

// Funcs adds the elements of the argument map to the template's function map.
// The return value is the updated template.
func (tpl Templates) Funcs(funcmap template.FuncMap) Templates {
	// create new funcmap to avoid race conditions
	newmap := template.FuncMap{}
	for name, fn := range tpl.funcmap {
		newmap[name] = fn
	}

	// overwrite/add functions
	for name, fn := range funcmap {
		newmap[name] = fn
	}

	tpl.funcmap = newmap
	return tpl
}

// Load loads any template from the filesystem.
func (tpl Templates) Load() (Templates, error) {
	tpl.templates = template.New("").Funcs(tpl.funcmap)
	err := filepath.Walk(tpl.dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			tpl.templates, err = tpl.templates.
				New(strings.TrimPrefix(path, tpl.dir)).
				Parse(string(b))
			return err
		}
		return nil
	})

	return tpl, err
}

// HTML outputs a rendered HTML template to the client.
func (tpl Templates) HTML(w http.ResponseWriter, r *http.Request, code int, name string, data interface{}) error {
	w.Header().Add("content-type", "text/html; charset=utf-8")
	w.WriteHeader(code)

	return tpl.Execute(w, name, data)
}

// Execute outputs a rendered template to the Writer. If you want to stream
// HTML to an ResponseWriter, use HTML(..) as it sets some required headers.
func (tpl Templates) Execute(w io.Writer, name string, data interface{}) error {
	// reload templates in debug mode
	if tpl.reload {
		var err error
		tpl, err = tpl.Load()
		if err != nil {
			return err
		}
	}

	// clone underlying templates, so we can safely update the functions
	templates, err := tpl.templates.Clone()
	if err != nil {
		return err
	}

	templates.Funcs(tpl.funcmap) // update funcmap
	err = templates.ExecuteTemplate(w, name, data)
	if err != nil {
		return err
	}

	return nil
}
