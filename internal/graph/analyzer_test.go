package graph

import (
	"strings"
	"testing"
	"time"

	"github.com/taka-sho/teraflow/internal/index"
)

func TestStatusAggregates(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{NodeID: "A", Title: "A", Status: "confirmed", Tags: []string{"core", "api"}, DependsOn: []string{"B"}, UpdatedAt: time.Now()},
		{NodeID: "B", Title: "B", Status: "review", Tags: []string{"core"}, DependsOn: []string{"C"}, UpdatedAt: time.Now()},
		{NodeID: "C", Title: "C", Status: "draft", Tags: []string{"ux"}, DependsOn: nil, UpdatedAt: time.Now()},
		{NodeID: "D", Title: "D", Status: "confirmed", Tags: []string{"ops"}, DependsOn: nil, UpdatedAt: time.Now()},
		{NodeID: "E", Title: "E", Status: "", Tags: nil, DependsOn: []string{"A"}, UpdatedAt: time.Now()},
	}}

	res := NewAnalyzer(idx).Status()
	if res.TotalNodes != 5 {
		t.Fatalf("TotalNodes = %d, want 5", res.TotalNodes)
	}
	if res.TotalEdges != 3 {
		t.Fatalf("TotalEdges = %d, want 3", res.TotalEdges)
	}
	if res.IsolatedNodes != 1 {
		t.Fatalf("IsolatedNodes = %d, want 1", res.IsolatedNodes)
	}
	if res.ByStatus["confirmed"] != 2 || res.ByStatus["review"] != 1 || res.ByStatus["draft"] != 1 || res.ByStatus["unknown"] != 1 {
		t.Fatalf("ByStatus unexpected: %#v", res.ByStatus)
	}
	if res.ByTag["core"] != 2 || res.ByTag["api"] != 1 || res.ByTag["ux"] != 1 || res.ByTag["ops"] != 1 {
		t.Fatalf("ByTag unexpected: %#v", res.ByTag)
	}
}

func TestCheckDetectsBrokenRef(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{NodeID: "A", Status: "draft", DependsOn: []string{"MISSING"}},
	}}

	res := NewAnalyzer(idx).Check()
	if res.OK {
		t.Fatal("Check OK = true, want false")
	}
	if !hasIssue(res.Issues, "broken_ref", SeverityError) {
		t.Fatalf("broken_ref issue not found: %#v", res.Issues)
	}
}

func TestCheckDetectsStatusConflict(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{NodeID: "A", Status: "confirmed", DependsOn: []string{"B"}},
		{NodeID: "B", Status: "draft"},
	}}

	res := NewAnalyzer(idx).Check()
	if !hasIssue(res.Issues, "status_conflict", SeverityWarning) {
		t.Fatalf("status_conflict warning not found: %#v", res.Issues)
	}
}

func TestCheckDetectsCyclicDep(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{NodeID: "A", DependsOn: []string{"B"}},
		{NodeID: "B", DependsOn: []string{"C"}},
		{NodeID: "C", DependsOn: []string{"A"}},
	}}

	res := NewAnalyzer(idx).Check()
	if res.OK {
		t.Fatal("Check OK = true, want false")
	}
	if !hasIssue(res.Issues, "cyclic_dep", SeverityError) {
		t.Fatalf("cyclic_dep issue not found: %#v", res.Issues)
	}
}

func TestExportMermaidContainsAllNodes(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{NodeID: "A", Title: "Alpha", Status: "confirmed", DependsOn: []string{"B"}},
		{NodeID: "B", Title: "Beta", Status: "review"},
		{NodeID: "C", Title: "Gamma", Status: "draft"},
	}}

	out := NewAnalyzer(idx).ExportMermaid()
	if !strings.Contains(out, "flowchart TD") {
		t.Fatalf("missing flowchart header: %q", out)
	}
	for _, nodeID := range []string{"A", "B", "C"} {
		if !strings.Contains(out, nodeID) {
			t.Fatalf("mermaid missing node %s: %q", nodeID, out)
		}
	}
}

func TestExportDOTContainsAllNodes(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{NodeID: "A", Title: "Alpha", Status: "confirmed", DependsOn: []string{"B"}},
		{NodeID: "B", Title: "Beta", Status: "review"},
		{NodeID: "C", Title: "Gamma", Status: "draft"},
	}}

	out := NewAnalyzer(idx).ExportDOT()
	if !strings.Contains(out, "digraph G") {
		t.Fatalf("missing digraph header: %q", out)
	}
	for _, nodeID := range []string{"A", "B", "C"} {
		if !strings.Contains(out, nodeID) {
			t.Fatalf("dot missing node %s: %q", nodeID, out)
		}
	}
}

func TestEmptyIndex(t *testing.T) {
	res := NewAnalyzer(&index.Index{}).Status()
	if res.TotalNodes != 0 || res.TotalEdges != 0 || res.IsolatedNodes != 0 {
		t.Fatalf("unexpected status: %#v", res)
	}
	check := NewAnalyzer(&index.Index{}).Check()
	if !check.OK || len(check.Issues) != 0 {
		t.Fatalf("unexpected check result: %#v", check)
	}
}

func hasIssue(issues []Issue, typ string, sev Severity) bool {
	for _, issue := range issues {
		if issue.Type == typ && issue.Severity == sev {
			return true
		}
	}
	return false
}
