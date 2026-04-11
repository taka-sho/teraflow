"""
Subprocess entry point for teraflow-graphrag.
Protocol: stdin JSON -> stdout JSON
Request format: {"command": "...", ...} or {"command": "...", "args": {...}}
Response format (success): command-specific JSON payload
Response format (error): {"error": "...", "code": "..."}
"""
from __future__ import annotations

import json
import sys
from pathlib import Path
from typing import Any

import networkx as nx


def _error(message: str, code: str = "internal_error") -> dict[str, str]:
    return {"error": message, "code": code}


def _param(request: dict[str, Any], args: dict[str, Any], key: str, default: Any = None) -> Any:
    if key in request:
        return request.get(key)
    return args.get(key, default)


def _resolve_graph_path(request: dict[str, Any], args: dict[str, Any]) -> Path:
    graph_path = _param(request, args, "graph_path", "")
    storage_path = _param(request, args, "storage_path", "")

    if graph_path:
        return Path(str(graph_path))
    if storage_path:
        return Path(str(storage_path)) / "graph.graphml"
    return Path(".teraflow") / "graphrag" / "graph.graphml"


def _load_graph(request: dict[str, Any], args: dict[str, Any]) -> nx.DiGraph:
    graph_file = _resolve_graph_path(request, args)
    if not graph_file.exists():
        raise FileNotFoundError(f"graph file not found: {graph_file}")
    return nx.read_graphml(str(graph_file))


def _search_result_to_dict(result: Any) -> dict[str, Any]:
    return {
        "query": result.query,
        "mode": result.mode,
        "answer": result.answer,
        "sources": result.sources,
        "metadata": result.metadata,
    }


def main() -> None:
    try:
        raw = sys.stdin.read()
        request = json.loads(raw)
        command = request.get("command", "")
        args = request.get("args", {})
        if not isinstance(args, dict):
            args = {}

        legacy_wrap = False
        if command == "build":
            from teraflow_graphrag.build import run_build

            data = run_build(args)
            legacy_wrap = True
        elif command == "status":
            from teraflow_graphrag.build import run_status

            data = run_status(args)
            legacy_wrap = True
        elif command == "query":
            from teraflow_graphrag.query.search import GlobalSearch, LocalSearch

            graph = _load_graph(request, args)
            query = str(_param(request, args, "query", "")).strip()
            mode = str(_param(request, args, "mode", "local")).lower()
            top_k = int(_param(request, args, "top_k", 5))
            storage_path = _param(request, args, "storage_path", "")

            if not query:
                raise ValueError("query is required")
            if mode not in {"local", "global"}:
                raise ValueError(f"unsupported mode: {mode}")

            if mode == "local":
                result = LocalSearch(graph=graph).search(query=query, top_k=top_k)
            else:
                if storage_path:
                    output_dir = Path(str(storage_path))
                else:
                    output_dir = _resolve_graph_path(request, args).parent
                result = GlobalSearch(output_dir=output_dir).search(query=query, top_k=top_k)
            data = _search_result_to_dict(result)
        elif command == "impact":
            from teraflow_graphrag.query.impact import analyze_impact

            graph = _load_graph(request, args)
            node_id = str(_param(request, args, "node_id", "")).strip()
            depth = int(_param(request, args, "depth", 2))
            include_graphrag = bool(_param(request, args, "include_graphrag", True))
            if not node_id:
                raise ValueError("node_id is required")

            data = analyze_impact(
                graph=graph,
                node_id=node_id,
                depth=depth,
                include_graphrag=include_graphrag,
            ).to_dict()
        elif command == "check":
            from teraflow_graphrag.query.check import check_graph

            graph = _load_graph(request, args)
            data = check_graph(graph).to_dict()
        else:
            raise ValueError(f"Unknown command: {command}")

        if legacy_wrap:
            print(json.dumps({"ok": True, "data": data}))
        else:
            print(json.dumps(data))
    except ValueError as e:
        msg = str(e)
        code = "invalid_argument"
        if msg.startswith("Unknown command"):
            code = "unknown_command"
        print(json.dumps(_error(msg, code)), file=sys.stdout)
        sys.exit(1)
    except FileNotFoundError as e:
        print(json.dumps(_error(str(e), "not_found")), file=sys.stdout)
        sys.exit(1)
    except Exception as e:
        print(json.dumps(_error(str(e), "internal_error")), file=sys.stdout)
        sys.exit(1)


if __name__ == "__main__":
    main()
