package gw2imageserver

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func (app *app) initDB() error {
	db, err := sql.Open("sqlite3", "./cache.db")
	if err != nil {
		return err
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
		return err
	}

	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS 
			raw_file_fileType
		ON 
			raw
		(
			file,
			fileType
		)
	`)

	if err != nil {
		return err
	}

	app.db = db

	return nil
}

func (app *app) getFileFromCache(fileToLookup string, fileTypeToLookup string) (*file, error) {
	log.Printf("[getFileFromCache] fileToLookup=%v type=%v", fileToLookup, fileTypeToLookup)

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

	err := row.Scan(
		&file.file,
		&lastModified,
		&file.fileType,
		&file.content,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	} else if err == sql.ErrNoRows {
		log.Printf("[getFileFromCache] not found")
		return nil, nil
	}

	file.lastModified, err = time.Parse(time.RFC1123Z, lastModified)
	if err != nil {
		return nil, err
	}

	return &file, nil
}

func (app *app) saveFileToCache(file *file) error {
	log.Printf("[saveFileToCache] file=%v type=%v size=%v time=%v", file.file, file.fileType, len(file.content), file.lastModified)

	lastModified := file.lastModified.Format(time.RFC1123Z)

	_, err := app.db.Exec(`
		INSERT OR REPLACE INTO
			raw
				(
					file, lastModified, fileType, content
				)
		VALUES
				(?, ?, ?, ?)
	`, file.file, lastModified, file.fileType, file.content)

	return err
}

func (app *app) closeDB() {
	if err := app.db.Close(); err != nil {
		log.Printf("[closeDB] %v", err)
	}
}
