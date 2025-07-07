package model

import (
	"encoding/base64"
	"time"
)

// ==========
// == FILE ==
// ==========

// A File that contains a list of commits (version)
type File struct {
	filename string
	filepath string
	history  []Commit
}

// Init initializes the [File] struct
func (f *File) Init(filename string, filepath string, history []Commit, components []Component) {
	f.filename = filename
	f.filepath = filepath
	f.history = history
}

// GetFilename returns the filename of a [File] struct
func (f *File) GetFilename() string {
	return f.filename
}

// GetFilepath returns the filepath of a [File] struct
func (f *File) GetFilepath() string {
	return f.filepath
}

// GetHistory returns the complete history of a [File] struct
func (f *File) GetHistory() []Commit {
	return f.history
}

// ============
// == COMMIT ==
// ============

// A Commit contains the commit hash, date, and content (in base64) of a specific git commit
type Commit struct {
	hash       string
	date       time.Time
	content    string
	components []*Component
}

// Init initializes the [Commit] struct
func (c *Commit) Init(hash string, date time.Time, content string, components []*Component) {
	c.hash = hash
	c.date = date
	c.content = base64.StdEncoding.EncodeToString([]byte(content))
	c.components = components
}

// GetHash returns the hash of a [Commit] struct
func (c *Commit) GetHash() string {
	return c.hash
}

// GetDate returns the date of a [Commit] struct
func (c *Commit) GetDate() time.Time {
	return c.date
}

// GetContent returns either the decoded or the base64 encoded content of a [Commit] struct
func (c *Commit) GetContent(decode bool) (string, error) {
	if decode {
		if decoded, err := base64.StdEncoding.DecodeString(c.content); err != nil {
			return "", err
		} else {
			return string(decoded), nil
		}
	} else {
		return c.content, nil
	}
}

// GetComponents returns the external components used in the [Commit] struct
func (c *Commit) GetComponents() []*Component {
	return c.components
}
