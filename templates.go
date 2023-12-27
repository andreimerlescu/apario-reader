package main

import (
	`fmt`
	`html/template`
	`strings`

	`github.com/gin-gonic/gin`
)

// render_partial_template is a function defined for gin html templates that allows you to render a file from the
// bundled/assets/templates/*.html where * is the first argument of the function call
//   {{ render_partial "head" .dark_mode }}
// the .dark_mode is a second required parameter for all templates, regardless if the template component actually
// uses a dark mode switcher like
//   {{ if eq .dark_mode "1" }}dark{{else}}light{{end}}
// this special function will use the gin_func_vars type to store a list of special variables
// that need to be accessible from within the partial template. When writing to the gin_func_vars method, it is
// critical that you use the mu_gin_func_vars to .RLock(), .RUnlock(), .Lock() and .Unlock() the mutex before
// reading and writing to the gin_func_vars map for threadsafety. These variables defined can then be used
// within the template called sample you'd define myvar as
//   {{ .myvar }}
// when defining .myvar you would do something like:
//   mu_gin_func_vars.RLock()
//   existing_vars, have_vars := gin_func_vars['sample']
//   mu_gin_func_vars.RUnlock()
//   if !have_vars || len(existing_vars) == 0 {
//     mu_gin_func_vars.Lock()
//     gin_func_map['sample'] = gin.H{
//       "myvar":     "hello world",
//     }
//     mu_gin_func_vars.Unlock()
//   }
// the best place to define this would actually be inside the NewWebServer method since it has a sync.Once called
// once_server_start that ensure that you're defining and modifying the gin_func_vars map on every page request.
// For example, from within the NewWebServer method, where new routes are defined:
//   r.GET("/new-route", getMyNewRoute)
// you would define the template variables used for /new-route below this line (or above it). Consider having one
// section where these are set to prevent excessive lock/unlock on the mutex.
func render_partial_template(path string, dark_mode string) template.HTML {
	body, err := compile_partial_template(path, dark_mode)
	if err != nil {
		return template.HTML("Error: " + err.Error())
	}
	return body
}

func compile_partial_template(path string, dark_mode string) (template.HTML, error) {
	filename := fmt.Sprintf("bundled/assets/templates/%v.html", path)
	data, bundle_err := bundled_files.ReadFile(filename)
	if bundle_err != nil {
		return "", fmt.Errorf("failed to load %v due to err %v", filename, bundle_err)
	}

	tmpl := template.Must(template.New(path).Funcs(gin_func_map).Parse(string(data)))

	mu_gin_func_vars.RLock()
	existing_vars, have_vars := gin_func_vars[path]
	mu_gin_func_vars.RUnlock()
	if !have_vars || len(existing_vars) == 0 {
		existing_vars = gin.H{
			"title":        *flag_s_site_title,
			"company":      *flag_s_site_company,
			"domain":       *flag_s_primary_domain,
			"is_dark_mode": dark_mode,
		}
	}

	existing_vars["is_dark_mode"] = dark_mode // override with argument value

	var htmlBuilder strings.Builder

	mu_gin_func_vars.RLock()
	template_err := tmpl.Execute(&htmlBuilder, existing_vars)
	mu_gin_func_vars.RUnlock()

	if template_err != nil {
		return "", fmt.Errorf("error executing template: %v", template_err)
	}
	return template.HTML(htmlBuilder.String()), nil
}
