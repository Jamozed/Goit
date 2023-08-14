package repo

import (
	"log"
	"net/http"

	goit "github.com/Jamozed/Goit/src"
	"github.com/gorilla/mux"
)

func HandleEdit(w http.ResponseWriter, r *http.Request) {
	auth, uid := goit.AuthCookie(w, r, true)

	repo, err := goit.GetRepoByName(mux.Vars(r)["repo"])
	if err != nil {
		log.Println("[/repo/edit]", err.Error())
		goit.HttpError(w, http.StatusInternalServerError)
		return
	} else if repo == nil || (!auth || repo.OwnerId != uid) {
		goit.HttpError(w, http.StatusNotFound)
		return
	}

	// data := struct {
	// 	Title string
	// }{
	// 	Title: "Repository - Edit",
	// }
}
