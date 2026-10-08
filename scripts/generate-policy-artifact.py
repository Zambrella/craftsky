#!/usr/bin/env python3
import hashlib
import html
import json
import re
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
POLICIES = [
    ("/terms", "terms-of-service.md", "terms.html", ["BR-005", "BR-006"]),
    ("/privacy", "privacy-policy.md", "privacy.html", ["BR-006", "FR-030", "FR-031"]),
    (
        "/community-guidelines",
        "community-guidelines.md",
        "community-guidelines.html",
        ["FR-015", "FR-016"],
    ),
    (
        "/reporting",
        "reporting-complaints-and-appeals.md",
        "reporting.html",
        ["FR-014", "FR-018", "FR-020"],
    ),
    ("/copyright", "copyright-and-trade-marks.md", "copyright.html", ["FR-038"]),
]


def digest(data: bytes) -> str:
    return "sha256:" + hashlib.sha256(data).hexdigest()


def metadata(lines: list[str]) -> dict[str, str]:
    result: dict[str, str] = {}
    for line in lines:
        match = re.match(r"^\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|$", line)
        if match and match.group(1).strip() not in {"Field", "---"}:
            result[match.group(1).strip()] = match.group(2).strip()
    return result


def inline(value: str) -> str:
    escaped = html.escape(value, quote=False)
    escaped = re.sub(
        r"\[([^]]+)]\(([^)]+)\.md\)",
        lambda m: f'<a href="/{policy_route(m.group(2))}">{m.group(1)}</a>',
        escaped,
    )
    escaped = re.sub(
        r"\[([^]]+)]\((https?://[^)]+)\)",
        r'<a href="\2" rel="noopener">\1</a>',
        escaped,
    )
    escaped = re.sub(r"`([^`]+)`", r"<code>\1</code>", escaped)
    escaped = re.sub(r"\*\*([^*]+)\*\*", r"<strong>\1</strong>", escaped)
    return escaped


def policy_route(source: str) -> str:
    return {
        "terms-of-service": "terms",
        "privacy-policy": "privacy",
        "community-guidelines": "community-guidelines",
        "reporting-complaints-and-appeals": "reporting",
        "copyright-and-trade-marks": "copyright",
    }.get(source, source)


def public_body(lines: list[str]) -> list[str]:
    start = 1
    while start < len(lines) and (
        not lines[start].strip() or lines[start].startswith("|")
    ):
        start += 1
    body = lines[start:]
    for index, line in enumerate(body):
        if line.strip() == "## Change History":
            return body[:index]
    return body


def markdown_to_html(lines: list[str]) -> str:
    output: list[str] = []
    paragraph: list[str] = []
    list_type: str | None = None
    list_items: list[str] = []
    table: list[list[str]] = []

    def flush_paragraph() -> None:
        if paragraph:
            output.append(f"<p>{inline(' '.join(paragraph))}</p>")
            paragraph.clear()

    def flush_list() -> None:
        nonlocal list_type
        if list_type:
            output.extend(f"<li>{inline(item)}</li>" for item in list_items)
            output.append(f"</{list_type}>")
            list_type = None
            list_items.clear()

    def flush_table() -> None:
        if not table:
            return
        rows = table[:]
        table.clear()
        if len(rows) > 1 and all(set(cell) <= {"-", ":"} for cell in rows[1]):
            rows.pop(1)
        output.append('<div class="legal__table-wrap"><table>')
        for row_index, row in enumerate(rows):
            tag = "th" if row_index == 0 else "td"
            output.append(
                "<tr>"
                + "".join(f"<{tag}>{inline(cell)}</{tag}>" for cell in row)
                + "</tr>"
            )
        output.append("</table></div>")

    for raw in lines + [""]:
        line = raw.strip()
        if line.startswith("|") and line.endswith("|"):
            flush_paragraph()
            flush_list()
            table.append([cell.strip() for cell in line.strip("|").split("|")])
            continue
        flush_table()
        heading = re.match(r"^(#{2,4})\s+(.+)$", line)
        item = re.match(r"^[-*]\s+(.+)$", line)
        numbered = re.match(r"^\d+\.\s+(.+)$", line)
        if heading:
            flush_paragraph()
            flush_list()
            level = len(heading.group(1))
            output.append(f"<h{level}>{inline(heading.group(2))}</h{level}>")
        elif item or numbered:
            flush_paragraph()
            desired = "ul" if item else "ol"
            if list_type != desired:
                flush_list()
                list_type = desired
                output.append(f"<{desired}>")
            item_text = item.group(1) if item is not None else numbered.group(1)  # type: ignore[union-attr]
            list_items.append(item_text)
        elif line.startswith("> "):
            flush_paragraph()
            flush_list()
            output.append(f"<blockquote>{inline(line[2:])}</blockquote>")
        elif not line:
            flush_paragraph()
            flush_list()
        elif list_type and list_items:
            list_items[-1] += " " + line
        else:
            paragraph.append(line)
    return "\n".join(output)


