import 'dart:typed_data';

import 'package:craftsky_app/feed/composer/published_post_embed_edit.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/image/craftsky_image_attachment_preview.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:material_ui/material_ui.dart';

final class PublishedExternalEditCard extends StatelessWidget {
  const new({
    required this.edit,
    required this.enabled,
    required this.onChanged,
    required this.onRemove,
    required this.onRefresh,
    required this.onRemoveThumbnail,
    this.thumbnailBytes,
    this.thumbnailUrl,
    super.key,
  });

  final PublishedPostEmbedEdit edit;
  final bool enabled;
  final VoidCallback onChanged;
  final VoidCallback onRemove;
  final VoidCallback onRefresh;
  final VoidCallback onRemoveThumbnail;
  final Uint8List? thumbnailBytes;
  final String? thumbnailUrl;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return DecoratedBox(
      decoration: BoxDecoration(
        border: Border.all(color: Theme.of(context).colorScheme.outline),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (thumbnailBytes != null ||
                (edit.hasExternalThumbnail && thumbnailUrl != null))
              CraftskyImageAttachmentPreview(
                imageUrl: thumbnailUrl,
                bytes: thumbnailBytes,
                aspectRatio: 16 / 9,
              ),
            BrandTextField(
              key: const ValueKey('published-external-uri-field'),
              textFieldKey: const Key('published-external-uri'),
              label: l10n.postEditExternalUrl,
              initialValue: edit.externalUri,
              enabled: enabled,
              maxBytes: 8192,
              keyboardType: TextInputType.url,
              onChanged: (value) {
                edit.updateExternal(uri: value);
                onChanged();
              },
            ),
            const SizedBox(height: 12),
            BrandTextField(
              key: const ValueKey('published-external-title-field'),
              textFieldKey: const Key('published-external-title'),
              label: l10n.postEditExternalTitle,
              initialValue: edit.externalTitle,
              enabled: enabled,
              maxLength: 200,
              maxBytes: 2000,
              onChanged: (value) {
                edit.updateExternal(title: value);
                onChanged();
              },
            ),
            const SizedBox(height: 12),
            BrandTextField(
              key: const ValueKey('published-external-description-field'),
              textFieldKey: const Key('published-external-description'),
              label: l10n.postEditExternalDescription,
              initialValue: edit.externalDescription,
              enabled: enabled,
              maxLength: 300,
              maxBytes: 3000,
              minLines: 2,
              maxLines: 4,
              onChanged: (value) {
                edit.updateExternal(description: value);
                onChanged();
              },
            ),
            Wrap(
              spacing: 8,
              children: [
                TextButton(
                  key: const Key('published-external-refresh'),
                  onPressed: enabled ? onRefresh : null,
                  child: Text(l10n.postEditRefreshPreview),
                ),
                if (edit.hasExternalThumbnail || thumbnailBytes != null)
                  TextButton(
                    key: const Key('published-external-remove-thumbnail'),
                    onPressed: enabled ? onRemoveThumbnail : null,
                    child: Text(l10n.postEditRemovePreviewImage),
                  ),
                TextButton(
                  key: const Key('published-external-remove'),
                  onPressed: enabled ? onRemove : null,
                  child: Text(l10n.linkPreviewDismiss),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
