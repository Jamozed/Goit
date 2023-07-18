package goit

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/Jamozed/Goit/res"
)

var htmlError *template.Template = template.Must(template.New("error").Parse(res.Error))

func HttpError(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
	s := fmt.Sprint(code) + " " + http.StatusText(code)
	htmlError.Execute(w, struct{ Status string }{s})
}

func HttpNotFound(w http.ResponseWriter, r *http.Request) {
	HttpError(w, http.StatusNotFound)
}
