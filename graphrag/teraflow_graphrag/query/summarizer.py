"""Community summary generation using LLM (design doc ch.6, ch.7)."""
from __future__ import annotations

import json
import logging
from pathlib import Path

logger = logging.getLogger(__name__)

COMMUNITY_SUMMARY_PROMPT = """以下のコミュニティに属するドキュメント群を分析し、
このコミュニティが表す概念や主題を1-2文で要約してください。

コミュニティID: {community_id}
ノード一覧:
{nodes_str}

JSON形式で回答してください:
{{"summary": "要約テキスト", "themes": ["テーマ1", "テーマ2"]}}
"""


class CommunitySummarizer:
    """Generate LLM summaries for each community."""

    def __init__(self, model: str = "claude-haiku-4-5-20251001", max_tokens: int = 512) -> None:
        self.model = model
        self.max_tokens = max_tokens

    def summarize_community(
        self,
        community_id: int,
        nodes: list[dict],
    ) -> dict:
        """Generate a summary for a single community."""
        import litellm

        nodes_str = "\n".join(
            f"- {n.get('node_id', '')}: {n.get('label', '')} ({n.get('node_type', '')})"
            for n in nodes[:30]
        )
        prompt = COMMUNITY_SUMMARY_PROMPT.format(
            community_id=community_id,
            nodes_str=nodes_str,
        )
        try:
            response = litellm.completion(
                model=self.model,
                messages=[{"role": "user", "content": prompt}],
                max_tokens=self.max_tokens,
                response_format={"type": "json_object"},
            )
            raw = response.choices[0].message.content or "{}"
            data = json.loads(raw)
            return {
                "community_id": community_id,
                "summary": data.get("summary", ""),
                "themes": data.get("themes", []),
                "node_count": len(nodes),
            }
        except Exception as exc:
            logger.warning("Community summarization failed for %d: %s", community_id, exc)
            return {
                "community_id": community_id,
                "summary": "",
                "themes": [],
                "node_count": len(nodes),
            }

    def summarize_all(
        self,
        communities: dict[str, int],
        graph_nodes: dict,
        output_dir: Path,
    ) -> dict[int, dict]:
        """Summarize all communities and save to summaries/."""
        community_nodes: dict[int, list[dict]] = {}
        for node_id, cid in communities.items():
            node_data = graph_nodes.get(node_id, {})
            community_nodes.setdefault(cid, []).append({
                "node_id": node_id,
                **node_data,
            })

        summaries: dict[int, dict] = {}
        for cid, nodes in community_nodes.items():
            summaries[cid] = self.summarize_community(cid, nodes)

        summaries_dir = output_dir / "summaries"
        summaries_dir.mkdir(parents=True, exist_ok=True)
        with open(summaries_dir / "communities.json", "w", encoding="utf-8") as f:
            json.dump(
                {str(k): v for k, v in summaries.items()},
                f,
                indent=2,
                ensure_ascii=False,
            )
        return summaries
