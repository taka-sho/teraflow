package graph

import (
	"fmt"
	"sort"
	"strings"
)

func (a *Analyzer) ExportMermaid() string {
	if a == nil || a.idx == nil {
		return "flowchart TD\n"
	}

	aliases := makeNodeAliases(a)
	var b strings.Builder
	b.WriteString("flowchart TD\n")

	for _, entry := range sortedEntries(a) {
		alias := aliases[entry.NodeID]
		label := entry.NodeID
		if strings.TrimSpace(entry.Title) != "" {
			label = fmt.Sprintf("%s<br/>%s", entry.NodeID, entry.Title)
		}
		fmt.Fprintf(&b, "  %s[\"%s\"]\n", alias, escapeMermaidLabel(label))
	}

	for _, entry := range sortedEntries(a) {
		for _, dep := range entry.DependsOn {
			depAlias, ok := aliases[dep]
			if !ok {
				continue
			}
			fmt.Fprintf(&b, "  %s --> %s\n", aliases[entry.NodeID], depAlias)
		}
	}

	for _, entry := range sortedEntries(a) {
		fmt.Fprintf(&b, "  class %s %s\n", aliases[entry.NodeID], mermaidClass(normalizeStatus(entry.Status)))
	}

	b.WriteString("  classDef confirmed fill:#90EE90,stroke:#333,stroke-width:1px;\n")
	b.WriteString("  classDef review fill:#87CEEB,stroke:#333,stroke-width:1px;\n")
	b.WriteString("  classDef draft fill:#FFE4B5,stroke:#333,stroke-width:1px;\n")
	b.WriteString("  classDef other fill:#D3D3D3,stroke:#333,stroke-width:1px;\n")

	return b.String()
}

func (a *Analyzer) ExportDOT() string {
	if a == nil || a.idx == nil {
		return "digraph G {}\n"
	}

	var b strings.Builder
	b.WriteString("digraph G {\n")
	b.WriteString("  rankdir=LR;\n")

	for _, entry := range sortedEntries(a) {
		label := entry.NodeID
		if strings.TrimSpace(entry.Title) != "" {
			label = fmt.Sprintf("%s\\n%s", entry.NodeID, entry.Title)
		}
		b.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\", style=filled, fillcolor=\"%s\"];\n",
			escapeDOT(entry.NodeID), escapeDOT(label), dotColor(normalizeStatus(entry.Status))))
	}

	nodes := make(map[string]struct{}, len(a.idx.Entries))
	for _, entry := range a.idx.Entries {
		nodes[entry.NodeID] = struct{}{}
	}

	for _, entry := range sortedEntries(a) {
		for _, dep := range entry.DependsOn {
			if _, ok := nodes[dep]; !ok {
				continue
			}
			b.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", escapeDOT(entry.NodeID), escapeDOT(dep)))
		}
	}

	b.WriteString("}\n")
	return b.String()
}

func sortedEntries(a *Analyzer) []struct {
	NodeID    string
	Title     string
	DependsOn []string
	Status    string
} {
	entries := make([]struct {
		NodeID    string
		Title     string
		DependsOn []string
		Status    string
	}, 0, len(a.idx.Entries))
	for _, entry := range a.idx.Entries {
		entries = append(entries, struct {
			NodeID    string
			Title     string
			DependsOn []string
			Status    string
		}{
			NodeID:    entry.NodeID,
			Title:     entry.Title,
			DependsOn: entry.DependsOn,
			Status:    entry.Status,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].NodeID < entries[j].NodeID
	})
	return entries
}

func makeNodeAliases(a *Analyzer) map[string]string {
	aliases := make(map[string]string, len(a.idx.Entries))
	used := make(map[string]int, len(a.idx.Entries))
	for _, entry := range sortedEntries(a) {
		base := sanitizeID(entry.NodeID)
		alias := base
		if n := used[base]; n > 0 {
			alias = fmt.Sprintf("%s_%d", base, n)
		}
		used[base]++
		aliases[entry.NodeID] = alias
	}
	return aliases
}

func sanitizeID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "n"
	}
	var b strings.Builder
	b.WriteByte('n')
	b.WriteByte('_')
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	return b.String()
}

func mermaidClass(status string) string {
	switch status {
	case "confirmed":
		return "confirmed"
	case "review":
		return "review"
	case "draft":
		return "draft"
	default:
		return "other"
	}
}

func dotColor(status string) string {
	switch status {
	case "confirmed":
		return "#90EE90"
	case "review":
		return "#87CEEB"
	case "draft":
		return "#FFE4B5"
	default:
		return "#D3D3D3"
	}
}

func escapeMermaidLabel(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}

func escapeDOT(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}
