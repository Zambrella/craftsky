import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/feed/models/video_thumbnail_frame.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/feed/providers/composer_images_provider.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/widgets/local_video_preview_resource.dart';
import 'package:craftsky_app/feed/widgets/native_video_player.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:craftsky_app/shared/widgets/form_action_layout.dart';
import 'package:craftsky_app/shared/widgets/form_keyboard_scope.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/foundation.dart' show compute;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image/image.dart' as image;
import 'package:image_picker/image_picker.dart';
import 'package:material_ui/material_ui.dart';

Future<void> showComposerVideoDetails(
  BuildContext context, {
  required LocalVideoSelection selection,
  required ValueChanged<String> onAltTextChanged,
  ValueChanged<Uint8List?>? onThumbnailChanged,
  ValueChanged<VideoThumbnailFrame>? onThumbnailFrameChanged,
  Future<VideoUploadLimits?> Function()? checkLimits,
  Widget Function(BuildContext, LocalVideoSelection)? previewBuilder,
}) {
  FocusManager.instance.primaryFocus?.unfocus();
  return Navigator.of(context).push<void>(
    MaterialPageRoute<void>(
      builder: (context) => ComposerVideoDetailsPage(
        selection: selection,
        onAltTextChanged: onAltTextChanged,
        onThumbnailChanged: onThumbnailChanged,
        onThumbnailFrameChanged: onThumbnailFrameChanged,
        checkLimits: checkLimits,
        previewBuilder: previewBuilder,
      ),
    ),
  );
}

class ComposerVideoDetailsPage extends StatefulWidget {
  const new({
    required this.selection,
    required this.onAltTextChanged,
    this.onThumbnailChanged,
    this.onThumbnailFrameChanged,
    this.checkLimits,
    this.previewBuilder,
    super.key,
  });
  final LocalVideoSelection selection;
  final ValueChanged<String> onAltTextChanged;
  final ValueChanged<Uint8List?>? onThumbnailChanged;
  final ValueChanged<VideoThumbnailFrame>? onThumbnailFrameChanged;
  final Future<VideoUploadLimits?> Function()? checkLimits;
  final Widget Function(BuildContext, LocalVideoSelection)? previewBuilder;
  @override
  State<ComposerVideoDetailsPage> createState() =>
      _ComposerVideoDetailsPageState();
}

class _ComposerVideoDetailsPageState extends State<ComposerVideoDetailsPage> {
  late Uint8List? _thumbnail = widget.selection.thumbnail;
  late VideoThumbnailFrame _thumbnailFrame = widget.selection.thumbnailFrame;
  final _framePreviews = <VideoThumbnailFrame, Uint8List>{};
  bool _available = false;
  bool _preparing = false;
  bool _failed = false;

  @override
  void initState() {
    super.initState();
    unawaited(_checkAvailability());
  }

  Future<void> _checkAvailability() async {
    try {
      final limits = await widget.checkLimits?.call();
      if (mounted) {
        setState(() => _available = limits?.customThumbnail ?? false);
      }
    } on Object {
      // Keep video details usable when capability discovery fails.
    }
  }

  Future<void> _pickThumbnail() async {
    setState(() {
      _preparing = true;
      _failed = false;
    });
    try {
      final file = await ProviderScope.containerOf(context, listen: false)
          .read(imagePickerProvider)
          .pickImage(
            source: ImageSource.gallery,
            maxWidth: 1280,
            maxHeight: 1280,
            imageQuality: 88,
          );
      if (file == null || !mounted) return;
      if (await file.length() > 10 * 1024 * 1024) {
        throw const FormatException(
          'Thumbnail source exceeds the image size limit',
        );
      }
      final bytes = await compute(_encodeThumbnail, await file.readAsBytes());
      if (!mounted) return;
      _setThumbnail(bytes);
    } on Object {
      if (mounted) setState(() => _failed = true);
    } finally {
      if (mounted) setState(() => _preparing = false);
    }
  }

  void _setThumbnail(Uint8List? bytes) {
    setState(() {
      _thumbnail = bytes;
      _failed = false;
    });
    widget.onThumbnailChanged?.call(bytes);
  }

