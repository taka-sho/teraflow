"""E2E tests for GraphRAG CLI protocol (build -> search/impact/check)."""
from __future__ import annotations

import hashlib
import io
import json
import sys
from pathlib import Path
from types import SimpleNamespace

import yaml

from teraflow_graphrag.__main__ import main

FIXTURES_DIR = Path(__file__).parent / "fixtures" / "codd"


def _run_main(payload: dict, monkeypatch) -> tuple[int, dict]:
    stdin = io.StringIO(json.dumps(payload))
    stdout = io.StringIO()
    monkeypatch.setattr(sys, "stdin", stdin)
    monkeypatch.setattr(sys, "stdout", stdout)

    code = 0
    try:
        main()
    except SystemExit as e:
        code = int(e.code)

    stdout.seek(0)
    return code, json.loads(stdout.read())


def _prepare_project(tmp_path: Path) -> Path:
    project_root = tmp_path / "project"
    docs_dir = project_root / "docs"
    index_dir = project_root / ".teraflow"
    docs_dir.mkdir(parents=True)
    index_dir.mkdir(parents=True)

    entries: list[dict] = []
    for src in sorted(FIXTURES_DIR.glob("*.codd.md")):
        text = src.read_text(encoding="utf-8")
        copied = docs_dir / src.name
        copied.write_text(text, encoding="utf-8")

        parts = text.split("---", 2)
        frontmatter = yaml.safe_load(parts[1]) if len(parts) >= 3 else {}
        codd = frontmatter.get("codd", {})
        body = parts[2] if len(parts) >= 3 else text

        depends_on = []
        for dep in codd.get("depends_on", []):
            if isinstance(dep, dict):
                dep_id = str(dep.get("id", "")).strip()
                if dep_id:
                    depends_on.append(dep_id)
            elif dep:
                depends_on.append(str(dep))

        entries.append(
            {
                "node_id": codd["node_id"],
                "title": codd.get("title", codd["node_id"]),
                "path": f"docs/{src.name}",
                "status": codd.get("status", "draft"),
                "content_hash": hashlib.sha256(body.encode("utf-8")).hexdigest(),
                "depends_on": depends_on,
                "tags": codd.get("tags", []),
            }
        )

    index = {"entries": entries}
    (index_dir / "index.yml").write_text(
        yaml.safe_dump(index, sort_keys=False, allow_unicode=True),
        encoding="utf-8",
    )
    return project_root


def _build_graph(project_root: Path, monkeypatch) -> Path:
    code, response = _run_main(
        {"command": "build", "args": {"project_root": str(project_root)}},
        monkeypatch,
    )
    assert code == 0
    assert response["ok"] is True
    assert response["data"]["node_count"] >= 3
    graph_path = project_root / ".teraflow" / "graphrag" / "graph.graphml"
    assert graph_path.exists()
    return graph_path


def test_e2e_build_and_search_local(tmp_path, monkeypatch):
    project_root = _prepare_project(tmp_path)
    graph_path = _build_graph(project_root, monkeypatch)

    fake_litellm = SimpleNamespace(
        completion=lambda **kwargs: SimpleNamespace(
            choices=[SimpleNamespace(message=SimpleNamespace(content="local mocked answer"))]
        )
    )
    monkeypatch.setitem(sys.modules, "litellm", fake_litellm)

    code, response = _run_main(
        {
            "command": "query",
            "args": {
                "query": "認証",
                "mode": "local",
                "graph_path": str(graph_path),
                "storage_path": str(graph_path.parent),
            },
        },
        monkeypatch,
    )

    assert code == 0
    assert response["mode"] == "local"
    assert response["answer"] == "local mocked answer"
    assert len(response["sources"]) > 0


def test_e2e_build_and_impact(tmp_path, monkeypatch):
    project_root = _prepare_project(tmp_path)
    graph_path = _build_graph(project_root, monkeypatch)

    code, response = _run_main(
        {
            "command": "impact",
            "args": {
                "node_id": "design:auth",
                "depth": 2,
                "include_graphrag": True,
                "graph_path": str(graph_path),
            },
        },
        monkeypatch,
    )

    assert code == 0
    assert response["root_node_id"] == "design:auth"
    assert response["total_count"] >= 1


def test_e2e_build_and_check(tmp_path, monkeypatch):
    project_root = _prepare_project(tmp_path)
    graph_path = _build_graph(project_root, monkeypatch)

    code, response = _run_main(
        {
            "command": "check",
            "args": {
                "graph_path": str(graph_path),
            },
        },
        monkeypatch,
    )

    assert code == 0
    assert "ok" in response
    assert "issues" in response


def test_e2e_graceful_degradation(tmp_path, monkeypatch):
    project_root = _prepare_project(tmp_path)
    graph_path = _build_graph(project_root, monkeypatch)

    fake_litellm = SimpleNamespace(
        completion=lambda **kwargs: (_ for _ in ()).throw(RuntimeError("litellm unavailable"))
    )
    monkeypatch.setitem(sys.modules, "litellm", fake_litellm)

    code, response = _run_main(
        {
            "command": "query",
            "args": {
                "query": "認証",
                "mode": "local",
                "graph_path": str(graph_path),
                "storage_path": str(graph_path.parent),
            },
        },
        monkeypatch,
    )

    assert code == 0
    assert response["mode"] == "local"
    assert "検索結果:" in response["answer"]
