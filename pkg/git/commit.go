package git

import (
	"time"
)

// A Commit contains the commit hash, date, and content (in base64) of a specific git commit
type Commit struct {
	Hash    string
	Date    time.Time
	Content string
}
