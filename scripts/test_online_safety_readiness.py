import hashlib
import json
import tempfile
import unittest
from pathlib import Path

import online_safety_readiness as readiness


ROUTES = ["/terms", "/privacy", "/community-guidelines", "/reporting", "/copyright"]


class ReadinessTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        (self.root / "appview" / "internal" / "safetyincident").mkdir(parents=True)
        (self.root / "appview" / "go.mod").write_text("module example.test\n")
        (self.root / "policies").mkdir()
        for reference in (
            "tests/control",
            "external/provider-validation",
            "tests/video-gate",
            "evidence/approval.md",
        ):
            path = self.root / reference
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text("approved evidence\n")
        policies = []
        for index, route in enumerate(ROUTES):
            source = self.root / "policies" / f"source-{index}.md"
            artifact = self.root / "policies" / f"artifact-{index}.html"
            source.write_text(f"policy {route}\n")
            artifact.write_text(f"<h1>{route}</h1>\n")
            digest = lambda path: (
                "sha256:" + hashlib.sha256(path.read_bytes()).hexdigest()
            )
            policies.append(
                {
                    "route": route,
                    "source": str(source.relative_to(self.root)),
                    "artifact": str(artifact.relative_to(self.root)),
                    "version": "1.0",
                    "effectiveDate": "2030-01-01",
                    "approvalStatus": "Approved",
                    "approvedBy": "Accountable owner",
                    "sourceDigest": digest(source),
                    "artifactDigest": digest(artifact),
                    "presentTenseClaims": [
                        {
                            "control": "CONTROL-1",
                            "status": "approved",
                            "evidence": [
                                {
                                    "reference": "tests/control",
                                    "status": "approved",
                                    "policyVersion": "1.0",
                                }
                            ],
                        }
                    ],
                }
            )
        self.manifest = self.root / "manifest.json"
        self.manifest.write_text(
            json.dumps(
                {
                    "publicationStatus": "approved",
                    "policies": policies,
                    "releaseControls": {
                        "scannerProduction": {
                            "status": "approved",
                            "evidence": ["external/provider-validation"],
                        },
                        "videoProductionDisabled": {
                            "status": "approved",
                            "evidence": ["tests/video-gate"],
                        },
                    },
                }
            )
        )
        self.register = self.root / "register.md"
        self.register.write_text(
            "| ID | Priority | Gap | Required outcome | Evidence/status |\n|---|---|---|---|---|\n| SAFE-01 | P0 | x | y | Closed: [approval](evidence/approval.md) |\n"
        )

    def tearDown(self) -> None:
        self.temp.cleanup()

    def evaluate(self):
        return readiness.evaluate(self.root, self.manifest, self.register)

    def test_approved_current_complete_artifact_passes(self):
        self.assertEqual([], self.evaluate())

    def test_open_p0_blocks_release_without_exposing_restricted_data(self):
        self.register.write_text(
            "| ID | Priority | Gap | Required outcome | Evidence/status |\n|---|---|---|---|---|\n| SAFE-01 | P0 | restricted details omitted | y | Review pending |\n"
        )
        findings = self.evaluate()
        self.assertTrue(any(item.control == "SAFE-01" for item in findings))
        self.assertFalse(any("restricted details" in item.message for item in findings))

    def test_missing_or_stale_policy_evidence_blocks_publication(self):
        manifest = json.loads(self.manifest.read_text())
        manifest["policies"][0]["presentTenseClaims"][0]["evidence"] = []
        manifest["policies"][1]["presentTenseClaims"][0]["evidence"][0][
            "policyVersion"
        ] = "0.9"
        self.manifest.write_text(json.dumps(manifest))
        findings = self.evaluate()
        self.assertGreaterEqual(
            sum(item.control == "CONTROL-1" for item in findings), 2
        )

    def test_incomplete_policy_set_blocks_publication(self):
        manifest = json.loads(self.manifest.read_text())
        manifest["policies"].pop()
        self.manifest.write_text(json.dumps(manifest))
        self.assertTrue(
            any(item.control == "POLICY-ARTIFACT" for item in self.evaluate())
        )

    def test_malformed_manifest_structure_fails_closed(self):
        self.manifest.write_text(
            json.dumps(
                {
                    "publicationStatus": "approved",
                    "policies": ["not-a-policy"],
                    "releaseControls": [],
                }
            )
        )
        findings = self.evaluate()
        self.assertTrue(any(item.control == "POLICY-ARTIFACT" for item in findings))
        self.assertTrue(any(item.control == "POLICY-EVIDENCE" for item in findings))

    def test_ozone_dependency_is_rejected(self):
        (self.root / "appview" / "go.mod").write_text(
            "module example.test\nrequire example.com/ozone v1.0.0\n"
        )
        self.assertTrue(any(item.control == "RULE-008" for item in self.evaluate()))

    def test_nonexistent_evidence_references_fail_closed(self):
        manifest = json.loads(self.manifest.read_text())
        manifest["policies"][0]["presentTenseClaims"][0]["evidence"][0]["reference"] = (
            "tests/missing-control"
        )
        manifest["releaseControls"]["scannerProduction"]["evidence"] = [
            "external/missing-provider-validation"
        ]
        self.manifest.write_text(json.dumps(manifest))
        self.register.write_text(
            "| ID | Priority | Gap | Required outcome | Evidence/status |\n"
            "|---|---|---|---|---|\n"
            "| SAFE-01 | P0 | x | y | Closed: [approval](evidence/missing.md) |\n"
        )

        findings = self.evaluate()

        self.assertTrue(any(item.control == "CONTROL-1" for item in findings))
        self.assertTrue(any(item.control == "scannerProduction" for item in findings))
        self.assertTrue(any(item.control == "SAFE-01" for item in findings))


if __name__ == "__main__":
    unittest.main()
