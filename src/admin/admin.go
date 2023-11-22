package admin

import (
	"log"
	"net/http"

	"github.com/Jamozed/Goit/src/goit"
)

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	auth, user, err := goit.Auth(w, r, true)
	if err != nil {
		log.Println("[admin]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
	}

	if !auth || !user.IsAdmin {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	if err := goit.Tmpl.ExecuteTemplate(w, "admin/index", struct{ Title string }{"Admin"}); err != nil {
		log.Println("[/admin/index]", err.Error())
	}
}
