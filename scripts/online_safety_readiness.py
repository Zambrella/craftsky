#!/usr/bin/env python3
import argparse
import hashlib
import json
import re
from dataclasses import dataclass
from pathlib import Path


REQUIRED_ROUTES = {
    "/terms",
    "/privacy",
    "/community-guidelines",
    "/reporting",
    "/copyright",
}


@dataclass(frozen=True)
class Finding:
    control: str
    location: str
    message: str


def file_digest(path: Path) -> str:
    return "sha256:" + hashlib.sha256(path.read_bytes()).hexdigest()


def evidence_reference_exists(root: Path, reference: object) -> bool:
    if not isinstance(reference, str) or not reference.strip():
        return False
    try:
        candidate = (root / reference).resolve()
        candidate.relative_to(root.resolve())
    except (OSError, ValueError):
        return False
    return candidate.is_file()


def check_manifest(root: Path, manifest_path: Path) -> list[Finding]:
    findings: list[Finding] = []
    try:
        manifest = json.loads(manifest_path.read_text())
    except (OSError, json.JSONDecodeError) as error:
        return [
            Finding(
                "POLICY-ARTIFACT", str(manifest_path), f"manifest unreadable: {error}"
            )
        ]

    if not isinstance(manifest, dict):
        return [
            Finding(
                "POLICY-ARTIFACT",
                str(manifest_path),
                "manifest root must be an object",
            )
        ]

    policies = manifest.get("policies", [])
    if not isinstance(policies, list):
        findings.append(
            Finding(
                "POLICY-ARTIFACT",
                str(manifest_path),
                "policies must be an array",
            )
        )
        policies = []
    if manifest.get("publicationStatus") != "approved":
        findings.append(
            Finding(
                "POLICY-PUBLICATION",
                str(manifest_path),
                "publication status is not approved",
            )
        )
    routes = {policy.get("route") for policy in policies if isinstance(policy, dict)}
    if routes != REQUIRED_ROUTES or len(policies) != len(REQUIRED_ROUTES):
        findings.append(
            Finding(
                "POLICY-ARTIFACT",
                str(manifest_path),
                "the exact five policy routes are required",
            )
        )
    for policy in policies:
        if not isinstance(policy, dict):
            findings.append(
                Finding(
                    "POLICY-ARTIFACT",
                    str(manifest_path),
                    "each policy must be an object",
                )
            )
            continue
        route = str(policy.get("route", "unknown"))
        location = f"{manifest_path}:{route}"
        for field in (
            "source",
            "artifact",
            "version",
            "effectiveDate",
            "approvalStatus",
            "approvedBy",
        ):
            if not policy.get(field):
                findings.append(
                    Finding("POLICY-METADATA", location, f"missing {field}")
                )
        for field in ("source", "artifact"):
            value = policy.get(field)
            if not value:
                continue
            path = root / value
            if not path.is_file():
                findings.append(
                    Finding(
                        "POLICY-ARTIFACT", location, f"missing {field} file: {value}"
                    )
                )
                continue
            digest_field = f"{field}Digest"
            if policy.get(digest_field) != file_digest(path):
                findings.append(
                    Finding("POLICY-DRIFT", location, f"{field} digest is stale")
                )
            if field == "source" and re.search(
                r"\[(?:TO BE CONFIRMED|TBD|TODO)\]|\bFIXME\b",
                path.read_text(),
                re.IGNORECASE,
            ):
                findings.append(
                    Finding(
                        "POLICY-PLACEHOLDER",
                        location,
                        "policy contains an unresolved placeholder",
                    )
                )
        if (
            policy.get("approvalStatus") != "Approved"
            or "pending" in str(policy.get("approvedBy", "")).lower()
        ):
            findings.append(
                Finding("POLICY-APPROVAL", location, "policy is not approved")
            )
        effective = str(policy.get("effectiveDate", ""))
        if "subject to" in effective.lower() or "confirm" in effective.lower():
            findings.append(
                Finding(
                    "POLICY-METADATA",
                    location,
                    "effective date is conditional or unresolved",
                )
            )
        claims = policy.get("presentTenseClaims", [])
        if not isinstance(claims, list) or not claims:
            findings.append(
                Finding(
                    "POLICY-EVIDENCE", location, "no control claims are inventoried"
                )
            )
            claims = []
        for claim in claims:
            if not isinstance(claim, dict):
                findings.append(
                    Finding(
                        "POLICY-EVIDENCE",
                        location,
                        "each control claim must be an object",
                    )
                )
                continue
            control = str(claim.get("control", "unidentified"))
            evidence = claim.get("evidence", [])
            if not isinstance(evidence, list):
                evidence = []
            if claim.get("status") != "approved" or not evidence:
                findings.append(
                    Finding(
                        control,
                        location,
                        "claim lacks approved implementation/test evidence",
                    )
                )
            for item in evidence:
                if not isinstance(item, dict) or (
                    item.get("status") != "approved"
                    or item.get("policyVersion") != policy.get("version")
                    or not evidence_reference_exists(root, item.get("reference"))
                ):
                    findings.append(
                        Finding(
                            control,
                            location,
                            "claim evidence is stale, incomplete, or version-mismatched",
                        )
                    )

    controls = manifest.get("releaseControls", {})
    if not isinstance(controls, dict):
        findings.append(
            Finding(
                "POLICY-EVIDENCE",
                str(manifest_path),
                "releaseControls must be an object",
            )
        )
        controls = {}
    for name in ("scannerProduction", "videoProductionDisabled"):
        control = controls.get(name, {})
        if not isinstance(control, dict):
            control = {}
        evidence = control.get("evidence", [])
        if (
            control.get("status") != "approved"
            or not isinstance(evidence, list)
            or not evidence
            or not all(evidence_reference_exists(root, item) for item in evidence)
        ):
            findings.append(
                Finding(
                    name, str(manifest_path), "release control lacks approved evidence"
                )
            )
    return findings