def render(title: str, route: str, meta: dict[str, str], body: str) -> bytes:
    version = meta.get("Version", "")
    status = meta.get("Status", "")
    effective = meta.get("Proposed effective date", meta.get("Effective date", ""))
    warning = ""
    if status != "Approved":
        warning = '<p class="legal__draft" role="status"><strong>Draft, not effective.</strong> Publication remains blocked pending the approvals and evidence recorded in the release manifest.</p>'
    document = f"""<!DOCTYPE html>
<html lang="en-GB">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{html.escape(title)} - CraftSky</title>
  <link rel="canonical" href="https://craftsky.social{route}" />
  <link rel="icon" href="/assets/favicon.svg" type="image/svg+xml" />
  <link rel="stylesheet" href="/styles.css" />
</head>
<body>
  <a class="skip-link" href="#main">Skip to main content</a>
  <header class="page-header"><div class="container"><a class="page-header__home" href="/"><img src="/assets/footer-logo.png" alt="" /><span>CraftSky</span></a></div></header>
  <hr class="section-rule" />
  <main id="main" class="legal"><article class="container container--narrow">
    <h1 class="legal__title">{html.escape(title)}</h1>
    <p class="legal__updated">Version {html.escape(version)}. Proposed effective date: {html.escape(effective)}.</p>
    {warning}
    {body}
  </article></main>
  <script src="/main.js" defer></script>
</body>
</html>
"""
    return document.encode()


def main() -> None:
    entries = []
    for route, source_name, artifact_name, controls in POLICIES:
        source_path = ROOT / "online-safety" / "policies" / source_name
        source = source_path.read_bytes()
        lines = source.decode().splitlines()
        meta = metadata(lines)
        title = lines[0].removeprefix("# ")
        artifact = render(title, route, meta, markdown_to_html(public_body(lines)))
        artifact_path = ROOT / "web" / artifact_name
        artifact_path.write_bytes(artifact)
        entries.append(
            {
                "route": route,
                "source": str(source_path.relative_to(ROOT)),
                "artifact": str(artifact_path.relative_to(ROOT)),
                "version": meta.get("Version", ""),
                "effectiveDate": meta.get(
                    "Proposed effective date", meta.get("Effective date", "")
                ),
                "approvalStatus": meta.get("Status", ""),
                "approvedBy": meta.get("Approved by", ""),
                "sourceDigest": digest(source),
                "artifactDigest": digest(artifact),
                "presentTenseClaims": [
                    {"control": control, "evidence": [], "status": "pending"}
                    for control in controls
                ],
            }
        )

    manifest = {
        "schemaVersion": 1,
        "publicationStatus": "blocked",
        "generatedFrom": "online-safety/policies",
        "policies": entries,
        "releaseControls": {
            "scannerProduction": {
                "status": "blocked",
                "evidence": [],
                "reason": "Production IWF access and validation evidence are external and missing.",
            },
            "videoProductionDisabled": {
                "status": "pending",
                "evidence": [],
                "reason": "Production video release-gate evidence is not yet linked.",
            },
        },
        "externalBlockers": [
            "Accountable owner and specialist policy approval",
            "Final operator/controller details and DPO position",
            "Worldwide jurisdiction review",
            "Commercial and subscription policy scope decision",
            "Atomic Cloudflare deployment and rollback evidence",
        ],
    }
    (ROOT / "web" / "policy-publication-manifest.json").write_text(
        json.dumps(manifest, indent=2) + "\n"
    )


if __name__ == "__main__":
    main()
