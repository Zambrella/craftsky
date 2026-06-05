import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/rich_text/faceted_text_model.dart';
import 'package:flutter/material.dart';

/// Launches a URL for a link facet. Tests override this seam.
typedef FacetUrlLauncher = Future<bool> Function(Uri uri);

/// Dispatches supported rich-text facet tap actions safely.
class FacetActionHandler {
  /// Creates a facet action handler.
  const FacetActionHandler({required this.launchUrl});

  /// Injectable URL launcher callback.
  final FacetUrlLauncher launchUrl;

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
          if (uriText.isEmpty) return;
          final uri = Uri.tryParse(uriText);
          if (uri == null) return;
          await launchUrl(uri);
        case TagFacetFeature(:final tag):
          if (tag.isEmpty) return;
          await SearchRoute(tag: tag).push<void>(context);
      }
    } on Object {
      // Destination failures must not crash rendered text surfaces.
    }
  }
}

String? _visibleMentionHandle(String visibleText) {
  if (!visibleText.startsWith('@')) return null;
  final handle = visibleText.substring(1);
  final valid = RegExp(r'^[A-Za-z0-9][A-Za-z0-9.-]*\.[A-Za-z][A-Za-z0-9-]*$');
  return valid.hasMatch(handle) ? handle : null;
}
