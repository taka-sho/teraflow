package actions_test

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func reqAgentTemplate(t *testing.T) string {
	t.Helper()
	templates := generatedTemplates(t)
	reqAgent, ok := templates["teraflow-req-agent"]
	if !ok {
		t.Fatal("teraflow-req-agent template not generated")
	}
	return string(reqAgent)
}

func extractPyFunc(t *testing.T, tpl, funcName string) string {
	t.Helper()
	needle := "          def " + funcName + "("
	start := strings.Index(tpl, needle)
	if start == -1 {
		t.Fatalf("function not found in template: %s", funcName)
	}

	end := len(tpl)
	for _, marker := range []string{
		"\n          def ",
		"\n          parsed_answers = []",
		"\n          mark_resolved(",
	} {
		next := strings.Index(tpl[start+1:], marker)
		if next == -1 {
			continue
		}
		candidate := start + 1 + next
		if candidate < end {
			end = candidate
		}
	}

	block := tpl[start:end]
	lines := strings.Split(block, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, "          ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n")) + "\n"
}

func runDiscoveryPython(t *testing.T, code string) map[string]any {
	t.Helper()
	cmd := exec.Command("python3", "-c", code)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python failed: %v\n%s", err, string(out))
	}

	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("json decode failed: %v\noutput=%s", err, string(out))
	}
	return result
}

func mustMap(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", v)
	}
	return m
}

func TestDiscoveryParseUserAnswerPatterns(t *testing.T) {
	tpl := reqAgentTemplate(t)
	py := strings.Join([]string{
		"import json",
		"import re",
		extractPyFunc(t, tpl, "parse_user_answer"),
`cases = {
	"plain": parse_user_answer("1a 2b 3d", ""),
	"dot_slash": parse_user_answer("1. a,b / 2. a", ""),
	"hyphen": parse_user_answer("1-a 2-b", ""),
	"empty": parse_user_answer("", ""),
	"invalid": parse_user_answer("???", ""),
}
print(json.dumps(cases, ensure_ascii=False))`,
	}, "\n")

	got := runDiscoveryPython(t, py)

	plain := mustMap(t, got["plain"])
	if plain["1"] != "a" || plain["2"] != "b" || plain["3"] != "d" {
		t.Fatalf("unexpected plain parse result: %#v", plain)
	}

	dotSlash := mustMap(t, got["dot_slash"])
	if dotSlash["1"] != "a,b" || dotSlash["2"] != "a" {
		t.Fatalf("unexpected dot/slash parse result: %#v", dotSlash)
	}

	hyphen := mustMap(t, got["hyphen"])
	if hyphen["1"] != "a" || hyphen["2"] != "b" {
		t.Fatalf("unexpected hyphen parse result: %#v", hyphen)
	}

	empty := mustMap(t, got["empty"])
	if len(empty) != 0 {
		t.Fatalf("empty input should produce empty result: %#v", empty)
	}
	invalid := mustMap(t, got["invalid"])
	if len(invalid) != 0 {
		t.Fatalf("invalid input should produce empty result: %#v", invalid)
	}
}

func TestDiscoveryExtractQuestionsFromComment(t *testing.T) {
	tpl := reqAgentTemplate(t)
	py := strings.Join([]string{
		"import json",
		"import re",
		extractPyFunc(t, tpl, "infer_category"),
		extractPyFunc(t, tpl, "extract_questions_from_comment"),
		`comment = """## 🔍 要件探索
1. 対象ユーザーはどれですか？
   a) 社内利用者  b) 一般ユーザー  c) 管理者
2. 納期はどれですか？
   a) 1週間  b) 1か月
"""
out = {
	"ok": extract_questions_from_comment(comment),
	"malformed": extract_questions_from_comment("not a numbered question list"),
}
print(json.dumps(out, ensure_ascii=False))`,
	}, "\n")

	got := runDiscoveryPython(t, py)
	ok := mustMap(t, got["ok"])
	if len(ok) != 2 {
		t.Fatalf("expected 2 questions, got %#v", ok)
	}
	q1 := mustMap(t, ok["1"])
	q1Choices := mustMap(t, q1["choices"])
	if q1Choices["a"] != "社内利用者" || q1Choices["b"] != "一般ユーザー" {
		t.Fatalf("unexpected q1 choices: %#v", q1Choices)
	}
	q2 := mustMap(t, ok["2"])
	if q2["category"] != "timeline" {
		t.Fatalf("question 2 category should be timeline, got %#v", q2["category"])
	}

	malformed := mustMap(t, got["malformed"])
	if len(malformed) != 0 {
		t.Fatalf("malformed input should return empty map: %#v", malformed)
	}
}

