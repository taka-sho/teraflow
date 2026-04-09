"""GraphML storage backend for knowledge graph (design doc ch.5)."""
from __future__ import annotations

from pathlib import Path

import networkx as nx


def save_graph(graph: nx.DiGraph, output_dir: Path) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    path = output_dir / "graph.graphml"
    nx.write_graphml(graph, str(path))


def load_graph(output_dir: Path) -> nx.DiGraph:
    path = output_dir / "graph.graphml"
    if not path.exists():
        return nx.DiGraph()
    return nx.read_graphml(str(path))
