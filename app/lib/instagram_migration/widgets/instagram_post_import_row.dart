import 'dart:async';

import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_published_import.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_post_archive.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/foundation.dart';
import 'package:material_ui/material_ui.dart';

typedef LoadInstagramImportPreview = Future<Uint8List?> Function(
  bool Function() thumbnailIsMounted,
);

class InstagramPostImportRow extends StatelessWidget {
  const new({
    required this.post,
    required this.published,
    required this.loadPreview,
    required this.onCreate,
    required this.onView,
    super.key,
  });

  final InstagramArchivePost post;
  final InstagramPublishedImport? published;
  final LoadInstagramImportPreview loadPreview;
  final ValueChanged<LocalPostDraftKind>? onCreate;
  final ValueChanged<InstagramPublishedImport>? onView;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final photoCount = post.media
        .where((entry) => entry.kind == InstagramArchiveMediaKind.photo)
        .length;
    final videoCount = post.media
        .where((entry) => entry.kind == InstagramArchiveMediaKind.video)
        .length;
    final caption = post.caption.characters;
    final warnings = post.warnings
        .map((warning) => _warning(l10n, warning))
        .toSet();
    final publishedPost = published;

    return CraftskyCard(
      padding: EdgeInsets.all(spacing.sp4),
      clipBehavior: Clip.none,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        spacing: spacing.sp3,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            spacing: spacing.sp3,
            children: [
              _ImportThumbnail(
                canPreview: post.media.any((entry) => entry.available),
                placeholder: photoCount == 0 && videoCount > 0
                    ? CraftskyIcons.video
                    : CraftskyIcons.image,
                loadPreview: loadPreview,
              ),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  spacing: spacing.sp1,
                  children: [
                    Text(
                      MaterialLocalizations.of(context).formatMediumDate(
                        post.createdAt.toLocal(),
                      ),
                      style: theme.textTheme.titleSmall,
                    ),
                    Text(
                      caption.isEmpty
                          ? l10n.instagramPostImportNoCaption
                          : caption.take(240).string,
                      maxLines: 3,
                      overflow: TextOverflow.ellipsis,
                    ),
                    Text(
                      l10n.instagramPostImportMediaSummary(
                        photoCount,
                        videoCount,
                        post.captionLength,
                      ),
                      style: theme.textTheme.bodySmall,
                    ),
                  ],
                ),
              ),
            ],
          ),
          for (final warning in warnings)
            Text(warning, style: theme.textTheme.bodySmall),
          if (publishedPost != null)
            Wrap(
              spacing: spacing.sp2,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: [
                Text(
                  publishedPost.isProject
                      ? l10n.instagramPostImportProjectCreated
                      : l10n.instagramPostImportPosted,
                  style: theme.textTheme.labelLarge,
                ),
                TextButton(
                  onPressed: onView == null
                      ? null
                      : () => onView!(publishedPost),
                  child: Text(
                    publishedPost.isProject
                        ? l10n.instagramPostImportViewProject
                        : l10n.instagramPostImportViewPost,
                  ),
                ),
              ],
            )
          else
            Wrap(
              spacing: spacing.sp3,
              runSpacing: spacing.sp2,
              children: [
                TextButton.icon(
                  onPressed: onCreate == null
                      ? null
                      : () => onCreate!(LocalPostDraftKind.standard),
                  icon: const Icon(CraftskyIconsBold.textPost),
                  label: Text(l10n.instagramPostImportCreatePost),
                ),
                TextButton.icon(
                  onPressed: onCreate == null
                      ? null
                      : () => onCreate!(LocalPostDraftKind.project),
                  icon: const Icon(CraftskyIconsBold.projectPost),
                  label: Text(l10n.instagramPostImportCreateProject),
                ),
              ],
            ),
        ],
      ),
    );
  }
}

String _warning(AppLocalizations l10n, InstagramArchiveWarning warning) =>
    switch (warning) {
      InstagramArchiveWarning.captionTooLong =>
        l10n.instagramPostImportCaptionLimit,
      InstagramArchiveWarning.tooManyPhotos =>
        l10n.instagramPostImportPhotoLimit,
      InstagramArchiveWarning.multipleVideos =>
        l10n.instagramPostImportMultipleVideos,
      InstagramArchiveWarning.mixedMedia => l10n.instagramPostImportMixedMedia,
      InstagramArchiveWarning.unsupportedMedia ||
      InstagramArchiveWarning.unavailableMedia ||
      InstagramArchiveWarning.oversizedMedia =>
        l10n.instagramPostImportUnavailableMedia,
    };

class _ImportThumbnail extends StatefulWidget {
  const new({
    required this.canPreview,
    required this.placeholder,
    required this.loadPreview,
  });

  final bool canPreview;
  final IconData placeholder;
  final LoadInstagramImportPreview loadPreview;

  @override
  State<_ImportThumbnail> createState() => _ImportThumbnailState();
}

class _ImportThumbnailState extends State<_ImportThumbnail> {
  MemoryImage? _image;

