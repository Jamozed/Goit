package goit

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/Jamozed/Goit/src/util"
)

/*
	Version 1 Table Schemas

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		name_full TEXT NOT NULL,
		pass BLOB NOT NULL,
		pass_algo TEXT NOT NULL,
		salt BLOB NOT NULL,
		is_admin BOOLEAN NOT NULL
	)

	CREATE TABLE IF NOT EXISTS repos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		owner_id INTEGER NOT NULL,
		name TEXT UNIQUE NOT NULL,
		name_lower TEXT UNIQUE NOT NULL,
		description TEXT NOT NULL,
		upstream TEXT NOT NULL,
		is_private BOOLEAN NOT NULL,
		is_mirror BOOLEAN NOT NULL
	)
*/

func dbUpdate(db *sql.DB) error {
	latestVersion := 3

	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}

	if version > latestVersion {
		return fmt.Errorf("database version is newer than supported (%d > %d)", version, latestVersion)
	}

	if version == 0 {
		/* Database is empty or new, initialise the newest version */
		log.Println("Initialising database at version", latestVersion)

		if _, err := db.Exec(
			`CREATE TABLE IF NOT EXISTS users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT UNIQUE NOT NULL,
				name_full TEXT NOT NULL,
				pass BLOB NOT NULL,
				pass_algo TEXT NOT NULL,
				salt BLOB NOT NULL,
				is_admin BOOLEAN NOT NULL
			)`,
		); err != nil {
			return err
		}

		if _, err := db.Exec(
			`CREATE TABLE IF NOT EXISTS repos (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				owner_id INTEGER NOT NULL,
				name TEXT UNIQUE NOT NULL,
				name_lower TEXT UNIQUE NOT NULL,
				description TEXT NOT NULL,
				default_branch TEXT NOT NULL,
				upstream TEXT NOT NULL,
				visibility INTEGER NOT NULL,
				is_mirror BOOLEAN NOT NULL
			)`,
		); err != nil {
			return err
		}

		if _, err := db.Exec(fmt.Sprint("PRAGMA user_version = ", latestVersion)); err != nil {
			return err
		}
	}

	for {
		switch version {
		case 1: /* 1 -> 2 */
			log.Println("Migrating database from version 1 to 2")

			if _, err := db.Exec(
				"ALTER TABLE repos ADD COLUMN default_branch TEXT NOT NULL DEFAULT 'master'",
			); err != nil {
				return err
			}

			version = 2

		case 2: /* 2 -> 3 */
			log.Println("Migrating database from version 2 to 3")

			if _, err := db.Exec(
				"ALTER TABLE repos ADD COLUMN visibility INTEGER NOT NULL DEFAULT 0",
			); err != nil {
				return err
			}

			/* Set values for each repo according to is_private */
			var visibilities = map[int64]Visibility{}

			if rows, err := db.Query("SELECT id, is_private FROM repos"); err != nil {
				return err
			} else {
				for rows.Next() {
					var id int64
					var isPrivate bool

					if err := rows.Scan(&id, &isPrivate); err != nil {
						return err
					}

					visibilities[id] = util.If(isPrivate, Private, Public)
				}

				rows.Close()
			}

			for id, visibility := range visibilities {
				if _, err := db.Exec(
					"UPDATE repos SET visibility = ? WHERE id = ?", visibility, id,
				); err != nil {
					return err
				}
			}

			/* Remove is_private column */
			if _, err := db.Exec("ALTER TABLE repos DROP COLUMN is_private"); err != nil {
				return err
			}

			version = 3

		default: /* No required migrations */
			goto done
		}
	}

done:
	if _, err := db.Exec(fmt.Sprint("PRAGMA user_version = ", version)); err != nil {
		return err
	}

	return nil
}
