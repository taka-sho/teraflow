"""Build orchestration - combines CoDD graph loading with entity extraction."""
from __future__ import annotations

from pathlib import Path
from typing import Any


def run_build(args: dict) -> dict[str, Any]:
    """Orchestrate graph build (CoDD + optional LLM extraction)."""
    from teraflow_graphrag.config import load_config
    from teraflow_graphrag.graph.builder import GraphBuilder

    project_root = Path(args.get("project_root", "."))
    config = load_config(project_root)
    output_dir = project_root / config.output_dir
    index_path = project_root / ".teraflow" / "index.yml"

    builder = GraphBuilder(project_root=project_root, output_dir=output_dir)
    builder.load_codd(index_path=index_path)

    # LLM entity extraction will be integrated in full build phase.
    # For now, CoDD-only build is supported.
    builder.save()

    return {
        "node_count": builder.graph.number_of_nodes(),
        "edge_count": builder.graph.number_of_edges(),
        "document_count": len(builder.state.documents),
    }


def run_status(args: dict) -> dict[str, Any]:
    """Return graph status. Stub for S169-1."""
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
