package templates

import "embed"

//go:embed issues/*.yml
var IssueFS embed.FS

//go:embed discussions/requirements.yml discussions/design.yml discussions/retrospective.yml
var DiscussionFS embed.FS

//go:embed discussions/categories.yml
var DiscussionCategoryFS embed.FS
