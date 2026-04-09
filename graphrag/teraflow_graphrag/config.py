"""Configuration loader for teraflow-graphrag."""
from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

import yaml


@dataclass
class GraphRAGConfig:
    model: str = "claude-haiku-4-5-20251001"
    max_tokens: int = 4096
    output_dir: str = ".teraflow/graphrag"
    community_threshold: float = 0.1  # edge_change_rate > 10% -> Leiden re-run
    incremental: bool = True


def load_config(project_root: Path) -> GraphRAGConfig:
    """Load GraphRAG configuration from .github/teraflow.yml."""
    config_path = project_root / ".github" / "teraflow.yml"
    if not config_path.exists():
        return GraphRAGConfig()

    with open(config_path) as f:
        raw = yaml.safe_load(f) or {}

    graphrag_section = raw.get("graphrag", {})
    cfg = GraphRAGConfig()
    if "model" in graphrag_section:
        cfg.model = graphrag_section["model"]
    if "max_tokens" in graphrag_section:
        cfg.max_tokens = int(graphrag_section["max_tokens"])
    if "output_dir" in graphrag_section:
        cfg.output_dir = graphrag_section["output_dir"]
    if "community_threshold" in graphrag_section:
        cfg.community_threshold = float(graphrag_section["community_threshold"])
    if "incremental" in graphrag_section:
        cfg.incremental = bool(graphrag_section["incremental"])
    return cfg
