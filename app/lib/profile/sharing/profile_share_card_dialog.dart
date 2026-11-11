import 'dart:async';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/sharing/profile_share_card.dart';
import 'package:craftsky_app/profile/sharing/share_card_motion.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

Future<void> showProfileShareCard(BuildContext context, Profile profile) =>
    showGeneralDialog<void>(
      context: context,
      barrierDismissible: true,
      barrierLabel: MaterialLocalizations.of(context).modalBarrierDismissLabel,
      barrierColor: BrandColors.ink.withValues(alpha: 0.75),
      transitionDuration: MediaQuery.disableAnimationsOf(context)
          ? Duration.zero
          : const Duration(milliseconds: 320),
      pageBuilder: (_, _, _) => ProfileShareCardDialog(profile: profile),
      transitionBuilder: (_, animation, _, child) => FadeTransition(
        opacity: animation,
        child: child,
      ),
    );

class ProfileShareCardDialog extends ConsumerStatefulWidget {
  const new({required this.profile, super.key});

  final Profile profile;

  @override
  ConsumerState<ProfileShareCardDialog> createState() =>
      _ProfileShareCardDialogState();
}

class _ProfileShareCardDialogState
    extends ConsumerState<ProfileShareCardDialog> {
  ImageProvider<Object>? _avatar;
  bool _didPrepare = false;
  bool _shareRequested = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (!_didPrepare) {
      _didPrepare = true;
      unawaited(_prepare());
    }
  }

  Future<void> _prepare() async {
    final url = widget.profile.avatar;
    if (url == null || url.isEmpty) return;
    final provider = CachedNetworkImageProvider(
      url,
      cacheManager: ref.read(profileImageCacheManagerProvider),
    );
    final avatar = await _resolvedImage(
      provider,
      createLocalImageConfiguration(context),
    );
    if (mounted) setState(() => _avatar = avatar);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Material(
      color: Colors.transparent,
      child: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 560),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          l10n.profileShareCardTitle,
                          style: Theme.of(context).textTheme.titleLarge
                              ?.copyWith(color: BrandColors.paper),
                        ),
                      ),
                      IconButton.filled(
                        tooltip: MaterialLocalizations.of(context)
                            .closeButtonTooltip,
                        style: IconButton.styleFrom(
                          foregroundColor: BrandColors.ink,
                          backgroundColor: BrandColors.paper,
                        ),
                        onPressed: () => Navigator.of(context).pop(),
                        icon: const Icon(CraftskyIcons.close),
                      ),
                    ],
                  ),
                  const SizedBox(height: 20),
                  LayoutBuilder(
                    builder: (_, constraints) => ShareCardMotion(
                      builder: (_, frames) => DecoratedBox(
                        decoration: BoxDecoration(
                          borderRadius: BorderRadius.circular(
                            22 *
                                constraints.maxWidth /
                                shareCardCanvasSize.width,
                          ),
                          boxShadow: const [
                            BoxShadow(
                              color: BrandColors.ink,
                              offset: Offset(6, 7),
                            ),
                          ],
                        ),
                        child: ProfileShareCard(
                          profile: widget.profile,
                          avatar: _avatar,
                          frames: frames,
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(height: 28),
                  ChunkyButton(
                    key: const Key('share-card-share'),
                    backgroundColor: BrandColors.paper,
                    foregroundColor: BrandColors.ink,
                    onPressed: () => setState(() => _shareRequested = true),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const Icon(CraftskyIcons.share, size: 20),
                        const SizedBox(width: 8),
                        Flexible(child: Text(l10n.profileShareAction)),
                      ],
                    ),
                  ),
                  if (_shareRequested) ...[
                    const SizedBox(height: 12),
                    Semantics(
                      liveRegion: true,
                      child: Text(
                        l10n.profileShareComingSoon,
                        textAlign: TextAlign.center,
                        style: const TextStyle(color: BrandColors.paper),
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

Future<ImageProvider<Object>?> _resolvedImage(
  ImageProvider<Object> provider,
  ImageConfiguration configuration,
) async {
  final result = Completer<bool>();
  final stream = provider.resolve(configuration);
  final listener = ImageStreamListener(
    (info, _) {
      info.dispose();
      if (!result.isCompleted) result.complete(true);
    },
    onError: (_, _) {
      if (!result.isCompleted) result.complete(false);
    },
  );
  stream.addListener(listener);
  try {
    final loaded = await result.future.timeout(
      const Duration(seconds: 10),
      onTimeout: () => false,
    );
    return loaded ? provider : null;
  } finally {
    stream.removeListener(listener);
  }
}
