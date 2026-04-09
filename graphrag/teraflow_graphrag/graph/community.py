"""Community detection using Leiden algorithm (graspologic)."""
from __future__ import annotations

import json
import logging
from pathlib import Path

import networkx as nx

from teraflow_graphrag.graph.schema import EdgeType, NodeType
from teraflow_graphrag.storage.state import GraphState, save_state

logger = logging.getLogger(__name__)


def detect_communities(
    graph: nx.DiGraph,
    state: GraphState,
    output_dir: Path,
    threshold: float = 0.1,
    force: bool = False,
) -> dict[str, int]:
    """Detect communities using Leiden algorithm with threshold-based re-run."""
    if not force and state.edge_change_rate <= threshold:
        logger.info(
            "Skipping community detection: edge_change_rate=%.3f <= threshold=%.3f",
            state.edge_change_rate,
            threshold,
        )
        return _load_communities(output_dir)

    logger.info(
        "Running Leiden community detection: edge_change_rate=%.3f > threshold=%.3f",
        state.edge_change_rate,
        threshold,
    )

    communities = _run_leiden(graph)
    _save_communities(communities, output_dir)

    state.community_edge_count = graph.number_of_edges()
    save_state(state, output_dir)

    return communities


def _run_leiden(graph: nx.DiGraph) -> dict[str, int]:
    """Run Leiden algorithm via graspologic with fallback on failure."""
    try:
        from graspologic.partition import leiden

        undirected = graph.to_undirected()
        if undirected.number_of_nodes() == 0:
            return {}

        result = leiden(undirected)
        if isinstance(result, tuple):
            partition = result[0]
        else:
            partition = result
        return {str(node_id): int(community_id) for node_id, community_id in partition.items()}
    except ImportError:
        logger.warning("graspologic not available. Using connected-components fallback.")
        return _fallback_components(graph)
    except Exception as exc:
        logger.warning("Leiden failed: %s. Using fallback.", exc)
        return _fallback_components(graph)


def _fallback_components(graph: nx.DiGraph) -> dict[str, int]:
    """Fallback using weakly connected components as communities."""
    communities: dict[str, int] = {}
    for idx, component in enumerate(nx.weakly_connected_components(graph)):
        for node_id in component:
            communities[str(node_id)] = idx
    return communities


def _save_communities(communities: dict[str, int], output_dir: Path) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    path = output_dir / "communities.json"
    with open(path, "w") as f:
        json.dump(communities, f, indent=2)


def _load_communities(output_dir: Path) -> dict[str, int]:
    path = output_dir / "communities.json"
    if not path.exists():
        return {}
    with open(path) as f:
        raw = json.load(f)
    return {str(k): int(v) for k, v in raw.items()}


def add_community_nodes(
    graph: nx.DiGraph,
    communities: dict[str, int],
) -> nx.DiGraph:
    """Add Community nodes and BELONGS_TO edges for assignments."""
    for community_id in set(communities.values()):
        community_node_id = f"community-{community_id}"
        if not graph.has_node(community_node_id):
            graph.add_node(
                community_node_id,
                node_type=NodeType.COMMUNITY.value,
                label=f"Community {community_id}",
            )

    for node_id, community_id in communities.items():
        community_node_id = f"community-{community_id}"
        if graph.has_node(node_id) and not graph.has_edge(node_id, community_node_id):
            graph.add_edge(
                node_id,
                community_node_id,
                edge_type=EdgeType.BELONGS_TO.value,
            )

    return graph
