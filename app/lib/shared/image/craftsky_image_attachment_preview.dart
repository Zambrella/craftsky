import 'dart:typed_data';

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class CraftskyImageAttachmentPreview extends StatelessWidget {
  const CraftskyImageAttachmentPreview({
    required this.aspectRatio,
    super.key,
    this.bytes,
    this.imageUrl,
    this.unavailable = false,
    this.placeholderIcon,
    this.busyLabel,
    this.busyProgress,
    this.imageKey,
    this.overlayKey,
    this.progressKey,
    this.labelKey,
  });

  final Uint8List? bytes;
  final String? imageUrl;
  final double aspectRatio;
  final bool unavailable;
  final IconData? placeholderIcon;
  final String? busyLabel;
  final double? busyProgress;
  final Key? imageKey;
  final Key? overlayKey;
  final Key? progressKey;
  final Key? labelKey;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return DecoratedBox(
      position: DecorationPosition.foreground,
      decoration: BoxDecoration(
        border: Border.all(color: colors.onSurface),
      ),
      child: AnimatedSize(
        duration: const Duration(milliseconds: 180),
        curve: Curves.easeOutCubic,
        alignment: Alignment.topCenter,
        child: AspectRatio(
          aspectRatio: aspectRatio.clamp(0.7, 1.6),
          child: Stack(
            fit: StackFit.expand,
            children: [
              switch ((bytes, imageUrl)) {
                (final value?, _) => Image.memory(
                  value,
                  key: imageKey,
                  fit: BoxFit.cover,
                  width: double.infinity,
                ),
                (null, final url?) => CachedNetworkImage(
                  imageUrl: url,
                  key: imageKey,
                  fit: BoxFit.cover,
                  width: double.infinity,
                  errorWidget: (_, _, _) => const Center(
                    child: Icon(Icons.broken_image_outlined),
                  ),
                ),
                (null, null) => DecoratedBox(
                  decoration: const BoxDecoration(color: Color(0xFFEAEAEA)),
                  child: unavailable
                      ? const Center(child: Icon(Icons.broken_image_outlined))
                      : switch (placeholderIcon) {
                          final icon? => Center(child: Icon(icon, size: 56)),
                          null => null,
                        },
                ),
              },
              if (busyLabel case final label?)
                _ImageBusyOverlay(
                  label: label,
                  progress: busyProgress,
                  overlayKey: overlayKey,
                  progressKey: progressKey,
                  labelKey: labelKey,
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ImageBusyOverlay extends StatelessWidget {
  const _ImageBusyOverlay({
    required this.label,
    required this.progress,
    required this.overlayKey,
    required this.progressKey,
    required this.labelKey,
  });

  final String label;
  final double? progress;
  final Key? overlayKey;
  final Key? progressKey;
  final Key? labelKey;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Semantics(
      label: label,
      liveRegion: true,
      child: DecoratedBox(
        key: overlayKey,
        decoration: const BoxDecoration(color: Color(0x99000000)),
        child: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              SizedBox.square(
                dimension: 72,
                child: CircularProgressIndicator(
                  key: progressKey,
                  value: progress,
                  strokeWidth: 6,
                  backgroundColor: Colors.white24,
                  valueColor: const AlwaysStoppedAnimation<Color>(Colors.white),
                ),
              ),
              SizedBox(height: spacing.sp3),
              Text(
                label,
                key: labelKey,
                textAlign: TextAlign.center,
                style: theme.textTheme.titleMedium?.copyWith(
                  color: Colors.white,
                  fontWeight: FontWeight.w900,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
