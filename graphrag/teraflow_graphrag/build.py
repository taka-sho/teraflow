"""Build orchestration stub - will be implemented in S169-2/3/4."""
from __future__ import annotations

from typing import Any


def run_build(args: dict) -> dict[str, Any]:
    """Orchestrate full graph build. Stub for S169-1."""
    raise NotImplementedError(
        "Graph build requires entity extraction (S169-2) and builder (S169-3). "
        "Install and implement those subtasks first."
    )


def run_status(args: dict) -> dict[str, Any]:
    """Return graph status. Stub for S169-1."""
    from pathlib import Path

    from teraflow_graphrag.storage.graphml import load_graph
    from teraflow_graphrag.storage.state import load_state

    project_root = Path(args.get("project_root", "."))
    output_dir = project_root / ".teraflow" / "graphrag"
    state = load_state(output_dir)
    graph = load_graph(output_dir)
    return {
        "node_count": graph.number_of_nodes(),
        "edge_count": graph.number_of_edges(),
        "document_count": len(state.documents),
        "total_edge_count": state.total_edge_count,
    }
