import 'dart:async';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/feed/media/youtube_consent.dart';
import 'package:craftsky_app/feed/media/youtube_external.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/youtube_inline_player.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

enum ExternalCardVariant { full, compact }

final externalCardLauncherProvider = Provider<ExternalLinkLauncher>(
  (_) => launchExternalLink,
);

class ExternalCard extends ConsumerStatefulWidget {
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
  ConsumerState<ExternalCard> createState() => _ExternalCardState();
}

class _ExternalCardState extends ConsumerState<ExternalCard> {
  var _isPlayingYouTube = false;
  var _youtubePlaybackFailed = false;

  @override
  Widget build(BuildContext context) {
    final external = widget.external;
    final uri = normalizeExternalLinkUri(external.uri);
    final host = uri == null ? '' : _displayHost(uri);
    final youtube = switch ((uri, widget.variant)) {
      (final uri?, ExternalCardVariant.full) => parseYouTubeExternal(uri),
      _ => null,
    };
    final theme = Theme.of(context);
    final radii = theme.extension<RadiusTheme>()!;
    final l10n = AppLocalizations.of(context);
    final label = youtube == null
        ? l10n.externalCardOpen(host)
        : _youtubePlaybackFailed
        ? l10n.youtubePlaybackUnavailable
        : _isPlayingYouTube
        ? l10n.youtubeVideoPlayer(external.title)
        : l10n.youtubePlayVideo(external.title);
    final launcher = switch (widget.launchUrl) {
      final launch? => launch,
      null => ref.watch(externalCardLauncherProvider),
    };
    final VoidCallback? onTap;
    if (uri == null || (youtube != null && _isPlayingYouTube)) {
      onTap = null;
    } else if (youtube != null) {
      onTap = () => unawaited(_playYouTube());
    } else {
      onTap = () => unawaited(
        confirmAndLaunchExternalLink(
          context,
          uri: uri,
          launchUrl: launcher,
        ),
      );
    }

    return Semantics(
      button: onTap != null,
      label: label,
      child: Material(
        color: Colors.transparent,
        clipBehavior: Clip.antiAlias,
        borderRadius: BorderRadius.circular(radii.r2),
        child: InkWell(
          borderRadius: BorderRadius.circular(radii.r2),
          onTap: onTap,
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
                  host,
                  youtube: youtube,
                  uri: uri,
                  launcher: launcher,
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
    String host, {
    required YouTubeExternal? youtube,
    required Uri? uri,
    required ExternalLinkLauncher launcher,
    required bool narrow,
  }) {
    final external = widget.external;
    final thumbnail = external.thumb;
    final copy = _ExternalCardCopy(
      title: external.title,
      description: external.description,
      host: host,
      compact: widget.variant == ExternalCardVariant.compact,
      showPlayIndicator: youtube != null && thumbnail == null,
    );
    if (_isPlayingYouTube && youtube != null) {
      final player = ref.watch(youtubePlayerBuilderProvider);
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (_youtubePlaybackFailed)
            const _YouTubePlaybackError()
          else
            player(context, youtube, _handleYouTubePlaybackError),
          Padding(padding: const EdgeInsets.all(12), child: copy),
          if (uri != null)
            Align(
              alignment: AlignmentDirectional.centerStart,
              child: TextButton.icon(
                key: const Key('youtube-open-externally'),
                onPressed: () => unawaited(
                  confirmAndLaunchExternalLink(
                    context,
                    uri: uri,
                    launchUrl: launcher,
                  ),
                ),
                icon: const Icon(CraftskyIconsBold.externalLink),
                label: Text(
                  AppLocalizations.of(context).youtubeOpenExternally,
                ),
              ),
            ),
        ],
      );
    }
    if (thumbnail == null) {
      return Padding(padding: const EdgeInsets.all(12), child: copy);
    }
    final image = _ExternalCardImage(
      external: external,
      cacheManager: ref.watch(feedImageCacheManagerProvider),
    );
    final preview = youtube == null
        ? image
        : Stack(
            fit: StackFit.expand,
            children: [
              image,
              const ColoredBox(color: Color(0x33000000)),
              const Center(
                child: Icon(
                  CraftskyIconsBold.play,
                  key: Key('youtube-play-indicator'),
                  size: 56,
                  color: Colors.white,
                ),
              ),
            ],
          );
    if (widget.variant == ExternalCardVariant.compact) {
      return Row(
        children: [
          SizedBox(width: 72, height: 72, child: preview),
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
          child: preview,
        ),
        Padding(padding: const EdgeInsets.all(12), child: copy),
      ],
    );
  }

  Future<void> _playYouTube() async {
    final preferences = ref.read(youtubeConsentPreferencesProvider);
    if (!preferences.alwaysAllow) {
      final choice = await showDialog<_YouTubeConsentChoice>(
        context: context,
        builder: (context) => const _YouTubeConsentDialog(),
      );
      if (!mounted || choice == null) {
        return;
      }
      if (choice == _YouTubeConsentChoice.alwaysAllow) {
        await preferences.setAlwaysAllow();
        if (!mounted) {
          return;
        }
      }
    }
    setState(() => _isPlayingYouTube = true);
  }

  void _handleYouTubePlaybackError() {
    if (!mounted || _youtubePlaybackFailed) {
      return;
    }
    setState(() => _youtubePlaybackFailed = true);
  }

  @override
  void didUpdateWidget(covariant ExternalCard oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.external.uri != widget.external.uri) {
      _isPlayingYouTube = false;
      _youtubePlaybackFailed = false;
    }
  }
}