  Widget _frameOption(
    BuildContext context,
    VideoThumbnailFrame frame,
    double aspectRatio,
    RoundedRectangleBorder shape,
  ) {
    final selected = frame == _thumbnailFrame;
    final preview = _framePreviews[frame];
    final colors = Theme.of(context).colorScheme;
    return MergeSemantics(
      child: Semantics(
        label: AppLocalizations.of(context)
            .postVideoThumbnailFrame(frame.percent),
        selected: selected,
        child: OutlinedButton(
          onPressed: () {
            setState(() => _thumbnailFrame = frame);
            widget.onThumbnailFrameChanged?.call(frame);
          },
          clipBehavior: Clip.antiAlias,
          style: OutlinedButton.styleFrom(
            padding: EdgeInsets.zero,
            minimumSize: const Size(0, 48),
            side: BorderSide.none,
            shape: shape,
          ),
          child: DecoratedBox(
            position: DecorationPosition.foreground,
            decoration: ShapeDecoration(
              shape: selected
                  ? shape.copyWith(
                      side: shape.side.copyWith(
                        color: colors.primary,
                        width: shape.side.width + 3,
                      ),
                    )
                  : shape,
            ),
            child: AspectRatio(
              aspectRatio: aspectRatio,
              child: Stack(
                fit: StackFit.expand,
                children: [
                  if (preview == null)
                    const Center(child: Icon(Icons.image_outlined))
                  else
                    Image.memory(
                      preview,
                      fit: BoxFit.cover,
                      excludeFromSemantics: true,
                    ),
                  if (selected)
                    PositionedDirectional(
                      top: 6,
                      end: 6,
                      child: CircleAvatar(
                        radius: 12,
                        backgroundColor: colors.primary,
                        foregroundColor: colors.onPrimary,
                        child: const Icon(Icons.check, size: 16),
                      ),
                    ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final headingStyle = theme.textTheme.titleSmall;
    final radius = theme.extension<RadiusTheme>()?.r2 ?? const RadiusTheme().r2;
    final thumbnailShape = RoundedRectangleBorder(
      borderRadius: BorderRadius.circular(radius),
      side: BorderSide(color: theme.colorScheme.outlineVariant),
    );
    final ratio = nativeVideoAspectRatio(
      width: widget.selection.width,
      height: widget.selection.height,
    );
    return FormKeyboardScope(
      child: Scaffold(
        key: const Key('composer-video-details'),
        appBar: AppBar(title: Text(l10n.postComposeVideoDetailsTitle)),
        body: FormActionLayout(
          keyboardVisible: MediaQuery.viewInsetsOf(context).bottom > 0,
          label: AppLocalizations.of(context).postComposeDone,
          actionKey: const Key('composer-details-done'),
          onPressed: () => Navigator.of(context).maybePop(),
          contentBuilder: (_, clearance) => SafeArea(
            top: false,
            bottom: false,
            child: SingleChildScrollView(
              padding: EdgeInsets.fromLTRB(16, 16, 16, 16 + clearance),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Semantics(
                    header: true,
                    child: Text(
                      l10n.postComposeVideoPreviewHeading,
                      style: headingStyle,
                    ),
                  ),
                  const SizedBox(height: 8),
                  widget.previewBuilder?.call(context, widget.selection) ??
                      _LocalPreview(
                        selection: widget.selection,
                        onFramePreview: (frame, bytes) =>
                            setState(() => _framePreviews[frame] = bytes),
                      ),
                  const SizedBox(height: 16),
                  if (widget.onThumbnailChanged != null &&
                      (_available || _thumbnail != null)) ...[
                    Semantics(
                      header: true,
                      child: Text(
                        l10n.postComposeVideoThumbnailHeading,
                        style: headingStyle,
                      ),
                    ),
                    const SizedBox(height: 8),
                    if (_thumbnail case final thumbnail?)
                      Align(
                        alignment: Alignment.centerLeft,
                        child: ConstrainedBox(
                          constraints: const BoxConstraints(
                            maxWidth: 420,
                            maxHeight: 240,
                          ),
                          child: Material(
                            shape: thumbnailShape,
                            clipBehavior: Clip.antiAlias,
                            child: AspectRatio(
                              aspectRatio:
                                  widget.selection.width /
                                  widget.selection.height,
                              child: Image.memory(
                                thumbnail,
                                fit: BoxFit.cover,
                                semanticLabel: l10n.postVideoThumbnailPreview,
                              ),
                            ),
                          ),
                        ),
                      ),
                    if (_thumbnail == null &&
                        widget.onThumbnailFrameChanged != null &&
                        (widget.selection.duration ?? Duration.zero) >
                            Duration.zero)
                      Center(
                        child: ConstrainedBox(
                          constraints: const BoxConstraints(maxWidth: 420),
                          child: LayoutBuilder(
                            builder: (context, constraints) => Wrap(
                              spacing: 8,
                              runSpacing: 8,
                              children: [
                                for (final frame in VideoThumbnailFrame.values)
                                  SizedBox(
                                    width: (constraints.maxWidth - 8) / 2,
                                    child: _frameOption(
                                      context,
                                      frame,
                                      ratio,
                                      thumbnailShape,
                                    ),
                                  ),
                              ],
                            ),
                          ),
                        ),
                      ),
                    if (_thumbnail != null)
                      Align(
                        alignment: AlignmentDirectional.centerStart,
                        child: TextButton(
                          onPressed: () => _setThumbnail(null),
                          style: TextButton.styleFrom(
                            foregroundColor: theme.colorScheme.error,
                          ),
                          child: Text(l10n.postVideoThumbnailRemove),
                        ),
                      )
                    else if (_available) ...[
                      const SizedBox(height: 8),
                      OutlinedButton.icon(
                        onPressed: _preparing ? null : _pickThumbnail,
                        icon: const Icon(Icons.image_outlined),
                        label: Text(l10n.postVideoThumbnailUpload),
                      ),
                    ],
                    if (_failed)
                      Text(
                        l10n.postVideoThumbnailInvalid,
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.error,
                        ),
                      ),
                    const SizedBox(height: 16),
                  ],
                  BrandTextField(
                    textFieldKey: const Key('composer-video-alt-text'),
                    label: l10n.postComposeVideoAltTextHeading,
                    labelStyle: headingStyle,
                    initialValue: widget.selection.altText,
                    maxLength: RecordTextLimits.videoAlt.graphemes,
                    maxBytes: RecordTextLimits.videoAlt.bytes,
                    maxLines: 3,
                    minLines: 3,
                    keyboardType: TextInputType.multiline,
                    textInputAction: TextInputAction.newline,
                    textCapitalization: TextCapitalization.sentences,
                    hintText: l10n.postComposeVideoAltHint,
                    onChanged: widget.onAltTextChanged,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

Uint8List _encodeThumbnail(Uint8List bytes) {
  final decoded = image.decodeImage(bytes);
  if (decoded == null) {
    throw const FormatException('Unsupported thumbnail image');
  }
  var oriented = image.bakeOrientation(decoded);
  if (oriented.width > 1280 || oriented.height > 1280) {
    oriented = oriented.width >= oriented.height
        ? image.copyResize(oriented, width: 1280)
        : image.copyResize(oriented, height: 1280);
  }
  final thumbnail = image.encodeJpg(oriented, quality: 85);
  if (thumbnail.length > 1000000) {
    throw const FormatException('Thumbnail exceeds the image size limit');
  }
  return thumbnail;
}

class _LocalPreview extends StatefulWidget {
  const new({required this.selection, required this.onFramePreview});
  final LocalVideoSelection selection;
  final void Function(VideoThumbnailFrame, Uint8List) onFramePreview;
  @override
  State<_LocalPreview> createState() => _LocalPreviewState();
}

class _LocalPreviewState extends State<_LocalPreview> {
  LocalVideoPreviewResource? _resource;
  Future<void>? _frameRequest;
  bool _failed = false;
  @override
  void initState() {
    super.initState();
    unawaited(_prepare());
  }

  Future<void> _prepare() async {
    final LocalVideoPreviewResource resource;
    try {
      resource = await createLocalVideoPreviewResource(widget.selection);
      if (!mounted) {
        await resource.dispose();
        return;
      }
      setState(() => _resource = resource);
    } on Object {
      if (mounted) setState(() => _failed = true);
      return;
    }
    final duration = widget.selection.duration;
    if (duration != null && duration > Duration.zero) {
      await (_frameRequest = _captureFrames(resource.uri, duration));
    }
  }

  Future<void> _captureFrames(Uri uri, Duration duration) async {
    for (final frame in VideoThumbnailFrame.values) {
      if (!mounted) return;
      final bytes = await readLocalVideoThumbnailFrame(uri, duration, frame);
      if (mounted && bytes != null && bytes.isNotEmpty) {
        widget.onFramePreview(frame, bytes);
      }
    }
  }

  @override
  void dispose() {
    final resource = _resource;
    if (resource != null) unawaited(_release(resource));
    super.dispose();
  }

  Future<void> _release(LocalVideoPreviewResource resource) async {
    try {
      await _frameRequest;
    } finally {
      await resource.dispose();
    }
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
