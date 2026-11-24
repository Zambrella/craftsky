import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/projects/options/project_option_catalogs.dart';
import 'package:craftsky_app/projects/options/project_taxonomy.g.dart';
import 'package:flutter/foundation.dart';

/// Saved craft IDs and their display labels, including previously offered IDs.
enum Craft {
  sewing('sewing'),
  quilting('quilting'),
  knitting('knitting'),
  crochet('crochet'),
  embroidery('embroidery'),
  crossStitch('cross-stitch'),
  weaving('weaving'),
  spinning('spinning'),
  dyeing('dyeing'),
  punchNeedle('punch-needle'),
  felting('felting'),
  macrame('macrame'),
  pottery('pottery'),
  woodworking('woodworking'),
  leatherwork('leatherwork'),
  jewellery('jewellery'),
  bookbinding('bookbinding'),
  calligraphy('calligraphy'),
  printmaking('printmaking'),
  papercraft('papercraft'),
  painting('painting'),
  drawing('drawing'),
  candleMaking('candlemaking'),
  soapMaking('soapmaking');

  new(this.id);

  /// Wire identifier persisted on the profile and round-tripped to the
  /// AppView. Stable across releases — renaming would orphan stored
  /// data.
  final String id;

  /// Resolves a wire id back to its enum. Returns `null` for unknown
  /// tags — callers should preserve unknowns rather than dropping them
  /// (the user might be on a newer build that's added a tag we don't
  /// know about yet).
  static Craft? fromId(String id) {
    final normalized = ProjectOptionCatalogs.profileIdForCraftToken(id);
    for (final craft in values) {
      if (craft.id == normalized) return craft;
    }
    return null;
  }
}

/// Crafts currently offered as profile and onboarding choices.
///
/// Keep this separate from [Craft.values]: the full enum remains the wire
/// compatibility catalog for profiles saved by earlier app versions.
final canonicalSelectableCrafts = List<Craft>.unmodifiable([
  for (final entry in ProjectTaxonomy.entries)
    if (entry.family == 'craft' &&
        (entry.phase ?? 1) <= ProjectTaxonomy.releasePhase &&
        entry.profileId != null)
      ?Craft.fromId(entry.profileId!),
]);

bool isCanonicalSelectableCraft(String id) =>
    canonicalSelectableCrafts.contains(Craft.fromId(id));

List<String> selectableCraftIdsFrom(Iterable<String> stored) => [
  for (final craft in stored.map(Craft.fromId).toSet())
    if (craft != null && canonicalSelectableCrafts.contains(craft)) craft.id,
];

/// Keep saved spellings for retained interests and round-trip hidden values.
List<String> storedCraftSelection(
  List<String> baseline,
  List<String> selected,
) {
  if (listEquals(selectableCraftIdsFrom(baseline), selected)) {
    return List.of(baseline);
  }

  final retained = <String, String>{};
  final hidden = <String>[];
  for (final value in baseline) {
    final craft = Craft.fromId(value);
    if (craft != null && canonicalSelectableCrafts.contains(craft)) {
      retained.putIfAbsent(craft.id, () => value);
    } else {
      hidden.add(value);
    }
  }
  return [for (final id in selected) retained[id] ?? id, ...hidden];
}

/// Localised display label for [craft]. Falls back to the wire id if
/// the catalog ever drifts ahead of the ARB — a safer fail than
/// crashing on a missing key.
String craftLabel(Craft craft, AppLocalizations l10n) {
  return switch (craft) {
    Craft.sewing => l10n.craftSewing,
    Craft.quilting => l10n.craftQuilting,
    Craft.knitting => l10n.craftKnitting,
    Craft.crochet => l10n.craftCrochet,
    Craft.embroidery => l10n.craftEmbroidery,
    Craft.crossStitch => l10n.craftCrossStitch,
    Craft.weaving => l10n.craftWeaving,
    Craft.spinning => l10n.craftSpinning,
    Craft.dyeing => l10n.craftDyeing,
    Craft.punchNeedle => l10n.craftPunchNeedle,
    Craft.felting => l10n.craftFelting,
    Craft.macrame => l10n.craftMacrame,
    Craft.pottery => l10n.craftPottery,
    Craft.woodworking => l10n.craftWoodworking,
    Craft.leatherwork => l10n.craftLeatherwork,
    Craft.jewellery => l10n.craftJewellery,
    Craft.bookbinding => l10n.craftBookbinding,
    Craft.calligraphy => l10n.craftCalligraphy,
    Craft.printmaking => l10n.craftPrintmaking,
    Craft.papercraft => l10n.craftPapercraft,
    Craft.painting => l10n.craftPainting,
    Craft.drawing => l10n.craftDrawing,
    Craft.candleMaking => l10n.craftCandleMaking,
    Craft.soapMaking => l10n.craftSoapMaking,
  };
}
