import 'package:craftsky_app/l10n/generated/app_localizations.dart';

/// Canonical list of craft tags users can pick on their profile.
///
/// The wire format is the lowercase [id] — what the AppView stores in
/// `Profile.crafts`. The UI renders each tag through [craftLabel] so a
/// French viewer sees "Tricot" while the underlying tag stays
/// `'knitting'`. Adding a new craft here is the only place callers need
/// to touch (plus a matching `craft*` key in `app_en.arb`).
///
/// Order is the order they appear in the picker — grouped loosely by
/// "fibre & textile", "ceramics & wood", "paper & ink", "other" so the
/// list scans visually rather than alphabetically.
enum Craft {
  sewing('sewing'),
  quilting('quilting'),
  knitting('knitting'),
  crochet('crochet'),
  embroidery('embroidery'),
  crossStitch('cross-stitch'),
  weaving('weaving'),
  spinning('spinning'),
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

  const Craft(this.id);

  /// Wire identifier persisted on the profile and round-tripped to the
  /// AppView. Stable across releases — renaming would orphan stored
  /// data.
  final String id;

  /// Resolves a wire id back to its enum. Returns `null` for unknown
  /// tags — callers should preserve unknowns rather than dropping them
  /// (the user might be on a newer build that's added a tag we don't
  /// know about yet).
  static Craft? fromId(String id) {
    for (final craft in values) {
      if (craft.id == id) return craft;
    }
    return null;
  }
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