func TestDiscoveryTwoRoundParseIntegration(t *testing.T) {
	tpl := reqAgentTemplate(t)
	py := strings.Join([]string{
		"import json",
		"import re",
		"from datetime import datetime, timezone",
		`def normalize_list(value):
    if isinstance(value, list):
        return [str(v).strip() for v in value if str(v).strip()]
    if isinstance(value, str):
        text = value.strip()
        return [text] if text else []
    return []`,
		extractPyFunc(t, tpl, "walk"),
		extractPyFunc(t, tpl, "find_node"),
		extractPyFunc(t, tpl, "infer_category"),
		extractPyFunc(t, tpl, "extract_questions_from_comment"),
		extractPyFunc(t, tpl, "parse_user_answer"),
		extractPyFunc(t, tpl, "find_recent_assistant_comment"),
		extractPyFunc(t, tpl, "find_question_node"),
		extractPyFunc(t, tpl, "stage1_parse_user_answers"),
		extractPyFunc(t, tpl, "mark_resolved_from_parse"),
		extractPyFunc(t, tpl, "simplify_question_label"),
		extractPyFunc(t, tpl, "update_confirmed_from_parse"),
		`nodes = [
    {"id": "scope.users", "question": "対象ユーザーはどれですか？", "category": "scope", "status": "pending", "answer": "", "children": []},
    {"id": "auth.method", "question": "認証方式はどれですか？", "category": "functional", "status": "pending", "answer": "", "children": []},
    {
        "id": "dialogue.1",
        "category": "dialogue",
        "status": "answered",
        "meta": {
            "assistant_comment": """## 🔍 要件探索
1. 対象ユーザーはどれですか？
   a) 社内利用者  b) 一般ユーザー
2. 認証方式はどれですか？
   a) メール認証  b) SSO
"""
        },
    },
]

round1, q1 = stage1_parse_user_answers("1a 2b", nodes)
applied1 = mark_resolved_from_parse(round1, nodes)
summary = {"confirmed": []}
confirmed1 = update_confirmed_from_parse(round1, q1, summary)

nodes.append({"id": "quality.metric", "question": "成功条件はどれですか？", "category": "quality", "status": "pending", "answer": "", "children": []})
nodes.append({
    "id": "dialogue.2",
    "category": "dialogue",
    "status": "answered",
    "meta": {
        "assistant_comment": """## 🔍 要件探索
1. 成功条件はどれですか？
   a) 納期順守  b) 品質重視
"""
    },
})

round2, q2 = stage1_parse_user_answers("1b", nodes)
applied2 = mark_resolved_from_parse(round2, nodes)
confirmed2 = update_confirmed_from_parse(round2, q2, summary)

print(json.dumps({
    "round1_applied": applied1,
    "round2_applied": applied2,
    "confirmed1": confirmed1,
    "confirmed2": confirmed2,
    "quality_node": find_node(nodes, "quality.metric"),
}, ensure_ascii=False))`,
	}, "\n")

	got := runDiscoveryPython(t, py)

	r1, ok := got["round1_applied"].([]any)
	if !ok || len(r1) != 2 {
		t.Fatalf("round1 should resolve 2 answers: %#v", got["round1_applied"])
	}

	c1, ok := got["confirmed1"].([]any)
	if !ok || len(c1) != 2 {
		t.Fatalf("round1 confirmed should contain 2 entries: %#v", got["confirmed1"])
	}

	r2, ok := got["round2_applied"].([]any)
	if !ok || len(r2) != 1 {
		t.Fatalf("round2 should resolve the new question: %#v", got["round2_applied"])
	}

	c2, ok := got["confirmed2"].([]any)
	if !ok || len(c2) != 3 {
		t.Fatalf("confirmed should accumulate across rounds: %#v", got["confirmed2"])
	}

	qualityNode := mustMap(t, got["quality_node"])
	if qualityNode["status"] != "answered" || qualityNode["answer"] != "品質重視" {
		t.Fatalf("quality node should be answered in round2: %#v", qualityNode)
	}
}
