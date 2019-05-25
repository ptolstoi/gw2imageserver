package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func (app *app) initDB() {
	db, err := sql.Open("sqlite3", "./cache.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS 
			raw
		(
			file TEXT NOT NULL,
			lastModified TEXT,
			fileType TEXT,
			content BLOB,

			CONSTRAINT file_fileType UNIQUE (file, filetype)
		)
	`)

	if err != nil {
		log.Fatal(err)
	}

	app.db = db
}

func (app *app) getFileFromCache(fileToLookup string, fileTypeToLookup string) (*file, error) {
	log.Printf("[getFileFromCache] %v %v", fileToLookup, fileTypeToLookup)

	row := app.db.QueryRow(`
	SELECT 
		file, 
		lastModified, 
		fileType,
		content 
	FROM 
		raw 
	WHERE 
		file = ? AND fileType = ?`, fileToLookup, fileTypeToLookup)

	file := file{}
	var lastModified string

	if err := row.Scan(
		&file.file,
		&lastModified,
		&file.fileType,
		&file.content,
	); err != nil && err != sql.ErrNoRows {
		return nil, err
	} else if err == sql.ErrNoRows {
		log.Printf("[getFileFromCache] not found")
		return nil, nil
	}

	log.Printf("[getFileFromCache] lastModified: %v", lastModified)

	return &file, nil
}

func (app *app) closeDB() {
	app.db.Close()
}
