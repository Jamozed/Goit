package repo

import (
	"log"
	"net/http"
	"strconv"

	goit "github.com/Jamozed/Goit/src"
)

func HandleDelete(w http.ResponseWriter, r *http.Request) {
	auth, admin, uid := goit.AuthCookieAdmin(w, r, true)

	rid, err := strconv.ParseInt(r.URL.Query().Get("repo"), 10, 64)
	if err != nil {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	repo, err := goit.GetRepo(rid)
	if err != nil {
		log.Println("[/repo/delete]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	if !auth || (uid != repo.OwnerId && !admin) {
		goit.HttpError(w, http.StatusUnauthorized)
		return
	}

	if err := goit.DelRepo(repo.Name); err != nil {
		log.Println("[/repo/delete]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
