package minions

import (
	"encoding/json"
	"encoding/xml"
	"errors"
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

// XML outputs the data encoded as XML.
func XML(w http.ResponseWriter, r *http.Request, code int, data interface{}) error {
	w.Header().Add("content-type", "application/xml; charset=utf-8")
	w.WriteHeader(code)

	enc := xml.NewEncoder(w)
	enc.Indent("", "\t")

	err := enc.Encode(data)
	return err
}

func NewFileServer(prefix string, root string) http.HandlerFunc {
	fs := http.Dir(root)
	handler := http.StripPrefix(prefix, http.FileServer(fs))
	return handler.ServeHTTP
}

// Templates is a collection of HTML templates in the html/template format of the
// go stdlib. The templates are loaded and parsed on the first request, on every request
// when reload is enabled on explicitly when Load() is called.
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
// The return value is the template, so calls can be chained.
func (tpl *Templates) Funcs(funcmap template.FuncMap) *Templates {
	for name, fn := range funcmap {
		tpl.funcmap[name] = fn
	}

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

			name := path
			name = strings.TrimPrefix(name, tpl.dir)
			name = strings.TrimLeft(name, "./")
			tpl.tpl, err = tpl.tpl.
				New(name).
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
