import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/rich_text/facet_syntax.dart';
import 'package:craftsky_app/shared/rich_text/faceted_text_model.dart';
import 'package:flutter/material.dart';

/// Launches a URL for a link facet. Tests override this seam.
typedef FacetUrlLauncher = ExternalLinkLauncher;

/// Dispatches supported rich-text facet tap actions safely.
class FacetActionHandler {
  /// Creates a facet action handler.
  const FacetActionHandler({
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
        case MentionFacetFeature():
          final handle = _visibleMentionHandle(visibleText);
          if (handle == null) return;
          await UserProfileRoute(handle: handle).push<void>(context);
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

String? _visibleMentionHandle(String visibleText) {
  if (!visibleText.startsWith('@')) return null;
  final handle = visibleText.substring(1);
  return isValidMentionHandle(handle) ? handle : null;
}
