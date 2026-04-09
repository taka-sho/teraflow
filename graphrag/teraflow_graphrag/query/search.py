"""Local and Global search implementations (design doc ch.3.2)."""
from __future__ import annotations

import json
import logging
from pathlib import Path

import networkx as nx

from teraflow_graphrag.query.engine import SearchResult

logger = logging.getLogger(__name__)

LOCAL_SEARCH_PROMPT = """以下の知識グラフ情報を参考に、質問に答えてください。

質問: {query}

関連エンティティ:
{entities_str}

関連ドキュメント:
{documents_str}

簡潔に回答してください。"""

GLOBAL_SEARCH_PROMPT = """以下のコミュニティ要約を参考に、質問に答えてください。

質問: {query}

コミュニティ概要:
{communities_str}

簡潔に回答してください。"""


class LocalSearch:
    """Entity-based local search: find entities matching query, then traverse to related documents."""

    def __init__(
        self,
        graph: nx.DiGraph,
        model: str = "claude-haiku-4-5-20251001",
        max_tokens: int = 1024,
    ) -> None:
        self.graph = graph
        self.model = model
        self.max_tokens = max_tokens

    def search(self, query: str, top_k: int = 5) -> SearchResult:
        """Find entities related to query, return relevant documents."""
        import litellm

        query_lower = query.lower()
        matching_nodes: list[tuple[str, dict]] = []
        for node_id, data in self.graph.nodes(data=True):
            label = str(data.get("label", "")).lower()
            if query_lower in label or any(word in label for word in query_lower.split()):
                matching_nodes.append((node_id, data))

        related_docs: list[dict] = []
        seen_doc_ids: set[str] = set()
        for node_id, _ in matching_nodes[:top_k]:
            neighbors = list(self.graph.predecessors(node_id)) + list(self.graph.successors(node_id))
            for neighbor in neighbors:
                ndata = self.graph.nodes[neighbor]
                if ndata.get("node_type") == "Document" and neighbor not in seen_doc_ids:
                    seen_doc_ids.add(str(neighbor))
                    related_docs.append({"node_id": str(neighbor), **ndata})

        entities_str = "\n".join(
            f"- {nid}: {data.get('label', '')}" for nid, data in matching_nodes[:top_k]
        ) or "（なし）"
        documents_str = "\n".join(
            f"- {doc['node_id']}: {doc.get('label', '')}" for doc in related_docs[:top_k]
        ) or "（なし）"

        prompt = LOCAL_SEARCH_PROMPT.format(
            query=query,
            entities_str=entities_str,
            documents_str=documents_str,
        )

        try:
            response = litellm.completion(
                model=self.model,
                messages=[{"role": "user", "content": prompt}],
                max_tokens=self.max_tokens,
            )
            answer = response.choices[0].message.content or ""
        except Exception as exc:
            logger.warning("LLM search failed: %s", exc)
            answer = f"検索結果: {len(matching_nodes)} エンティティが見つかりました。"

        return SearchResult(
            query=query,
            mode="local",
            answer=answer,
            sources=[
                {"node_id": nid, "label": data.get("label", ""), "source": "graphrag"}
                for nid, data in matching_nodes[:top_k]
            ],
        )


class GlobalSearch:
    """Community summary-based global search."""

    def __init__(
        self,
        output_dir: Path,
        model: str = "claude-sonnet-4-6",
        max_tokens: int = 2048,
    ) -> None:
        self.output_dir = output_dir
        self.model = model
        self.max_tokens = max_tokens

    def _load_summaries(self) -> dict:
        summaries_path = self.output_dir / "summaries" / "communities.json"
        if not summaries_path.exists():
            return {}
        with open(summaries_path, encoding="utf-8") as f:
            return json.load(f)

    def search(self, query: str, top_k: int = 5) -> SearchResult:
        """Search using community summaries for global context."""
        import litellm

        summaries = self._load_summaries()
        if not summaries:
            return SearchResult(
                query=query,
                mode="global",
                answer="コミュニティ要約が見つかりません。graph build を実行してください。",
                sources=[],
            )

        communities_str = "\n".join(
            f"コミュニティ{cid}: {data.get('summary', '')}"
            for cid, data in list(summaries.items())[:top_k]
        )

        prompt = GLOBAL_SEARCH_PROMPT.format(
            query=query,
            communities_str=communities_str,
        )

        try:
            response = litellm.completion(
                model=self.model,
                messages=[{"role": "user", "content": prompt}],
                max_tokens=self.max_tokens,
            )
            answer = response.choices[0].message.content or ""
        except Exception as exc:
            logger.warning("Global search LLM failed: %s", exc)
            answer = f"グローバル検索: {len(summaries)} コミュニティを参照しました。"

        return SearchResult(
            query=query,
            mode="global",
            answer=answer,
            sources=[
                {"community_id": cid, "summary": data.get("summary", ""), "source": "graphrag"}
                for cid, data in list(summaries.items())[:top_k]
            ],
        )
