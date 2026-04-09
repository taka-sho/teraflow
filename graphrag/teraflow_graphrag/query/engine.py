"""GraphEngine interface for query operations (design doc ch.6)."""
from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any


@dataclass
class SearchResult:
    query: str
    mode: str  # "local" or "global"
    answer: str
    sources: list[dict] = field(default_factory=list)
    # source field per design doc 9.4: "codd" or "graphrag"
    metadata: dict = field(default_factory=dict)


class GraphEngine(ABC):
    """Abstract interface for all graph query operations."""

    @abstractmethod
    def search(self, query: str, mode: str = "local", top_k: int = 5) -> SearchResult:
        """Perform natural language search over the knowledge graph."""

    @abstractmethod
    def impact(self, node_id: str, depth: int = 2) -> dict[str, Any]:
        """Analyze impact of changes to a given node."""

    @abstractmethod
    def check(self) -> dict[str, Any]:
        """Check graph consistency (status conflicts, broken refs, cycles)."""