class _ExternalCardCopy extends StatelessWidget {
  const _ExternalCardCopy({
    required this.title,
    required this.description,
    required this.host,
    required this.compact,
    required this.showPlayIndicator,
  });

  final String title;
  final String description;
  final String host;
  final bool compact;
  final bool showPlayIndicator;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            if (showPlayIndicator) ...[
              const Icon(
                CraftskyIconsBold.play,
                key: Key('youtube-play-indicator'),
              ),
              const SizedBox(width: 8),
            ],
            Expanded(
              child: Text(
                title,
                maxLines: compact ? 1 : 2,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.titleSmall,
              ),
            ),
          ],
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
        errorWidget: (context, _, _) => ColoredBox(
          color: Theme.of(context).brightness == Brightness.dark
              ? Theme.of(context).colorScheme.surfaceContainerHighest
              : const Color(0xFFEAEAEA),
        ),
      ),
    ),
  );
}

enum _YouTubeConsentChoice { allowOnce, alwaysAllow }

class _YouTubeConsentDialog extends StatelessWidget {
  const _YouTubeConsentDialog();

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return AlertDialog(
      title: Text(l10n.youtubeConsentTitle),
      content: Text(l10n.youtubeConsentMessage),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(MaterialLocalizations.of(context).cancelButtonLabel),
        ),
        TextButton(
          onPressed: () =>
              Navigator.pop(context, _YouTubeConsentChoice.allowOnce),
          child: Text(l10n.youtubeAllowOnce),
        ),
        FilledButton(
          onPressed: () =>
              Navigator.pop(context, _YouTubeConsentChoice.alwaysAllow),
          child: Text(l10n.youtubeAlwaysAllow),
        ),
      ],
    );
  }
}

class _YouTubePlaybackError extends StatelessWidget {
  const _YouTubePlaybackError();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return AspectRatio(
      aspectRatio: 16 / 9,
      child: ColoredBox(
        color: theme.colorScheme.surfaceContainerHighest,
        child: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  CraftskyIcons.fileVideo,
                  size: 40,
                  color: theme.colorScheme.onSurfaceVariant,
                ),
                const SizedBox(height: 12),
                Text(
                  AppLocalizations.of(context).youtubePlaybackUnavailable,
                  textAlign: TextAlign.center,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

String _displayHost(Uri uri) {
  final defaultPort =
      (uri.scheme == 'http' && uri.port == 80) ||
      (uri.scheme == 'https' && uri.port == 443);
  return uri.hasPort && !defaultPort ? '${uri.host}:${uri.port}' : uri.host;
}
