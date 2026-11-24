#!/usr/bin/env python3
"""Generate a self-contained HTML explorer from the checkout's project Lexicons."""

import argparse
import json
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
POST = "social.craftsky.feed.post"
TEMPLATE = ROOT / "scripts/project-schema/explorer.html"
CATALOG = ROOT / "appview/internal/projectcatalog/catalog-v1.json"


def schema_data():
    definitions = {}
    for path in sorted((ROOT / "lexicon").rglob("*.json")):
        schema = json.loads(path.read_text(encoding="utf-8"))
        nsid = schema["id"]
        for name, definition in schema["defs"].items():
            identifier = nsid if name == "main" else f"{nsid}#{name}"
            definitions[identifier] = {
                "definition": definition,
                "source": path.relative_to(ROOT).as_posix(),
            }

    def qualify(reference, owner):
        if reference.startswith("#"):
            reference = owner.split("#", 1)[0] + reference
        return reference.removesuffix("#main")

    def collect_references(value):
        if isinstance(value, dict):
            if value.get("type") == "ref":
                yield value["ref"]
            yield from value.get("refs", ())
            for key in ("knownValues", "enum"):
                yield from (option for option in value.get(key, ()) if "#" in option)
            for child in value.values():
                yield from collect_references(child)
        elif isinstance(value, list):
            for child in value:
                yield from collect_references(child)

    reachable = {}
    pending = [POST]
    while pending:
        identifier = pending.pop()
        if identifier in reachable or identifier not in definitions:
            continue
        entry = definitions[identifier]
        reachable[identifier] = entry
        pending.extend(
            qualify(reference, identifier)
            for reference in collect_references(entry["definition"])
        )
    catalog = json.loads(CATALOG.read_text(encoding="utf-8"))
    labels = {
        entry["id"]: entry.get("displayLabel", entry["labels"]["en"])
        for entry in catalog["entries"] if entry["id"] in reachable
    }
    project_items = {
        entry["detailsType"]: catalog.get("projectItems", {}).get(entry["id"], [])
        for entry in catalog["entries"]
        if "detailsType" in entry
    }
    return {"post": POST, "definitions": reachable, "labels": labels, "projectItems": project_items}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("output", type=Path, help="path for the standalone HTML file")
    args = parser.parse_args()

    # Escape '<' so text in a Lexicon cannot terminate the JSON script element.
    data = json.dumps(schema_data(), ensure_ascii=False, separators=(",", ":")).replace("<", "\\u003c")
    page = TEMPLATE.read_text(encoding="utf-8").replace("__LEXICON_DATA__", data, 1)
    args.output.write_text(page, encoding="utf-8")


if __name__ == "__main__":
    main()
