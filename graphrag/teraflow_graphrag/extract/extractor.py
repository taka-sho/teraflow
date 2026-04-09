"""Entity extractor using LLM via litellm (design doc ch.6, ch.7)."""
from __future__ import annotations

import json
import logging
from dataclasses import dataclass
from typing import Optional

from teraflow_graphrag.extract.prompts import (
    ENTITY_EXTRACTION_SYSTEM,
    build_extraction_prompt,
)
from teraflow_graphrag.graph.schema import Edge, EdgeType, Node, NodeType

logger = logging.getLogger(__name__)


@dataclass
class ExtractionResult:
    entities: list[Node]
    relations: list[Edge]
    raw_response: str = ""


class EntityExtractor:
    """Extract entities and relations from a CoDD document using LLM."""

    def __init__(
        self, model: str = "claude-haiku-4-5-20251001", max_tokens: int = 4096
    ) -> None:
        self.model = model
        self.max_tokens = max_tokens

    def extract(
        self,
        node_id: str,
        title: str,
        content: str,
        status: str = "",
        known_entities: Optional[list[dict]] = None,
    ) -> ExtractionResult:
        """Extract entities and relations from a document."""
        import litellm

        prompt = build_extraction_prompt(
            node_id=node_id,
            title=title,
            status=status,
            content=content,
            known_entities=known_entities or [],
        )

        try:
            response = litellm.completion(
                model=self.model,
                messages=[
                    {"role": "system", "content": ENTITY_EXTRACTION_SYSTEM},
                    {"role": "user", "content": prompt},
                ],
                max_tokens=self.max_tokens,
                response_format={"type": "json_object"},
            )
            raw = response.choices[0].message.content or ""
            return self._parse_response(raw)
        except Exception as e:
            logger.warning("LLM extraction failed for %s: %s", node_id, e)
            return ExtractionResult(entities=[], relations=[], raw_response=str(e))

    def _parse_response(self, raw: str) -> ExtractionResult:
        """Parse LLM JSON response into Node/Edge objects."""
        try:
            data = json.loads(raw)
        except json.JSONDecodeError:
            logger.warning("Failed to parse JSON response: %s", raw[:200])
            return ExtractionResult(entities=[], relations=[], raw_response=raw)

        entities: list[Node] = []
        for entity in data.get("entities", []):
            entity_id = entity.get("entity_id", "")
            if not entity_id:
                continue
            node_type = (
                NodeType.ENTITY
                if entity.get("entity_type") == "Entity"
                else NodeType.CONCEPT
            )
            entities.append(
                Node(
                    node_id=entity_id,
                    node_type=node_type,
                    label=entity.get("label", entity_id),
                    properties={"description": entity.get("description", "")},
                )
            )

        relations: list[Edge] = []
        for relation in data.get("relations", []):
            source_id = relation.get("source_id", "")
            target_id = relation.get("target_id", "")
            if not source_id or not target_id:
                continue
            relation_type = relation.get("relation_type", "RELATED_TO")
            try:
                edge_type = EdgeType(relation_type)
            except ValueError:
                edge_type = EdgeType.RELATED_TO
            relations.append(
                Edge(
                    source_id=source_id,
                    target_id=target_id,
                    edge_type=edge_type,
                    properties={"description": relation.get("description", "")},
                )
            )

        return ExtractionResult(entities=entities, relations=relations, raw_response=raw)
