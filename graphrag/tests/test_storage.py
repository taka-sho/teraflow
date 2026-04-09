"""Tests for storage modules."""

import networkx as nx
import pytest

from teraflow_graphrag.storage.graphml import load_graph, save_graph
from teraflow_graphrag.storage.state import DocumentState, GraphState, load_state, save_state


def test_save_and_load_graph(tmp_path):
    g = nx.DiGraph()
    g.add_node("n1", label="Node1")
    g.add_edge("n1", "n2", edge_type="DEPENDS_ON")
    save_graph(g, tmp_path)
    loaded = load_graph(tmp_path)
    assert loaded.number_of_nodes() == g.number_of_nodes()
    assert loaded.number_of_edges() == g.number_of_edges()


def test_load_graph_missing(tmp_path):
    g = load_graph(tmp_path)
    assert g.number_of_nodes() == 0


def test_save_and_load_state(tmp_path):
    state = GraphState(
        documents={"doc-1": DocumentState(content_hash="abc123", node_count=3)},
        total_edge_count=5,
        community_edge_count=4,
    )
    save_state(state, tmp_path)
    loaded = load_state(tmp_path)
    assert loaded.total_edge_count == 5
    assert loaded.documents["doc-1"].content_hash == "abc123"


def test_load_state_missing(tmp_path):
    state = load_state(tmp_path)
    assert state.total_edge_count == 0
    assert len(state.documents) == 0


def test_edge_change_rate():
    state = GraphState(total_edge_count=11, community_edge_count=10)
    assert state.edge_change_rate == pytest.approx(0.1)

    state_zero = GraphState(total_edge_count=5, community_edge_count=0)
    assert state_zero.edge_change_rate == 1.0
