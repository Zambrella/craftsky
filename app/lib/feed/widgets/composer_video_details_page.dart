import 'dart:async';

import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/widgets/local_video_preview_resource.dart';
import 'package:craftsky_app/feed/widgets/native_video_player.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:material_ui/material_ui.dart';

Future<void> showComposerVideoDetails(
  BuildContext context, {
  required LocalVideoSelection selection,
  required ValueChanged<String> onAltTextChanged,
  Widget Function(BuildContext, LocalVideoSelection)? previewBuilder,
}) => Navigator.of(context).push<void>(
  MaterialPageRoute<void>(
    builder: (context) => ComposerVideoDetailsPage(
      selection: selection,
      onAltTextChanged: onAltTextChanged,
      previewBuilder: previewBuilder,
    ),
  ),
);

class ComposerVideoDetailsPage extends StatelessWidget {
  const new({
    required this.selection,
    required this.onAltTextChanged,
    this.previewBuilder,
    super.key,
  });
  final LocalVideoSelection selection;
  final ValueChanged<String> onAltTextChanged;
  final Widget Function(BuildContext, LocalVideoSelection)? previewBuilder;
  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      key: const Key('composer-video-details'),
      appBar: AppBar(title: Text(l10n.postComposeVideoDetailsTitle)),
      body: SafeArea(
        top: false,
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              previewBuilder?.call(context, selection) ??
                  _LocalPreview(selection: selection),
              const SizedBox(height: 16),
              BrandTextField(
                textFieldKey: const Key('composer-video-alt-text'),
                label: l10n.postComposeAltTextLabel,
                initialValue: selection.altText,
                maxLength: RecordTextLimits.videoAlt.graphemes,
                maxBytes: RecordTextLimits.videoAlt.bytes,
                maxLines: 3,
                minLines: 3,
                keyboardType: TextInputType.multiline,
                textInputAction: TextInputAction.newline,
                textCapitalization: TextCapitalization.sentences,
                hintText: l10n.postComposeVideoAltHint,
                onChanged: onAltTextChanged,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _LocalPreview extends StatefulWidget {
  const new({required this.selection});
  final LocalVideoSelection selection;
  @override
  State<_LocalPreview> createState() => _LocalPreviewState();
}

class _LocalPreviewState extends State<_LocalPreview> {
  LocalVideoPreviewResource? _resource;
  bool _failed = false;
  @override
  void initState() {
    super.initState();
    unawaited(_prepare());
  }

  Future<void> _prepare() async {
    try {
      final resource = await createLocalVideoPreviewResource(widget.selection);
      if (!mounted) {
        await resource.dispose();
        return;
      }
      setState(() => _resource = resource);
    } on Object {
      if (mounted) setState(() => _failed = true);
    }
  }

  @override
  void dispose() {
    unawaited(_resource?.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_failed) {
      return Text(
        AppLocalizations.of(context).postVideoLocalPreviewUnavailable,
      );
    }
    final resource = _resource;
    if (resource == null) {
      return const Center(child: CircularProgressIndicator());
    }
    return NativeVideoPlayer.local(
      localSelection: widget.selection,
      localSource: resource.uri,
    );
  }
}
