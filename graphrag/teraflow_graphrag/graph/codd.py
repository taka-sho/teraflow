"""CoDD explicit graph loader: reads index.yml and creates Document nodes + DEPENDS_ON edges."""
from __future__ import annotations

from pathlib import Path

import yaml

from teraflow_graphrag.graph.schema import Edge, EdgeType, Node, NodeType


def load_codd_graph(index_path: Path) -> tuple[list[Node], list[Edge]]:
    """Load index.yml and return Document nodes + DEPENDS_ON edges.

    Args:
        index_path: Path to index.yml (e.g. .teraflow/index.yml)

    Returns:
        (nodes, edges) where nodes are Document type, edges are DEPENDS_ON type
    """
    if not index_path.exists():
        return [], []

    with open(index_path) as f:
        raw = yaml.safe_load(f) or {}

    entries = raw.get("entries", [])
    nodes: list[Node] = []
    edges: list[Edge] = []

    for entry in entries:
        node_id = entry.get("node_id", "")
        if not node_id:
            continue
        nodes.append(
            Node(
                node_id=node_id,
                node_type=NodeType.DOCUMENT,
                label=entry.get("title", node_id),
                properties={
                    "path": entry.get("path", ""),
                    "status": entry.get("status", ""),
                    "content_hash": entry.get("content_hash", ""),
                    "tags": ",".join(entry.get("tags") or []),
                },
            )
        )
        for dep_id in (entry.get("depends_on") or []):
            if dep_id:
                edges.append(
                    Edge(
                        source_id=node_id,
                        target_id=dep_id,
                        edge_type=EdgeType.DEPENDS_ON,
                    )
                )

    return nodes, edges
