"""Publish one public app profile using the maintainer's authenticated goat CLI."""
from __future__ import annotations

import argparse
import base64
import copy
from datetime import datetime, timezone
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import urllib.error
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
DIRECTORY = ROOT / "metadata/app-profile"
DID = "did:plc:lmmx63zcns6gewgxqfdt4kof"
COLLECTION = "community.lexicon.app.profile"
URI = f"at://{DID}/{COLLECTION}/self"
ICON = ROOT / "app/assets/app-icon/app-icon-light.png"


class PublicationError(Exception):
    """A bounded operator explanation; dependency details stay in the cause."""


def goat(*args: str, body: dict | None = None) -> str:
    try:
        result = subprocess.run(
            ["goat", "--log-level", "error", *args],
            input=json.dumps(body) if body is not None else None,
            capture_output=True, text=True, check=True, timeout=90,
        )
    except (OSError, subprocess.SubprocessError) as cause:
        raise PublicationError("goat operation failed; check CLI installation and authentication.") from cause
    return result.stdout


def read(url: str) -> dict:
    request = urllib.request.Request(url, headers={"Accept": "application/json"})
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            return json.load(response)
    except urllib.error.HTTPError:
        raise
    except (OSError, ValueError) as cause:
        raise PublicationError("Public record lookup failed.") from cause


def official_pds() -> str:
    try:
        identity = read("https://plc.directory/" + DID)
    except urllib.error.HTTPError as cause:
        cause.close()
        raise PublicationError("Official identity lookup failed.") from cause
    if (not isinstance(identity, dict) or identity.get("id") != DID
            or "at://craftsky.social" not in identity.get("alsoKnownAs", [])):
        raise PublicationError("Official identity no longer matches craftsky.social.")
    endpoints = [service.get("serviceEndpoint") for service in identity.get("service", [])
                 if isinstance(service, dict)
                 and service.get("id") in ("#atproto_pds", DID + "#atproto_pds")
                 and service.get("type") == "AtprotoPersonalDataServer"]
    if len(endpoints) != 1 or not isinstance(endpoints[0], str):
        raise PublicationError("Official identity has no unique PDS endpoint.")
    url = urllib.parse.urlsplit(endpoints[0])
    if url.scheme != "https" or not url.hostname or url.username or url.password or url.query or url.fragment:
        raise PublicationError("Official PDS endpoint must be an HTTPS origin.")
    if url.path not in ("", "/"):
        raise PublicationError("Official PDS endpoint must be an HTTPS origin.")
    return endpoints[0].rstrip("/")


def current_record(pds: str) -> dict | None:
    params = urllib.parse.urlencode({"repo": DID, "collection": COLLECTION, "rkey": "self"})
    try:
        record = read(pds + "/xrpc/com.atproto.repo.getRecord?" + params)
    except urllib.error.HTTPError as cause:
        # Only a typed missing-record result permits a create. Outages must not
        # silently turn an update into an unconditional overwrite.
        try:
            with cause:
                error = json.load(cause)
        except (ValueError, OSError) as decode_cause:
            raise PublicationError("Public record lookup returned an invalid error.") from decode_cause
        if isinstance(error, dict) and cause.code in (400, 404) and error.get("error") == "RecordNotFound":
            return None
        raise PublicationError("Public record lookup failed; no publication attempted.") from cause
    if (not isinstance(record, dict) or record.get("uri") != URI or not isinstance(record.get("cid"), str)
            or not record["cid"] or not isinstance(record.get("value"), dict)
            or record["value"].get("$type") != COLLECTION):
        raise PublicationError("Public record response does not match the publication target.")
    return record


def load_source() -> dict:
    value = json.loads((DIRECTORY / "profile.json").read_text())
    if not isinstance(value, dict) or value.get("$type") != COLLECTION or "updatedAt" in value:
        raise PublicationError("Profile source must use the app profile type and omit managed updatedAt.")
    images = value.get("images", [])
    if not isinstance(images, list):
        raise PublicationError("Profile images must be an array.")
    for image in images:
        if not isinstance(image, dict) or ("image" in image) == ("uri" in image):
            raise PublicationError("Each profile image must have exactly one blob or remote URI.")
    if len(images) != 1 or images[0].get("image") != icon_blob():
        raise PublicationError("Profile icon reference differs from the current app icon; update its CID and size.")
    return value


