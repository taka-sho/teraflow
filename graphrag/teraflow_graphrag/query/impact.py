"""Impact analysis: traverse DEPENDS_ON + MENTIONS + RELATED_TO edges (design doc ch.3.2)."""
from __future__ import annotations

from collections import deque
from dataclasses import dataclass, field
from typing import Any

import networkx as nx

from teraflow_graphrag.graph.schema import EdgeType


@dataclass
class ImpactNode:
    node_id: str
    label: str
    node_type: str
    depth: int
    edge_type: str
    source: str  # "codd" or "graphrag" (design doc 9.4)


@dataclass
class ImpactResult:
    root_node_id: str
    affected_nodes: list[ImpactNode] = field(default_factory=list)
    total_count: int = 0

    def to_dict(self) -> dict[str, Any]:
        return {
            "root_node_id": self.root_node_id,
            "total_count": self.total_count,
            "affected_nodes": [
                {
                    "node_id": n.node_id,
                    "label": n.label,
                    "node_type": n.node_type,
                    "depth": n.depth,
                    "edge_type": n.edge_type,
                    "source": n.source,
                }
                for n in self.affected_nodes
            ],
        }


CODD_EDGE_TYPES = {EdgeType.DEPENDS_ON.value}
GRAPHRAG_EDGE_TYPES = {EdgeType.MENTIONS.value, EdgeType.RELATED_TO.value}


def analyze_impact(
    graph: nx.DiGraph,
    node_id: str,
    depth: int = 2,
    include_graphrag: bool = True,
) -> ImpactResult:
    """BFS traversal to find all nodes affected by changes to node_id."""
    if not graph.has_node(node_id):
        return ImpactResult(root_node_id=node_id, affected_nodes=[], total_count=0)

    visited: set[str] = {node_id}
    queue: deque[tuple[str, int]] = deque([(node_id, 0)])
    affected: list[ImpactNode] = []

    while queue:
        current_id, current_depth = queue.popleft()
        if current_depth >= depth:
            continue

        for pred_id in graph.predecessors(current_id):
            if pred_id in visited:
                continue
            edge_data = graph.edges[pred_id, current_id]
            edge_type = edge_data.get("edge_type", "")

            if edge_type in CODD_EDGE_TYPES:
                source = "codd"
            elif edge_type in GRAPHRAG_EDGE_TYPES and include_graphrag:
                source = "graphrag"
            else:
                continue

            visited.add(pred_id)
            node_data = graph.nodes[pred_id]
            affected.append(
                ImpactNode(
                    node_id=str(pred_id),
                    label=str(node_data.get("label", pred_id)),
                    node_type=str(node_data.get("node_type", "")),
                    depth=current_depth + 1,
                    edge_type=str(edge_type),
                    source=source,
                )
            )
            queue.append((str(pred_id), current_depth + 1))

    return ImpactResult(root_node_id=node_id, affected_nodes=affected, total_count=len(affected))
