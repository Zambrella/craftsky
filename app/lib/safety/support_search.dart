import 'package:craftsky_app/safety/guide_data.dart';

// This only selects an optional on-device notice. It is not an offence,
// diagnosis, moderation decision or persisted attribute of a person/query.
bool shouldOfferSearchSupport(String query) {
  final normalized = query
      .toLowerCase()
      .replaceAll(RegExp(r'[-_\u2010-\u2015]'), ' ')
      .replaceAll(RegExp(r'\s+'), ' ')
      .trim();
  return (safetyGuide['searchTerms'] as List).cast<String>().any(
    (term) =>
        RegExp('(^|[^a-z0-9])${RegExp.escape(term)}([^a-z0-9]|\$)')
            .hasMatch(normalized),
  );
}
