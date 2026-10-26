import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/rich_text/faceted_text_model.dart';
import 'package:material_ui/material_ui.dart';

/// Launches a URL for a link facet. Tests override this seam.
typedef FacetUrlLauncher = ExternalLinkLauncher;

/// Dispatches supported rich-text facet tap actions safely.
class FacetActionHandler {
  /// Creates a facet action handler.
  const new({
    required this.launchUrl,
    this.confirmOpenLink = showOpenLinkDialog,
  });

  /// Injectable URL launcher callback.
  final FacetUrlLauncher launchUrl;

  /// Injectable confirmation callback for external links.
  final ExternalLinkConfirmer confirmOpenLink;

  /// Handles a tap for [feature] whose visible substring is [visibleText].
  Future<void> handle(
    BuildContext context, {
    required FacetFeature feature,
    required String visibleText,
  }) async {
    try {
      switch (feature) {
        case MentionFacetFeature(:final did):
          await UserProfileRoute(did: Did.parse(did)).push<void>(context);
        case LinkFacetFeature(uri: final uriText):
          final uri = normalizeExternalLinkUri(uriText);
          if (uri == null) return;
          await confirmAndLaunchExternalLink(
            context,
            uri: uri,
            launchUrl: launchUrl,
            confirmOpenLink: confirmOpenLink,
          );
        case TagFacetFeature(:final tag):
          if (tag.isEmpty) return;
          await TagSearchRoute(tag: tag).push<void>(context);
      }
    } on Object {
      // Destination failures must not crash rendered text surfaces.
    }
  }
}
