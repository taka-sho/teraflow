"""Graph-based consistency check: extends Phase 1 Go checks with implicit relations."""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

import networkx as nx

from teraflow_graphrag.graph.schema import EdgeType


@dataclass
class CheckIssue:
    severity: str
    issue_type: str
    message: str
    node_id: str = ""
    target_id: str = ""
    source: str = "graphrag"


@dataclass
class CheckResult:
    issues: list[CheckIssue] = field(default_factory=list)
    ok: bool = True

    def to_dict(self) -> dict[str, Any]:
        return {
            "ok": self.ok,
            "issue_count": len(self.issues),
            "issues": [
                {
                    "severity": i.severity,
                    "type": i.issue_type,
                    "message": i.message,
                    "node_id": i.node_id,
                    "target_id": i.target_id,
                    "source": i.source,
                }
                for i in self.issues
            ],
        }


def _is_missing_codd_target(graph: nx.DiGraph, target_id: str) -> bool:
    """Treat placeholders without CoDD metadata as unresolved references."""
    if not graph.has_node(target_id):
        return True
    target_data = graph.nodes[target_id]
    return "status" not in target_data


def check_graph(graph: nx.DiGraph) -> CheckResult:
    """Run all consistency checks: BrokenRef, StatusConflict, ImplicitStatusConflict, CyclicDep."""
    issues: list[CheckIssue] = []

    for src, tgt, edge_data in graph.edges(data=True):
        edge_type = edge_data.get("edge_type", "")

        if edge_type == EdgeType.DEPENDS_ON.value and _is_missing_codd_target(graph, str(tgt)):
            issues.append(
                CheckIssue(
                    severity="error",
                    issue_type="BrokenRef",
                    message=f"Node '{src}' depends on non-existent node '{tgt}'",
                    node_id=str(src),
                    target_id=str(tgt),
                    source="codd",
                )
            )
            continue

        if edge_type == EdgeType.DEPENDS_ON.value:
            src_s = str(graph.nodes[src].get("status", ""))
            tgt_s = str(graph.nodes[tgt].get("status", ""))
            if src_s == "confirmed" and tgt_s in ("draft", "review"):
                issues.append(
                    CheckIssue(
                        severity="warning",
                        issue_type="StatusConflict",
                        message=f"Confirmed node '{src}' depends on {tgt_s} node '{tgt}'",
                        node_id=str(src),
                        target_id=str(tgt),
                        source="codd",
                    )
                )

        if edge_type in (EdgeType.MENTIONS.value, EdgeType.RELATED_TO.value):
            if graph.has_node(tgt):
                src_s = str(graph.nodes[src].get("status", ""))
                tgt_s = str(graph.nodes[tgt].get("status", ""))
                if src_s == "confirmed" and tgt_s == "draft":
                    issues.append(
                        CheckIssue(
                            severity="warning",
                            issue_type="ImplicitStatusConflict",
                            message=(
                                f"Confirmed '{src}' has implicit relation to draft '{tgt}' "
                                f"(via {edge_type})"
                            ),
                            node_id=str(src),
                            target_id=str(tgt),
                            source="graphrag",
                        )
                    )

    dep_graph = nx.DiGraph()
    for src, tgt, ed in graph.edges(data=True):
        if ed.get("edge_type") == EdgeType.DEPENDS_ON.value:
            dep_graph.add_edge(src, tgt)

    try:
        cycle = nx.find_cycle(dep_graph, orientation="original")
        if cycle:
            nodes = [str(cycle[0][0])] + [str(edge[1]) for edge in cycle]
            nodes_str = " \u2192 ".join(nodes)
            issues.append(
                CheckIssue(
                    severity="error",
                    issue_type="CyclicDependency",
                    message=f"Cyclic dependency detected: {nodes_str}",
                    source="codd",
                )
            )
    except nx.NetworkXNoCycle:
        pass

    return CheckResult(issues=issues, ok=len([i for i in issues if i.severity == "error"]) == 0)