  @override
  void initState() {
    super.initState();
    if (widget.canPreview) unawaited(_load());
  }

  Future<void> _load() async {
    try {
      final bytes = await widget.loadPreview(() => mounted);
      if (mounted && bytes != null) {
        setState(() => _image = MemoryImage(bytes));
      }
    } on InstagramPostArchiveException {
      // A missing thumbnail does not discard a recoverable source post.
    }
  }

  @override
  void dispose() {
    final image = _image;
    if (image != null) unawaited(image.evict());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final image = _image;
    return ExcludeSemantics(
      child: SizedBox.square(
        dimension: 72,
        child: ClipRRect(
          borderRadius: BorderRadius.circular(
            theme.extension<RadiusTheme>()!.r2,
          ),
          child: ColoredBox(
            color: theme.colorScheme.surfaceContainerHighest,
            child: image == null
                ? Icon(widget.placeholder)
                : Image(
                    image: image,
                    fit: BoxFit.cover,
                    errorBuilder: (_, _, _) => Icon(widget.placeholder),
                  ),
          ),
        ),
      ),
    );
  }
}

/// Ordinary photo or single-video imports open the composer directly. Mixed
/// media and carousels over the limit require an explicit choice first.
Future<List<int>?> chooseInstagramPostMedia(
  BuildContext context,
  InstagramArchivePost post,
) {
  final photos = <int>[];
  final videos = <int>[];
  for (final (index, media) in post.media.indexed) {
    if (!media.available) continue;
    switch (media.kind) {
      case InstagramArchiveMediaKind.photo:
        photos.add(index);
      case InstagramArchiveMediaKind.video:
        videos.add(index);
      case InstagramArchiveMediaKind.unsupported:
        break;
    }
  }
  if (videos.isEmpty && photos.length <= defaultMaxPostImages) {
    return Future.value(photos);
  }
  if (photos.isEmpty && videos.length <= 1) return Future.value(videos);
  return showCraftskyModal<List<int>>(
    context,
    builder: (_) => _InstagramMediaDialog(photos: photos, videos: videos),
  );
}

class _InstagramMediaDialog extends StatefulWidget {
  const new({required this.photos, required this.videos});

  final List<int> photos;
  final List<int> videos;

  @override
  State<_InstagramMediaDialog> createState() => _InstagramMediaDialogState();
}

class _InstagramMediaDialogState extends State<_InstagramMediaDialog> {
  late final Set<int> _photos = widget.photos
      .take(defaultMaxPostImages)
      .toSet();
  late int? _video = widget.videos.firstOrNull;
  late bool _showPhotos = widget.photos.isNotEmpty;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final items = _showPhotos ? widget.photos : widget.videos;
    final selected = _showPhotos ? (_photos.toList()..sort()) : [?_video];
    return SafeArea(
      child: CraftskyDialog(
        title: l10n.instagramPostImportMediaTitle,
        body: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          spacing: spacing.sp2,
          children: [
            Text(l10n.instagramPostImportMediaHelp),
            if (widget.photos.isNotEmpty && widget.videos.isNotEmpty)
              Wrap(
                spacing: spacing.sp2,
                children: [
                  ChoiceChip(
                    label: Text(l10n.instagramPostImportPhotos),
                    selected: _showPhotos,
                    onSelected: (_) => setState(() => _showPhotos = true),
                  ),
                  ChoiceChip(
                    label: Text(l10n.instagramPostImportVideo),
                    selected: !_showPhotos,
                    onSelected: (_) => setState(() => _showPhotos = false),
                  ),
                ],
              ),
            ConstrainedBox(
              constraints: BoxConstraints(
                maxHeight: MediaQuery.sizeOf(context).height * 0.35,
              ),
              child: RadioGroup<int>(
                groupValue: _video,
                onChanged: (value) => setState(() => _video = value),
                child: ListView.builder(
                  primary: false,
                  shrinkWrap: true,
                  itemCount: items.length,
                  itemBuilder: (context, index) {
                    final item = items[index];
                    if (!_showPhotos) {
                      return RadioListTile<int>(
                        value: item,
                        title: Text(
                          l10n.instagramPostImportVideoLabel(index + 1),
                        ),
                      );
                    }
                    final checked = _photos.contains(item);
                    return CheckboxListTile(
                      value: checked,
                      title: Text(
                        l10n.instagramPostImportPhotoLabel(index + 1),
                      ),
                      onChanged:
                          !checked && _photos.length >= defaultMaxPostImages
                          ? null
                          : (selected) => setState(() {
                              if (selected == true) {
                                _photos.add(item);
                              } else {
                                _photos.remove(item);
                              }
                            }),
                    );
                  },
                ),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(),
            child: Text(l10n.languageCancel),
          ),
          ChunkyButton(
            onPressed: selected.isEmpty
                ? null
                : () => Navigator.of(context).pop(selected),
            child: Text(l10n.instagramPostImportContinue),
          ),
        ],
      ),
    );
  }
}
