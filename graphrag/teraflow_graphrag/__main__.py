"""
Subprocess entry point for teraflow-graphrag.
Protocol: stdin JSON -> stdout JSON
Request format: {"command": "build|status", "args": {...}}
Response format: {"ok": true, "data": {...}} or {"ok": false, "error": "..."}
"""
import json
import sys


def main() -> None:
    try:
        raw = sys.stdin.read()
        request = json.loads(raw)
        command = request.get("command", "")
        args = request.get("args", {})

        if command == "build":
            from teraflow_graphrag.build import run_build

            data = run_build(args)
        elif command == "status":
            from teraflow_graphrag.build import run_status

            data = run_status(args)
        else:
            raise ValueError(f"Unknown command: {command}")

        print(json.dumps({"ok": True, "data": data}))
    except Exception as e:
        print(json.dumps({"ok": False, "error": str(e)}), file=sys.stdout)
        sys.exit(1)


if __name__ == "__main__":
    main()
