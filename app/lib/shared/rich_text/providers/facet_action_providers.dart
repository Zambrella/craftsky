import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/rich_text/facet_action_handler.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart' as url_launcher;

/// URL launcher seam for rendered link facets.
final facetUrlLauncherProvider = Provider<FacetUrlLauncher>(
  (ref) =>
      (uri) => url_launcher.launchUrl(
        uri,
        mode: url_launcher.LaunchMode.externalApplication,
      ),
);

/// Confirmation dialog seam for rendered link facets.
final facetLinkConfirmerProvider = Provider<ExternalLinkConfirmer>(
  (ref) => showOpenLinkDialog,
);

/// Handler for rendered facet taps.
final facetActionHandlerProvider = Provider<FacetActionHandler>(
  (ref) => FacetActionHandler(
    launchUrl: ref.watch(facetUrlLauncherProvider),
    confirmOpenLink: ref.watch(facetLinkConfirmerProvider),
  ),
);
