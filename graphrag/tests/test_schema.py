"""Tests for graph schema module."""

from teraflow_graphrag.graph.schema import Edge, EdgeType, Node, NodeType


def test_node_to_dict():
    node = Node(node_id="doc-1", node_type=NodeType.DOCUMENT, label="Spec")
    d = node.to_dict()
    assert d["node_id"] == "doc-1"
    assert d["node_type"] == "Document"
    assert d["label"] == "Spec"


def test_edge_to_dict():
    edge = Edge(source_id="doc-1", target_id="doc-2", edge_type=EdgeType.DEPENDS_ON)
    d = edge.to_dict()
    assert d["source_id"] == "doc-1"
    assert d["edge_type"] == "DEPENDS_ON"


def test_node_types():
    assert NodeType.DOCUMENT.value == "Document"
    assert NodeType.ENTITY.value == "Entity"
    assert NodeType.CONCEPT.value == "Concept"
    assert NodeType.COMMUNITY.value == "Community"


def test_edge_types():
    assert EdgeType.DEPENDS_ON.value == "DEPENDS_ON"
    assert EdgeType.MENTIONS.value == "MENTIONS"
    assert EdgeType.RELATED_TO.value == "RELATED_TO"
    assert EdgeType.BELONGS_TO.value == "BELONGS_TO"
