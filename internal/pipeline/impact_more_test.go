package pipeline

import (
	"context"
	"testing"

	"github.com/taka-sho/teraflow/internal/index"
)

func TestImpactAnalyzerNilIndex(t *testing.T) {
	_, err := (&ImpactAnalyzer{}).AnalyzeWithChangeType(context.Background(), "x", 1, "modify")
	if err == nil {
		t.Fatal("nil index should error")
	}
}

func TestImpactAnalyzerEmptyNodeID(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{{NodeID: "a"}}}
	analyzer := NewImpactAnalyzer(idx, nil, "/tmp")
	_, err := analyzer.AnalyzeWithChangeType(context.Background(), "", 1, "modify")
	if err == nil {
		t.Fatal("empty node id should error")
	}
}

func TestImpactAnalyzerInvalidDepth(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{{NodeID: "a"}}}
	analyzer := NewImpactAnalyzer(idx, nil, "/tmp")
	_, err := analyzer.AnalyzeWithChangeType(context.Background(), "a", 0, "modify")
	if err == nil {
		t.Fatal("depth < 1 should error")
	}
}

func TestImpactAnalyzerNodeNotFound(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{{NodeID: "a"}}}
	analyzer := NewImpactAnalyzer(idx, nil, "/tmp")
	_, err := analyzer.AnalyzeWithChangeType(context.Background(), "nonexistent", 1, "modify")
	if err == nil {
		t.Fatal("nonexistent node should error")
	}
}

func TestNormalizeChangeType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"add", "add"},
		{"added", "add"},
		{"create", "add"},
		{"created", "add"},
		{"delete", "delete"},
		{"deleted", "delete"},
		{"remove", "delete"},
		{"removed", "delete"},
		{"modify", "modify"},
		{"update", "modify"},
		{"", "modify"},
		{"ADDED", "add"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := normalizeChangeType(tt.input); got != tt.want {
				t.Fatalf("normalizeChangeType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestClassifyBandByDistance(t *testing.T) {
	tests := []struct {
		name       string
		distance   int
		changeType string
		want       string
	}{
		{"modify d=1 gray", 1, "modify", "gray"},
		{"modify d=2 amber", 2, "modify", "amber"},
		{"modify d=4 green", 4, "modify", "green"},
		{"delete d=1 gray", 1, "delete", "gray"},
		{"delete d=2 gray", 2, "delete", "gray"},
		{"delete d=3 amber", 3, "delete", "amber"},
		{"add d=1 amber", 1, "add", "amber"},
		{"add d=2 amber", 2, "add", "amber"},
		{"add d=3 green", 3, "add", "green"},
		{"zero distance treated as 1", 0, "modify", "gray"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyBandByDistance(tt.distance, tt.changeType)
			if got != tt.want {
				t.Fatalf("classifyBandByDistance(%d, %q) = %q, want %q", tt.distance, tt.changeType, got, tt.want)
			}
		})
	}
}

func TestInferChangeType(t *testing.T) {
	tests := []struct {
		name  string
		entry *index.Entry
		want  string
	}{
		{"nil entry", nil, "modify"},
		{"deleted status", &index.Entry{Status: "deleted"}, "delete"},
		{"removed status", &index.Entry{Status: "removed"}, "delete"},
		{"draft status", &index.Entry{Status: "draft"}, "add"},
		{"proposed status", &index.Entry{Status: "proposed"}, "add"},
		{"new status", &index.Entry{Status: "new"}, "add"},
		{"confirmed status", &index.Entry{Status: "confirmed"}, "modify"},
		{"empty status", &index.Entry{Status: ""}, "modify"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferChangeType(tt.entry)
			if got != tt.want {
				t.Fatalf("inferChangeType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseBridgeImpact(t *testing.T) {
	data := map[string]any{
		"affected_nodes": []any{
			map[string]any{
				"node_id":   "design:x",
				"depth":     float64(2),
				"source":    "graphrag",
				"edge_type": "depends_on",
			},
			map[string]any{
				"node_id": "impl:y",
				"depth":   float64(1),
			},
			map[string]any{}, // empty, should be skipped
			map[string]any{"node_id": ""}, // empty id, should be skipped
		},
	}
	result := parseBridgeImpact(data, "modify")
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0].NodeID != "design:x" {
		t.Fatalf("first node = %q", result[0].NodeID)
	}
	if result[0].Reason != "edge: depends_on" {
		t.Fatalf("reason = %q", result[0].Reason)
	}
	if result[1].Source != "graphrag" {
		t.Fatalf("source = %q, want graphrag", result[1].Source)
	}
}

func TestParseBridgeImpactNilData(t *testing.T) {
	result := parseBridgeImpact(nil, "modify")
	if len(result) != 0 {
		t.Fatalf("nil data should return empty, got %d", len(result))
	}
}

func TestParseBridgeImpactWithChangeType(t *testing.T) {
	data := map[string]any{
		"affected_nodes": []any{
			map[string]any{
				"node_id":     "design:x",
				"depth":       float64(1),
				"change_type": "delete",
			},
		},
	}
	result := parseBridgeImpact(data, "modify")
	if len(result) != 1 {
		t.Fatal("should have 1 result")
	}
	// delete at distance 1 = gray (1.4/1 = 1.4 >= 0.7)
	if result[0].Band != "gray" {
		t.Fatalf("band = %q, want gray", result[0].Band)
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name string
		raw  any
		want int
	}{
		{"int", 42, 42},
		{"int64", int64(99), 99},
		{"float64", float64(3.7), 3},
		{"string", "x", 0},
		{"nil", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toInt(tt.raw); got != tt.want {
				t.Fatalf("toInt(%v) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestImpactAnalyzerAddChangeType(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{NodeID: "req:a", Status: "draft"},
		{NodeID: "design:b", DependsOn: []string{"req:a"}},
	}}
	analyzer := NewImpactAnalyzer(idx, nil, t.TempDir())
	result, err := analyzer.Analyze(context.Background(), "req:a", 2)
	if err != nil {
		t.Fatal(err)
	}
	if result.ChangeType != "add" {
		t.Fatalf("change_type = %q, want add", result.ChangeType)
	}
}

func TestImpactAnalyzerNoAffectedNodes(t *testing.T) {
	idx := &index.Index{Entries: []index.Entry{
		{NodeID: "standalone"},
	}}
	analyzer := NewImpactAnalyzer(idx, nil, t.TempDir())
	result, err := analyzer.Analyze(context.Background(), "standalone", 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary["total"] != 0 {
		t.Fatalf("total = %d, want 0", result.Summary["total"])
	}
}
