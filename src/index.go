package goit

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Jamozed/Goit/src/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	auth, admin, uid := AuthCookieAdmin(w, r, true)

	user, err := GetUser(uid)
	if err != nil {
		log.Println("[/]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	}

	type row struct{ Name, Description, Owner, Visibility, LastCommit string }
	data := struct {
		Title, Username string
		Admin, Auth     bool
		Repos           []row
	}{Title: "Repositories", Admin: admin, Auth: auth}

	if user != nil {
		data.Username = user.Name
	}

	rows, err := db.Query("SELECT id, owner_id, name, description, is_private FROM repos")
	if err != nil {
		log.Println("[/]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		repo := Repo{}

		if err := rows.Scan(&repo.Id, &repo.OwnerId, &repo.Name, &repo.Description, &repo.IsPrivate); err != nil {
			log.Println("[/]", err.Error())
		} else if !repo.IsPrivate || (auth && uid == repo.OwnerId) {
			owner, err := GetUser(repo.OwnerId)
			if err != nil {
				log.Println("[/]", err.Error())
			}

			var lastCommit string
			if gr, err := git.PlainOpen(RepoPath(repo.Name)); err != nil {
				log.Println("[/]", err.Error())
			} else if ref, err := gr.Head(); err != nil {
				if !errors.Is(err, plumbing.ErrReferenceNotFound) {
					log.Println("[/]", err.Error())
				}
			} else if commit, err := gr.CommitObject(ref.Hash()); err != nil {
				log.Println("[/]", err.Error())
			} else {
				lastCommit = commit.Author.When.UTC().Format(time.DateTime)
			}

			data.Repos = append(data.Repos, row{
				Name: repo.Name, Description: repo.Description, Owner: owner.Name,
				Visibility: util.If(repo.IsPrivate, "private", "public"), LastCommit: lastCommit,
			})
		}
	}

	if err := rows.Err(); err != nil {
		log.Println("[/]", err.Error())
		HttpError(w, http.StatusInternalServerError)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "index", data); err != nil {
		log.Println("[/]", err.Error())
	}
}