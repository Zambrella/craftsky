import 'dart:async';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

enum ExternalCardVariant { full, compact }

final externalCardLauncherProvider = Provider<ExternalLinkLauncher>(
  (_) => launchExternalLink,
);

class ExternalCard extends ConsumerWidget {
  const ExternalCard({
    required this.external,
    this.variant = ExternalCardVariant.full,
    this.launchUrl,
    super.key,
  });

  final PostExternal external;
  final ExternalCardVariant variant;
  final ExternalLinkLauncher? launchUrl;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final uri = normalizeExternalLinkUri(external.uri);
    final host = uri == null ? '' : _displayHost(uri);
    final theme = Theme.of(context);
    final radii = theme.extension<RadiusTheme>()!;
    final label = AppLocalizations.of(context).externalCardOpen(host);
    final launcher = switch (launchUrl) {
      final launch? => launch,
      null => ref.watch(externalCardLauncherProvider),
    };

    return Semantics(
      button: uri != null,
      label: label,
      child: Material(
        color: Colors.transparent,
        clipBehavior: Clip.antiAlias,
        borderRadius: BorderRadius.circular(radii.r2),
        child: InkWell(
          borderRadius: BorderRadius.circular(radii.r2),
          onTap: uri == null
              ? null
              : () => unawaited(
                  confirmAndLaunchExternalLink(
                    context,
                    uri: uri,
                    launchUrl: launcher,
                  ),
                ),
          child: Ink(
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(radii.r2),
              color: theme.colorScheme.surfaceContainerLow,
            ),
            child: DecoratedBox(
              key: const Key('external-card-outline'),
              position: DecorationPosition.foreground,
              decoration: BoxDecoration(
                border: Border.all(color: theme.colorScheme.outlineVariant),
                borderRadius: BorderRadius.circular(radii.r2),
              ),
              child: LayoutBuilder(
                builder: (context, constraints) => _layout(
                  context,
                  ref,
                  host,
                  narrow: constraints.maxWidth < 320,
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _layout(
    BuildContext context,
    WidgetRef ref,
    String host, {
    required bool narrow,
  }) {
    final thumbnail = external.thumb;
    final copy = _ExternalCardCopy(
      title: external.title,
      description: external.description,
      host: host,
      compact: variant == ExternalCardVariant.compact,
    );
    if (thumbnail == null) {
      return Padding(padding: const EdgeInsets.all(12), child: copy);
    }
    final image = _ExternalCardImage(
      external: external,
      cacheManager: ref.watch(feedImageCacheManagerProvider),
    );
    if (variant == ExternalCardVariant.compact) {
      return Row(
        children: [
          SizedBox(width: 72, height: 72, child: image),
          Expanded(
            child: Padding(padding: const EdgeInsets.all(10), child: copy),
          ),
        ],
      );
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        AspectRatio(
          aspectRatio: narrow ? 2 : 16 / 9,
          child: image,
        ),
        Padding(padding: const EdgeInsets.all(12), child: copy),
      ],
    );
  }
}

class _ExternalCardCopy extends StatelessWidget {
  const _ExternalCardCopy({
    required this.title,
    required this.description,
    required this.host,
    required this.compact,
  });

  final String title;
  final String description;
  final String host;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title,
          maxLines: compact ? 1 : 2,
          overflow: TextOverflow.ellipsis,
          style: theme.textTheme.titleSmall,
        ),
        if (description.trim().isNotEmpty && !compact) ...[
          const SizedBox(height: 4),
          Text(
            description,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: theme.textTheme.bodySmall,
          ),
        ],
        if (host.isNotEmpty) ...[
          const SizedBox(height: 6),
          Text(
            host,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: theme.textTheme.labelSmall?.copyWith(
              color: theme.colorScheme.outline,
            ),
          ),
        ],
      ],
    );
  }
}

class _ExternalCardImage extends StatelessWidget {
  const _ExternalCardImage({
    required this.external,
    required this.cacheManager,
  });

  final PostExternal external;
  final BaseCacheManager cacheManager;

  @override
  Widget build(BuildContext context) => ClipRRect(
    key: const Key('external-card-thumbnail'),
    borderRadius: BorderRadius.circular(1),
    child: Semantics(
      image: true,
      label: AppLocalizations.of(context).externalCardThumbnail(external.title),
      child: CachedNetworkImage(
        imageUrl: external.thumb!.url,
        cacheManager: cacheManager,
        fit: BoxFit.cover,
        errorWidget: (_, _, _) => const ColoredBox(color: Color(0xFFEAEAEA)),
      ),
    ),
  );
}

String _displayHost(Uri uri) {
  final defaultPort =
      (uri.scheme == 'http' && uri.port == 80) ||
      (uri.scheme == 'https' && uri.port == 443);
  return uri.hasPort && !defaultPort ? '${uri.host}:${uri.port}' : uri.host;
}
