"""Graph schema definitions for CoDD knowledge graph (design doc ch.2)."""
from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum


class NodeType(str, Enum):
    DOCUMENT = "Document"
    ENTITY = "Entity"
    CONCEPT = "Concept"
    COMMUNITY = "Community"


class EdgeType(str, Enum):
    DEPENDS_ON = "DEPENDS_ON"  # CoDD explicit dependency
    MENTIONS = "MENTIONS"  # Document mentions Entity/Concept
    RELATED_TO = "RELATED_TO"  # Entity-Entity relationship (LLM extracted)
    BELONGS_TO = "BELONGS_TO"  # Node belongs to Community


@dataclass
class Node:
    node_id: str
    node_type: NodeType
    label: str
    properties: dict = field(default_factory=dict)

    def to_dict(self) -> dict:
        return {
            "node_id": self.node_id,
            "node_type": self.node_type.value,
            "label": self.label,
            "properties": self.properties,
        }


@dataclass
class Edge:
    source_id: str
    target_id: str
    edge_type: EdgeType
    properties: dict = field(default_factory=dict)

    def to_dict(self) -> dict:
        return {
            "source_id": self.source_id,
            "target_id": self.target_id,
            "edge_type": self.edge_type.value,
            "properties": self.properties,
        }
