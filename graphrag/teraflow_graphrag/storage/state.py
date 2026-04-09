"""State management for incremental graph builds (design doc ch.4)."""
from __future__ import annotations

import json
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Dict


@dataclass
class DocumentState:
    content_hash: str
    node_count: int = 0
    edge_count: int = 0


@dataclass
class GraphState:
    documents: Dict[str, DocumentState] = field(default_factory=dict)
    total_edge_count: int = 0
    community_edge_count: int = 0  # edge count when community last ran

    @property
    def edge_change_rate(self) -> float:
        if self.community_edge_count == 0:
            return 1.0
        added = abs(self.total_edge_count - self.community_edge_count)
        return added / self.community_edge_count


def load_state(output_dir: Path) -> GraphState:
    path = output_dir / "state.json"
    if not path.exists():
        return GraphState()
    with open(path) as f:
        raw = json.load(f)
    docs = {k: DocumentState(**v) for k, v in raw.get("documents", {}).items()}
    return GraphState(
        documents=docs,
        total_edge_count=raw.get("total_edge_count", 0),
        community_edge_count=raw.get("community_edge_count", 0),
    )


def save_state(state: GraphState, output_dir: Path) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    path = output_dir / "state.json"
    data = {
        "documents": {k: asdict(v) for k, v in state.documents.items()},
        "total_edge_count": state.total_edge_count,
        "community_edge_count": state.community_edge_count,
    }
    with open(path, "w") as f:
        json.dump(data, f, indent=2)
