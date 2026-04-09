"""Integration tests: CoDD fixtures -> graph build -> graphml output validation."""
from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path

from teraflow_graphrag.graph.builder import GraphBuilder
from teraflow_graphrag.graph.community import detect_communities
from teraflow_graphrag.graph.schema import NodeType
from teraflow_graphrag.storage.graphml import load_graph
from teraflow_graphrag.storage.state import GraphState

FIXTURES_DIR = Path(__file__).parent / "fixtures"


def test_codd_graph_from_fixtures(tmp_path):
    """Full flow: load fixtures/index.yml -> build CoDD graph -> save/load GraphML."""
    output_dir = tmp_path / "graphrag"
    index_path = FIXTURES_DIR / "index.yml"

    builder = GraphBuilder(project_root=FIXTURES_DIR, output_dir=output_dir)
    builder.load_codd(index_path=index_path)
    builder.save()

    assert (output_dir / "graph.graphml").exists()
    assert (output_dir / "state.json").exists()

    graph = load_graph(output_dir)

    assert graph.has_node("doc-auth")
    assert graph.has_node("doc-security")
    assert graph.has_node("doc-user")
    assert graph.has_edge("doc-auth", "doc-security")
    assert graph.has_edge("doc-user", "doc-auth")

    assert graph.nodes["doc-auth"]["node_type"] == NodeType.DOCUMENT.value
    assert graph.nodes["doc-auth"]["label"] == "認証設計書"


def test_incremental_build_skips_unchanged(tmp_path):
    """Documents with unchanged content should be skipped in incremental build."""
    output_dir = tmp_path / "graphrag"
    index_path = FIXTURES_DIR / "index.yml"

    builder = GraphBuilder(project_root=FIXTURES_DIR, output_dir=output_dir)
    builder.load_codd(index_path=index_path)
    builder.mark_processed("doc-auth", "content-v1")
    builder.save()

    builder2 = GraphBuilder(project_root=FIXTURES_DIR, output_dir=output_dir)
    assert builder2.needs_update("doc-auth", "content-v1") is False
    assert builder2.needs_update("doc-auth", "content-v2") is True


def test_community_detection_on_codd_graph(tmp_path):
    """Community detection on the CoDD fixture graph."""
    output_dir = tmp_path / "graphrag"
    index_path = FIXTURES_DIR / "index.yml"

    builder = GraphBuilder(project_root=FIXTURES_DIR, output_dir=output_dir)
    builder.load_codd(index_path=index_path)

    state = GraphState(total_edge_count=10, community_edge_count=0)
    communities = detect_communities(
        builder.graph,
        state,
        output_dir,
        threshold=0.1,
        force=True,
    )

    assert "doc-auth" in communities
    assert "doc-security" in communities
    assert "doc-user" in communities
    assert (output_dir / "communities.json").exists()


def test_subprocess_protocol_run_status(tmp_path):
    """Test __main__.py subprocess JSON protocol with 'status' command."""
    request = json.dumps(
        {
            "command": "status",
            "args": {"project_root": str(tmp_path)},
        }
    )

    result = subprocess.run(
        [sys.executable, "-m", "teraflow_graphrag"],
        input=request,
        capture_output=True,
        text=True,
        cwd=Path(__file__).parent.parent,
        check=False,
    )

    assert result.returncode == 0, f"stderr: {result.stderr} stdout: {result.stdout}"
    response = json.loads(result.stdout)
    assert response["ok"] is True
    assert "node_count" in response["data"]
    assert "edge_count" in response["data"]
