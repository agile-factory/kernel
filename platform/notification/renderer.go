package notification

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"strings"
	texttemplate "text/template"
)

//go:embed templates/*/*.tmpl
var templateFS embed.FS

// renderer compiles the embedded templates once and renders them per locale.
// Layout on disk: templates/<locale>/<name>.{subject,txt,html}.tmpl.
//
// The HTML variant uses html/template (auto-escaping); subject + text use
// text/template. Missing locales fall back to "en"; missing templates return
// an error (the consumer mapping is wrong if we ask for a non-existing one).
type renderer struct {
	subjects map[string]*texttemplate.Template // key: "<locale>/<name>"
	texts    map[string]*texttemplate.Template
	htmls    map[string]*htmltemplate.Template
}

func newRenderer() (*renderer, error) {
	r := &renderer{
		subjects: map[string]*texttemplate.Template{},
		texts:    map[string]*texttemplate.Template{},
		htmls:    map[string]*htmltemplate.Template{},
	}

	err := fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		body, err := fs.ReadFile(templateFS, path)
		if err != nil {
			return err
		}
		// path = "templates/<locale>/<name>.<kind>.tmpl"
		// e.g.  "templates/en/email_verification.html.tmpl"
		// fmt.Sscanf does not support C-style %[^/] character classes in Go;
		// parse with strings.Split + TrimSuffix instead.
		parts := strings.Split(path, "/")
		if len(parts) != 3 || parts[0] != "templates" {
			return fmt.Errorf("renderer: unexpected template path %q", path)
		}
		locale := parts[1]
		filename := strings.TrimSuffix(parts[2], ".tmpl")
		dot := strings.LastIndex(filename, ".")
		if dot < 0 {
			return fmt.Errorf("renderer: cannot determine kind for %q", path)
		}
		name := filename[:dot]
		kind := filename[dot+1:]
		key := locale + "/" + name
		switch kind {
		case "subject":
			t, err := texttemplate.New(key).Parse(string(body))
			if err != nil {
				return fmt.Errorf("renderer: parse %s: %w", path, err)
			}
			r.subjects[key] = t
		case "txt":
			t, err := texttemplate.New(key).Parse(string(body))
			if err != nil {
				return fmt.Errorf("renderer: parse %s: %w", path, err)
			}
			r.texts[key] = t
		case "html":
			t, err := htmltemplate.New(key).Parse(string(body))
			if err != nil {
				return fmt.Errorf("renderer: parse %s: %w", path, err)
			}
			r.htmls[key] = t
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

// render produces subject + text + html for (locale, name). When the
// requested locale is missing it falls back to "en"; if "en" is also
// missing the function returns an error.
func (r *renderer) render(locale, name string, data any) (subject, text, html string, err error) {
	if locale == "" {
		locale = "en"
	}
	subject, ok := r.execute(r.subjects, locale, name, data)
	if !ok {
		if subject, ok = r.execute(r.subjects, "en", name, data); !ok {
			return "", "", "", fmt.Errorf("renderer: missing subject template for %q", name)
		}
	}
	text, ok = r.execute(r.texts, locale, name, data)
	if !ok {
		if text, ok = r.execute(r.texts, "en", name, data); !ok {
			return "", "", "", fmt.Errorf("renderer: missing text template for %q", name)
		}
	}
	html, ok = r.executeHTML(r.htmls, locale, name, data)
	if !ok {
		if html, ok = r.executeHTML(r.htmls, "en", name, data); !ok {
			return "", "", "", fmt.Errorf("renderer: missing html template for %q", name)
		}
	}
	return subject, text, html, nil
}

func (r *renderer) execute(set map[string]*texttemplate.Template, locale, name string, data any) (string, bool) {
	t, ok := set[locale+"/"+name]
	if !ok {
		return "", false
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", false
	}
	return buf.String(), true
}

func (r *renderer) executeHTML(set map[string]*htmltemplate.Template, locale, name string, data any) (string, bool) {
	t, ok := set[locale+"/"+name]
	if !ok {
		return "", false
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", false
	}
	return buf.String(), true
}
