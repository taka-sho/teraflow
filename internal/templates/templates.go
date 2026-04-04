package templates

import "embed"

//go:embed issues/*.yml
var IssueFS embed.FS

//go:embed discussions/*.yml
var DiscussionFS embed.FS
