package templates

import "embed"

//go:embed issues/*.yml
var IssueFS embed.FS

//go:embed "discussions/要件議論-requirements-discussion.yml" "discussions/設計議論-design-discussion.yml" "discussions/振り返り-retrospective.yml"
var DiscussionFS embed.FS

//go:embed discussions/categories.yml
var DiscussionCategoryFS embed.FS
