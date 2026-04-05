package templates

import "embed"

//go:embed issues/*.yml
var IssueFS embed.FS

//go:embed discussions/01-requirements.yml discussions/02-design.yml discussions/03-retrospective.yml discussions/04-change-request.yml discussions/05-question.yml discussions/06-risk.yml discussions/07-release-planning.yml
var DiscussionFS embed.FS

//go:embed discussions/categories.yml
var DiscussionCategoryFS embed.FS