def check_p0_register(root: Path, register_path: Path) -> list[Finding]:
    findings: list[Finding] = []
    try:
        lines = register_path.read_text().splitlines()
    except OSError as error:
        return [
            Finding("P0-REGISTER", str(register_path), f"register unreadable: {error}")
        ]
    p0_count = 0
    for line_number, line in enumerate(lines, start=1):
        cells = [cell.strip() for cell in line.strip().strip("|").split("|")]
        if len(cells) < 5 or cells[1] != "P0":
            continue
        p0_count += 1
        control, evidence = cells[0], cells[4]
        references = re.findall(r"\[[^]]+]\(([^)]+)\)|`([^`]+)`", evidence)
        linked = [first or second for first, second in references]
        if (
            not evidence.startswith("Closed:")
            or not linked
            or not all(evidence_reference_exists(root, item) for item in linked)
        ):
            findings.append(
                Finding(
                    control,
                    f"{register_path}:{line_number}",
                    "P0 is open or lacks linked approved evidence",
                )
            )
    if p0_count == 0:
        findings.append(
            Finding("P0-REGISTER", str(register_path), "no P0 controls were found")
        )
    return findings


def check_no_ozone(root: Path) -> list[Finding]:
    findings: list[Finding] = []
    go_mod = root / "appview" / "go.mod"
    if go_mod.is_file() and re.search(
        r"(?im)^\s*(?:require\s+)?\S*ozone\S*\s+v", go_mod.read_text()
    ):
        findings.append(
            Finding(
                "RULE-008",
                str(go_mod),
                "Ozone dependency is forbidden in this architecture slice",
            )
        )
    safety_root = root / "appview" / "internal" / "safetyincident"
    for path in safety_root.glob("*.go") if safety_root.is_dir() else []:
        if re.search(r'(?i)["/]ozone(?:["/]|$)', path.read_text()):
            findings.append(
                Finding(
                    "RULE-008",
                    str(path),
                    "Ozone-dependent safety behavior is forbidden",
                )
            )
    return findings


def evaluate(root: Path, manifest: Path, register: Path) -> list[Finding]:
    return (
        check_manifest(root, manifest)
        + check_p0_register(root, register)
        + check_no_ozone(root)
    )


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Fail-closed CraftSky public-release readiness gate"
    )
    parser.add_argument(
        "--root", type=Path, default=Path(__file__).resolve().parents[1]
    )
    parser.add_argument("--manifest", type=Path)
    parser.add_argument("--register", type=Path)
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()
    root = args.root.resolve()
    manifest = (
        args.manifest or root / "web" / "policy-publication-manifest.json"
    ).resolve()
    register = (
        args.register
        or root / "online-safety" / "registers" / "compliance-gap-register.md"
    ).resolve()
    findings = evaluate(root, manifest, register)
    if args.json:
        print(
            json.dumps(
                {
                    "ready": not findings,
                    "findings": [finding.__dict__ for finding in findings],
                },
                indent=2,
            )
        )
    elif findings:
        print("online-safety-readiness: BLOCKED")
        for finding in findings:
            print(f"{finding.control}\t{finding.location}\t{finding.message}")
    else:
        print("online-safety-readiness: READY")
    return 1 if findings else 0


if __name__ == "__main__":
    raise SystemExit(main())
