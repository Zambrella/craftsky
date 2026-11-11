import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/sharing/cobalt_thread_surface.dart';
import 'package:craftsky_app/profile/sharing/share_card_motion.dart';
import 'package:craftsky_app/shared/sharing/record_links.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:flutter/foundation.dart';
import 'package:material_ui/material_ui.dart';
import 'package:qr_flutter/qr_flutter.dart';

const shareCardLogo = 'assets/app-icon/app-icon-light.png';
const shareCardTransparentLogo = 'assets/app-icon/android-icon-foreground.png';
const shareCardCanvasSize = Size(480, 600);
const shareCardImageSize = Size(1200, 1500);

/// A fixed 4:5 composition, uniformly scaled for the screen. The capture
/// boundary stays at 480x600 logical pixels inside the scale transform, so
/// the artwork and its exported pixel dimensions never depend on the device.
class ProfileShareCard extends StatelessWidget {
  const new({
    required this.profile,
    this.avatar,
    this.frames,
    this.artworkKey,
    super.key,
  });

  final Profile profile;
  final ImageProvider<Object>? avatar;
  final ValueListenable<ShareCardFrame>? frames;
  final GlobalKey? artworkKey;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final name = profile.displayName?.trim();
    final displayName = name == null || name.isEmpty
        ? profile.handle.value
        : name;
    const labelStyle = TextStyle(
      inherit: false,
      fontFamily: 'Outfit',
      fontSize: 14,
      height: 1.2,
      color: BrandColors.paper,
    );
    return AspectRatio(
      aspectRatio: 4 / 5,
      child: FittedBox(
        child: MediaQuery(
          data: MediaQuery.of(context).copyWith(
            textScaler: TextScaler.noScaling,
            boldText: false,
          ),
          child: Directionality(
            textDirection: TextDirection.ltr,
            child: DefaultTextStyle(
              style: labelStyle,
              child: RepaintBoundary(
                key: artworkKey,
                child: SizedBox.fromSize(
                  size: shareCardCanvasSize,
                  child: ClipRRect(
                    borderRadius: BorderRadius.circular(22),
                    child: Stack(
                      children: [
                        Positioned.fill(
                          child: RepaintBoundary(
                            child: CobaltThreadSurface(frames: frames),
                          ),
                        ),
                        Positioned(
                          top: 24,
                          left: 36,
                          right: 36,
                          height: 28,
                          child: Row(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              ClipRRect(
                                borderRadius: BorderRadius.circular(6),
                                child: Image.asset(
                                  shareCardLogo,
                                  width: 26,
                                  height: 26,
                                  excludeFromSemantics: true,
                                ),
                              ),
                              const SizedBox(width: 8),
                              const Text(
                                'CraftSky',
                                style: TextStyle(
                                  inherit: false,
                                  fontFamily: 'DMSerifDisplay',
                                  fontSize: 24,
                                  height: 1,
                                  color: BrandColors.paper,
                                ),
                              ),
                            ],
                          ),
                        ),
                        Positioned(
                          top: 72,
                          left: 192,
                          width: 96,
                          height: 96,
                          child: Container(
                            decoration: BoxDecoration(
                              shape: BoxShape.circle,
                              border: Border.all(
                                color: BrandColors.paper,
                                width: 3,
                              ),
                            ),
                            padding: const EdgeInsets.all(3),
                            child: ClipOval(
                              child: avatar == null
                                  ? _Initial(name: displayName)
                                  : Image(
                                      image: avatar!,
                                      fit: BoxFit.cover,
                                      excludeFromSemantics: true,
                                      errorBuilder: (_, _, _) =>
                                          _Initial(name: displayName),
                                    ),
                            ),
                          ),
                        ),
                        Positioned(
                          top: 184,
                          left: 36,
                          right: 36,
                          height: 34,
                          child: FittedBox(
                            fit: BoxFit.scaleDown,
                            child: Text(
                              displayName,
                              textAlign: TextAlign.center,
                              style: const TextStyle(
                                inherit: false,
                                fontFamily: 'DMSerifDisplay',
                                fontSize: 32,
                                height: 1,
                                color: BrandColors.paper,
                              ),
                            ),
                          ),
                        ),
                        Positioned(
                          top: 220,
                          left: 36,
                          right: 36,
                          height: 18,
                          child: FittedBox(
                            fit: BoxFit.scaleDown,
                            child: Text(
                              '@${profile.handle.value}',
                              textAlign: TextAlign.center,
                              style: labelStyle,
                            ),
                          ),
                        ),
                        if (profile.crafts.isNotEmpty)
                          Positioned(
                            top: 256,
                            left: 36,
                            right: 36,
                            height: 42,
                            child: FittedBox(
                              fit: BoxFit.scaleDown,
                              child: SizedBox(
                                width: 408,
                                child: Wrap(
                                  alignment: WrapAlignment.center,
                                  spacing: 8,
                                  runSpacing: 8,
                                  children: [
                                    for (final (index, id)
                                        in profile.crafts.toSet().indexed)
                                      _CraftPill(craft: id, index: index),
                                  ],
                                ),
                              ),
                            ),
                          ),
                        Positioned(
                          top: 320,
                          left: 140,
                          width: 200,
                          height: 200,
                          child: Semantics(
                            label: l10n.profileShareCardQrLabel,
                            image: true,
                            child: ExcludeSemantics(
                              child: Container(
                                clipBehavior: Clip.antiAlias,
                                padding: const EdgeInsets.all(3),
                                decoration: BoxDecoration(
                                  color: Colors.white,
                                  borderRadius: BorderRadius.circular(14),
                                ),
                                child: Stack(
                                  alignment: Alignment.center,
                                  children: [
                                    QrImageView(
                                      data: RecordLinks.profile(profile.did),
                                      errorCorrectionLevel:
                                          QrErrorCorrectLevel.H,
                                      // Half the old combined 24px padding.
                                      padding: const EdgeInsets.all(9),
                                      backgroundColor: Colors.white,
                                      eyeStyle: const QrEyeStyle(
                                        eyeShape: QrEyeShape.square,
                                        color: BrandColors.cobalt,
                                      ),
                                      dataModuleStyle: const QrDataModuleStyle(
                                        dataModuleShape:
                                            QrDataModuleShape.square,
                                        color: BrandColors.cobalt,
                                      ),
                                    ),
                                    ColoredBox(
                                      key: const Key('share-card-qr-logo'),
                                      color: Colors.white,
                                      child: SizedBox.square(
                                        dimension: 34,
                                        child: ClipRect(
                                          // The adaptive icon has a transparent
                                          // safe area. Crop that padding only.
                                          child: OverflowBox(
                                            maxWidth: 48,
                                            maxHeight: 48,
                                            child: Image.asset(
                                              shareCardTransparentLogo,
                                              width: 48,
                                              height: 48,
                                              excludeFromSemantics: true,
                                            ),
                                          ),
                                        ),
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            ),
                          ),
                        ),
                        Positioned(
                          top: 548,
                          left: 36,
                          right: 36,
                          height: 18,
                          child: FittedBox(
                            fit: BoxFit.scaleDown,
                            child: Text(
                              l10n.profileShareCardScan,
                              textAlign: TextAlign.center,
                              style: labelStyle,
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _Initial extends StatelessWidget {
  const new({required this.name});

  final String name;

  @override
  Widget build(BuildContext context) => ColoredBox(
    color: BrandColors.butter,
    child: Center(
      child: Text(
        name.characters.first.toUpperCase(),
        style: const TextStyle(
          inherit: false,
          fontFamily: 'DMSerifDisplay',
          fontSize: 42,
          color: BrandColors.ink,
        ),
      ),
    ),
  );
}

class _CraftPill extends StatelessWidget {
  const new({required this.craft, required this.index});

  final String craft;
  final int index;
  static const List<Color> _swatches = [
    BrandColors.butter,
    BrandColors.lilac,
    BrandColors.sky,
    BrandColors.clay,
    BrandColors.paper,
  ];

  @override
  Widget build(BuildContext context) {
    final id = craft.split('#').last;
    final known = Craft.fromId(id);
    final label = known == null
        ? id
        : craftLabel(known, AppLocalizations.of(context));
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: _swatches[index % _swatches.length],
        border: Border.all(color: BrandColors.ink, width: 1.5),
        borderRadius: BorderRadius.circular(999),
      ),
      child: CraftIconLabel(
        craft: craft,
        label: label,
        iconSize: 16,
        flexibleLabel: true,
        style: const TextStyle(
          inherit: false,
          fontFamily: 'Outfit',
          fontWeight: FontWeight.w600,
          fontSize: 13,
          color: BrandColors.ink,
        ),
      ),
    );
  }
}
