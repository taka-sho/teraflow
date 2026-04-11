"""Tests for subprocess entrypoint (__main__.py)."""

from __future__ import annotations

import io
import json
import sys
from pathlib import Path

import networkx as nx

from teraflow_graphrag.__main__ import main
from teraflow_graphrag.query.engine import SearchResult


def _run_main(payload: dict, monkeypatch) -> tuple[int, dict]:
    stdin = io.StringIO(json.dumps(payload))
    stdout = io.StringIO()

    monkeypatch.setattr(sys, "stdin", stdin)
    monkeypatch.setattr(sys, "stdout", stdout)

    code = 0
    try:
        main()
    except SystemExit as e:  # error paths call sys.exit(1)
        code = int(e.code)

    stdout.seek(0)
    return code, json.loads(stdout.read())


def _write_graph(path: Path) -> None:
    g = nx.DiGraph()
    g.add_node("req-auth", node_type="Document", label="Auth Requirement", status="confirmed")
    g.add_node("design-auth", node_type="Document", label="Auth Design", status="draft")
    g.add_node("auth-concept", node_type="Concept", label="Authentication")
    g.add_edge("req-auth", "design-auth", edge_type="DEPENDS_ON")
    g.add_edge("req-auth", "auth-concept", edge_type="MENTIONS")
    path.parent.mkdir(parents=True, exist_ok=True)
    nx.write_graphml(g, str(path))


def test_query_command_local(tmp_path, monkeypatch):
    graph_path = tmp_path / "graph.graphml"
    _write_graph(graph_path)

    def fake_search(self, query, top_k=5):
        return SearchResult(query=query, mode="local", answer="local ok", sources=[{"node_id": "req-auth"}])

    monkeypatch.setattr("teraflow_graphrag.query.search.LocalSearch.search", fake_search)

    code, data = _run_main(
        {
            "command": "query",
            "query": "auth",
            "mode": "local",
            "graph_path": str(graph_path),
            "storage_path": str(tmp_path),
        },
        monkeypatch,
    )

    assert code == 0
    assert data["mode"] == "local"
    assert data["answer"] == "local ok"


def test_query_command_global(tmp_path, monkeypatch):
    graph_path = tmp_path / "graph.graphml"
    _write_graph(graph_path)

    summaries_dir = tmp_path / "summaries"
    summaries_dir.mkdir(parents=True)
    (summaries_dir / "communities.json").write_text(
        json.dumps({"0": {"summary": "auth community"}}),
        encoding="utf-8",
    )

    def fake_search(self, query, top_k=5):
        return SearchResult(query=query, mode="global", answer="global ok", sources=[{"community_id": "0"}])

    monkeypatch.setattr("teraflow_graphrag.query.search.GlobalSearch.search", fake_search)

    code, data = _run_main(
        {
            "command": "query",
            "query": "auth",
            "mode": "global",
            "graph_path": str(graph_path),
            "storage_path": str(tmp_path),
        },
        monkeypatch,
    )

    assert code == 0
    assert data["mode"] == "global"
    assert data["answer"] == "global ok"


def test_impact_command(tmp_path, monkeypatch):
    graph_path = tmp_path / "graph.graphml"
    _write_graph(graph_path)

    code, data = _run_main(
        {
            "command": "impact",
            "node_id": "design-auth",
            "depth": 2,
            "include_graphrag": True,
            "graph_path": str(graph_path),
        },
        monkeypatch,
    )

    assert code == 0
    assert data["root_node_id"] == "design-auth"
    assert data["total_count"] >= 1


def test_check_command(tmp_path, monkeypatch):
    graph_path = tmp_path / "graph.graphml"
    _write_graph(graph_path)

    code, data = _run_main(
        {
            "command": "check",
            "graph_path": str(graph_path),
        },
        monkeypatch,
    )

    assert code == 0
    assert "ok" in data
    assert "issues" in data


def test_unknown_command(monkeypatch):
    code, data = _run_main({"command": "unknown-cmd"}, monkeypatch)

    assert code == 1
    assert "error" in data
    assert data["code"] == "unknown_command"
