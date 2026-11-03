"""Extract the pinned Flutter Phosphor outlines; no icon font/runtime is shipped.

Run from the checkout: uv run --with fonttools python web-client/scripts/generate-native-icons.py
Requires `fvm flutter pub get` in app/ so package_config.json exists.
"""
import json
import re
from pathlib import Path
from urllib.parse import unquote, urljoin, urlparse

from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.pens.transformPen import TransformPen
from fontTools.ttLib import TTFont

root = Path(__file__).resolve().parents[2]
config = root / "app/.dart_tool/package_config.json"
package = next(p for p in json.loads(config.read_text())["packages"] if p["name"] == "phosphor_icons")
source = Path(unquote(urlparse(urljoin(config.as_uri(), package["rootUri"])).path))

# Semantic names used by the web map to CraftskyIcons/SettingsRow leading glyphs.
aliases = {
    "home": "house", "search": "magnifyingGlass", "messages": "chatCircle",
    "activity": "bell", "profile": "user", "saved": "bookmarks", "scheduled": "clock",
    "drafts": "notePencil", "settings": "gear", "menu": "list", "plus": "plus",
    "close": "x", "clear": "xCircle", "instagramMatch": "userFocus", "back": "arrowLeft", "chevron": "caretRight", "down": "caretDown",
    "mute": "speakerSlash", "block": "prohibit", "report": "flag", "copy": "link",
    "heart": "heart", "comment": "chatCircle", "share": "shareNetwork", "repost": "repeat",
    "more": "dotsThree", "refresh": "arrowsClockwise", "filter": "funnelSimple",
    "sort": "sortDescending", "oldest": "sortAscending", "folder": "folder", "check": "check", "edit": "pencilSimple",
    "external": "arrowSquareOut", "link": "link", "pin": "pushPin", "shield": "shieldCheck",
    "calendar": "calendarBlank", "image": "imageSquare", "mail": "envelope",
    "location": "mapPin", "logout": "signOut", "moon": "moon", "globe": "globe",
    "info": "info", "lock": "lockKey", "business": "storefront", "star": "sparkle",
    "clock": "clock", "video": "videoCamera", "camera": "camera", "send": "paperPlaneRight", "trash": "trash",
    "download": "downloadSimple", "warning": "warning", "appearance": "circleHalf",
    "customisation": "palette", "languages": "globe", "translate": "translate",
    "notifications": "bell", "messaging": "chatCircle", "privacy": "lockKey",
    "errorReporting": "bug", "growth": "trendUp", "followers": "users",
    "following": "userPlus", "muted": "bellSlash", "blocked": "prohibit",
    "instagram": "instagramLogo", "products": "storefront", "events": "calendar",
    "moderation": "shieldCheck", "account": "userCircle", "about": "info",
    "feedback": "chatTeardropText", "switch": "userSwitch", "subscription": "sparkle",
    "system": "sunDim", "light": "sun", "dark": "moon", "delete": "trash",
    "like": "heart", "follow": "userPlus", "reply": "chatCircle", "mention": "at",
    "quote": "quotes", "document": "fileText", "cache": "broom",
    "currency": "currencyDollar", "pattern": "bookOpen", "book": "bookOpen",
    "palette": "palette", "adjustments": "slidersHorizontal",
    "textPost": "textAlignLeft", "projectPost": "cards", "moveUp": "arrowUp",
}
filled = {aliases[name] for name in ("home", "search", "messages", "activity", "profile", "saved", "scheduled", "drafts", "settings", "heart", "like", "subscription", "follow", "quote", "mention", "instagramMatch", "repost", "comment", "shield")}
bold = {"userPlus", "textAlignLeft", "cards", "arrowUp", "trash", "broom", "x", "shareNetwork", "speakerSlash", "prohibit", "flag", "chatCircle", "pushPin"}
paths = {}
for weight, filename in [("regular", "Phosphor.ttf"), ("fill", "Phosphor-Fill.ttf"), ("bold", "Phosphor-Bold.ttf")]:
    font = TTFont(source / "lib/fonts" / filename)
    if font["head"].unitsPerEm != 1024 or font["hhea"].ascent != 960:
        raise RuntimeError("Review changed Phosphor font metrics before regenerating")
    dart = (source / f"lib/src/phosphor_icons_{weight}.dart").read_text()
    codepoints = {name: int(code, 16) for name, code in re.findall(r"static const (\w+) = IconData\(0x([0-9a-f]+)", dart)}
    cmap, glyphs = font.getBestCmap(), font.getGlyphSet()
    for name in sorted(set(aliases.values())):
        if (weight == "fill" and name not in filled) or (weight == "bold" and name not in bold):
            continue
        pen = SVGPathPen(glyphs)
        glyphs[cmap[codepoints[name]]].draw(TransformPen(pen, (1, 0, 0, -1, 0, 960)))
        paths.setdefault(name, {})[weight] = pen.getCommands()

target = root / "web-client/src/lib/features/shell/native-icons.ts"
target.write_text(
    "// Generated from the Flutter app's pinned Phosphor Icons; MIT license in static/craft_icons/PHOSPHOR-LICENSE.txt.\n"
    "// Regenerate with web-client/scripts/generate-native-icons.py; SVG coordinates use a 1024-unit viewBox.\n"
    "export const nativeIconAliases: Record<string, string> = " + json.dumps(aliases, indent=2) + ";\n"
    "export const nativeIconPaths: Record<string, {regular: string; fill?: string; bold?: string}> = " + json.dumps(paths, indent=2) + ";\n"
)
(root / "web-client/static/craft_icons/PHOSPHOR-LICENSE.txt").write_text((source / "LICENSE").read_text())
print(f"Extracted {len(paths)} native glyphs and {len(filled)} selected states")
