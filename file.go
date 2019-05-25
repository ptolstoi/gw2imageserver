package main

import "time"

type file struct {
	file         string
	content      []byte
	fileType     string
	lastModified time.Time
}

func fetchFile(file string) (*file, error) {
	// https://render.guildwars2.com/file/BFD2CB5A0604A4425DF9CD22DF0F40C4E0AE9AAA/602790.jpg
	// https://render.guildwars2.com/file/BFD2CB5A0604A4425DF9CD22DF0F40C4E0AE9AAA/602790.png
	// http://assetcdn.101.arenanetworks.com/program/101/1/0/602790

	return nil, nil
}
