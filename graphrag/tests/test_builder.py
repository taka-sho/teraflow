"""Tests for GraphBuilder."""
from __future__ import annotations

from teraflow_graphrag.graph.builder import GraphBuilder
from teraflow_graphrag.graph.schema import Edge, EdgeType, Node, NodeType


INDEX_YAML = """
entries:
  - node_id: doc-a
    title: Doc A
    path: docs/a.md
    status: confirmed
    content_hash: hash-a
    depends_on:
      - doc-b
  - node_id: doc-b
    title: Doc B
    path: docs/b.md
    status: draft
    content_hash: hash-b
"""


def test_builder_load_codd(tmp_path):
    index_path = tmp_path / "index.yml"
    index_path.write_text(INDEX_YAML)
    output_dir = tmp_path / "graphrag"

    builder = GraphBuilder(project_root=tmp_path, output_dir=output_dir)
    builder.load_codd(index_path=index_path)

    assert builder.graph.has_node("doc-a")
    assert builder.graph.has_node("doc-b")
    assert builder.graph.has_edge("doc-a", "doc-b")


def test_builder_add_extracted(tmp_path):
    output_dir = tmp_path / "graphrag"
    builder = GraphBuilder(project_root=tmp_path, output_dir=output_dir)

    entities = [
        Node(node_id="auth-svc", node_type=NodeType.ENTITY, label="認証サービス"),
    ]
    edges = [
        Edge(source_id="doc-a", target_id="auth-svc", edge_type=EdgeType.MENTIONS),
    ]
    builder.add_extracted(entities, edges, source_node_id="doc-a")

    assert builder.graph.has_node("auth-svc")
    assert builder.graph.has_edge("doc-a", "auth-svc")


def test_builder_needs_update(tmp_path):
    output_dir = tmp_path / "graphrag"
    builder = GraphBuilder(project_root=tmp_path, output_dir=output_dir)

    assert builder.needs_update("doc-a", "content") is True
    builder.mark_processed("doc-a", "content")
    assert builder.needs_update("doc-a", "content") is False
    assert builder.needs_update("doc-a", "changed content") is True


def test_builder_save_and_reload(tmp_path):
    index_path = tmp_path / "index.yml"
    index_path.write_text(INDEX_YAML)
    output_dir = tmp_path / "graphrag"

    builder = GraphBuilder(project_root=tmp_path, output_dir=output_dir)
    builder.load_codd(index_path=index_path)
    builder.save()

    builder2 = GraphBuilder(project_root=tmp_path, output_dir=output_dir)
    assert builder2.graph.number_of_nodes() == builder.graph.number_of_nodes()
    assert builder2.graph.number_of_edges() == builder.graph.number_of_edges()


def test_builder_no_duplicate_nodes(tmp_path):
    output_dir = tmp_path / "graphrag"
    builder = GraphBuilder(project_root=tmp_path, output_dir=output_dir)

    node = Node(node_id="dup-node", node_type=NodeType.ENTITY, label="重複テスト")
    builder.add_extracted([node], [], source_node_id="doc-a")
    builder.add_extracted([node], [], source_node_id="doc-a")

    assert builder.graph.number_of_nodes() == 1
