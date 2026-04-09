"""Tests for CoDD graph loader."""
from __future__ import annotations

from teraflow_graphrag.graph.codd import load_codd_graph
from teraflow_graphrag.graph.schema import EdgeType, NodeType


INDEX_YAML = """
entries:
  - node_id: doc-001
    title: 認証設計書
    path: docs/auth.md
    status: confirmed
    content_hash: abc123
    depends_on:
      - doc-002
      - doc-003
  - node_id: doc-002
    title: セキュリティ方針
    path: docs/security.md
    status: confirmed
    content_hash: def456
    depends_on: []
  - node_id: doc-003
    title: ユーザー管理
    path: docs/user.md
    status: draft
    content_hash: ghi789
"""


def test_load_codd_graph(tmp_path):
    index_path = tmp_path / "index.yml"
    index_path.write_text(INDEX_YAML)

    nodes, edges = load_codd_graph(index_path)

    node_ids = {n.node_id for n in nodes}
    assert "doc-001" in node_ids
    assert "doc-002" in node_ids
    assert "doc-003" in node_ids
    assert all(n.node_type == NodeType.DOCUMENT for n in nodes)

    assert len(edges) == 2
    edge_pairs = {(e.source_id, e.target_id) for e in edges}
    assert ("doc-001", "doc-002") in edge_pairs
    assert ("doc-001", "doc-003") in edge_pairs
    assert all(e.edge_type == EdgeType.DEPENDS_ON for e in edges)


def test_load_codd_graph_missing_file(tmp_path):
    nodes, edges = load_codd_graph(tmp_path / "nonexistent.yml")
    assert nodes == []
    assert edges == []


def test_load_codd_graph_properties(tmp_path):
    index_path = tmp_path / "index.yml"
    index_path.write_text(INDEX_YAML)

    nodes, _ = load_codd_graph(index_path)
    doc001 = next(n for n in nodes if n.node_id == "doc-001")
    assert doc001.label == "認証設計書"
    assert doc001.properties["status"] == "confirmed"
    assert doc001.properties["path"] == "docs/auth.md"
