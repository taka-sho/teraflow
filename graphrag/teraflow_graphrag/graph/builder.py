"""Knowledge graph builder combining CoDD explicit graph and LLM-extracted entities."""
from __future__ import annotations

import hashlib
import logging
from pathlib import Path

import networkx as nx

from teraflow_graphrag.graph.codd import load_codd_graph
from teraflow_graphrag.graph.schema import Edge, Node, NodeType
from teraflow_graphrag.storage.graphml import load_graph, save_graph
from teraflow_graphrag.storage.state import (
    DocumentState,
    GraphState,
    load_state,
    save_state,
)

logger = logging.getLogger(__name__)


class GraphBuilder:
    """Build and maintain the knowledge graph."""

    def __init__(self, project_root: Path, output_dir: Path) -> None:
        self.project_root = project_root
        self.output_dir = output_dir
        self.graph: nx.DiGraph = load_graph(output_dir)
        self.state: GraphState = load_state(output_dir)

    def load_codd(self, index_path: Path | None = None) -> None:
        """Load CoDD explicit graph from index.yml."""
        if index_path is None:
            index_path = self.project_root / ".teraflow" / "index.yml"
        nodes, edges = load_codd_graph(index_path)
        for node in nodes:
            self._add_node(node)
        for edge in edges:
            self._add_edge(edge)
        logger.info("CoDD graph loaded: %d nodes, %d edges", len(nodes), len(edges))

    def add_extracted(self, nodes: list[Node], edges: list[Edge], source_node_id: str) -> None:
        """Add LLM-extracted entities and relations for a document."""
        for node in nodes:
            self._add_node(node)
        for edge in edges:
            self._add_edge(edge)
        # Keep arg used for forward compatibility until state linkage is added.
        _ = source_node_id

    def _add_node(self, node: Node) -> None:
        if not self.graph.has_node(node.node_id):
            self.graph.add_node(
                node.node_id,
                node_type=node.node_type.value,
                label=node.label,
                **node.properties,
            )

    def _add_edge(self, edge: Edge) -> None:
        if not self.graph.has_node(edge.target_id):
            self.graph.add_node(
                edge.target_id,
                node_type=NodeType.DOCUMENT.value,
                label=edge.target_id,
            )
        if not self.graph.has_node(edge.source_id):
            self.graph.add_node(
                edge.source_id,
                node_type=NodeType.DOCUMENT.value,
                label=edge.source_id,
            )
        if not self.graph.has_edge(edge.source_id, edge.target_id):
            self.graph.add_edge(
                edge.source_id,
                edge.target_id,
                edge_type=edge.edge_type.value,
                **edge.properties,
            )

    def needs_update(self, node_id: str, content: str) -> bool:
        """Check if document needs re-extraction (content hash changed)."""
        new_hash = hashlib.sha256(content.encode()).hexdigest()
        prev = self.state.documents.get(node_id)
        return prev is None or prev.content_hash != new_hash

    def mark_processed(
        self,
        node_id: str,
        content: str,
        node_count: int = 0,
        edge_count: int = 0,
    ) -> None:
        """Record that a document has been processed."""
        content_hash = hashlib.sha256(content.encode()).hexdigest()
        self.state.documents[node_id] = DocumentState(
            content_hash=content_hash,
            node_count=node_count,
            edge_count=edge_count,
        )

    def save(self) -> None:
        """Persist graph and state to disk."""
        self.state.total_edge_count = self.graph.number_of_edges()
        save_graph(self.graph, self.output_dir)
        save_state(self.state, self.output_dir)
        logger.info(
            "Graph saved: %d nodes, %d edges",
            self.graph.number_of_nodes(),
            self.graph.number_of_edges(),
        )