def icon_blob() -> dict:
    data = ICON.read_bytes()
    # CIDv1 / raw / sha2-256 is the AT Protocol blob content identifier.
    cid = "b" + base64.b32encode(b"\x01\x55\x12\x20" + hashlib.sha256(data).digest()).decode().lower().rstrip("=")
    return {"$type": "blob", "ref": {"$link": cid}, "mimeType": "image/png", "size": len(data)}


def profile_fields() -> set[str]:
    pins = json.loads((DIRECTORY / "schema-pins.json").read_text())
    pin = next(pin for pin in pins if pin["nsid"] == COLLECTION)
    schema = json.loads((DIRECTORY / pin["file"]).read_text())
    return set(schema["defs"]["main"]["record"]["properties"])


def desired_record(source: dict, existing: dict | None) -> dict:
    value = copy.deepcopy(existing["value"]) if existing else {}
    # Own the pinned schema's metadata fields, but retain future extensions.
    for field in profile_fields() - {"createdAt", "updatedAt"}:
        value.pop(field, None)
    created = value.get("createdAt", source["createdAt"])
    value.update(copy.deepcopy(source))
    value["createdAt"] = created
    return value


def validate(value: dict) -> None:
    with tempfile.TemporaryDirectory(prefix="craftsky-app-profile-") as temporary:
        path = Path(temporary) / "profile.json"
        path.write_text(json.dumps(value, ensure_ascii=False) + "\n")
        goat("lex", "validate", "--catalog", str(DIRECTORY / "schemas"), str(path))


def authenticated_owner() -> None:
    result = goat("account", "check-auth")
    match = re.search(r"^DID: (\S+)$", result, re.MULTILINE)
    if not match or match[1] != DID:
        raise PublicationError("goat must be authenticated as the official craftsky.social DID.")


def verify(pds: str, expected: dict) -> dict:
    actual = current_record(pds)
    if actual is None or actual["value"] != expected:
        raise PublicationError("Public profile differs from the expected record; inspect before retrying.")
    return actual


def publish(source: dict, pds: str, existing: dict | None) -> dict:
    authenticated_owner()
    value = desired_record(source, existing)
    if existing and value == existing["value"]:
        return verify(pds, value)
    if existing:
        value["updatedAt"] = datetime.now(timezone.utc).isoformat(timespec="seconds").replace("+00:00", "Z")
    validate(value)
    if not existing or not any(image.get("image") == icon_blob() for image in existing["value"].get("images", [])):
        uploaded = json.loads(goat("blob", "upload", str(ICON)))
        if not isinstance(uploaded, dict) or uploaded.get("blob", uploaded) != icon_blob():
            raise PublicationError("PDS returned an unexpected icon blob; profile was not written.")
    try:
        result = json.loads(goat("xrpc", "procedure", "@pds", "com.atproto.repo.putRecord", "-", body={
            "repo": DID, "collection": COLLECTION, "rkey": "self", "record": value,
            # The protocol default validates schemas known to the PDS. Some
            # PDSes do not resolve external schemas; local validation above is
            # mandatory regardless, and we never send validate=false.
            "swapRecord": existing["cid"] if existing else None,
        }))
    except (PublicationError, ValueError) as cause:
        raise PublicationError("Profile write failed; public state may have changed. Run verify before retrying.") from cause
    if not isinstance(result, dict) or result.get("uri") != URI or not result.get("cid"):
        raise PublicationError("Unexpected publication response; verify public state before retrying.")
    actual = verify(pds, value)
    if actual["cid"] != result["cid"]:
        raise PublicationError("Public profile changed during verification; inspect before retrying.")
    return actual


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("show", "validate", "publish", "verify"))
    args = parser.parse_args(argv)
    try:
        source = load_source()
        validate(source)
        if args.action == "show":
            print(json.dumps({"uri": URI, "record": source}, indent=2, ensure_ascii=False))
        elif args.action == "validate":
            print("Validated app profile and current icon against pinned community schemas.")
        else:
            pds = official_pds()
            existing = current_record(pds)
            if args.action == "verify":
                actual = verify(pds, desired_record(source, existing))
                print("Verified:", URI, actual["cid"])
            else:
                actual = publish(source, pds, existing)
                print("Unchanged:" if actual == existing else "Published:", URI, actual["cid"])
    except (PublicationError, OSError, ValueError, KeyError, TypeError, StopIteration) as cause:
        # Never echo CLI stderr, HTTP bodies or dependency exception prose.
        message = str(cause) if isinstance(cause, PublicationError) else "Invalid publication data or failed lookup."
        print("app-profile-publication:", message, file=sys.stderr)
        return 1
    return 0
