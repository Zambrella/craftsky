import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/drafts/composer/instagram_post_composer.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/feed/composer/video_failure_message.dart';
import 'package:craftsky_app/feed/media/video_source_validator.dart';
import 'package:craftsky_app/feed/models/instagram_post_import_source.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/feed/widgets/local_video_preview_resource.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_published_import.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_account_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_migration_repository_provider.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_export_file_picker.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_post_archive.dart';
import 'package:craftsky_app/instagram_migration/widgets/instagram_post_import_row.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/observability/consumed_failure.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:craftsky_app/theme/yarn_refresh_scroll_view.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';
import 'package:material_ui/material_ui.dart';

enum _Work { reading, checking, preparing }

enum _Failure { read, lookup }

/// The review and its temporary export stay on the navigator while a composer
/// is open. Its archive and import history are scoped to the initiating lease.
class InstagramPostImportPage extends ConsumerStatefulWidget {
  const new({required this.lease, super.key});

  final ActiveAccountLease lease;

  @override
  ConsumerState<InstagramPostImportPage> createState() =>
      _InstagramPostImportPageState();
}

class _InstagramPostImportPageState
    extends ConsumerState<InstagramPostImportPage> {
  static const _pageSize = 20;
  final _reader = const InstagramPostArchiveReader();
  final _checkedKeys = <String>{};
  final _published = <String, InstagramPublishedImport>{};
  final _accepted = <String, Post>{};
  InstagramPostExportFile? _file;
  InstagramPostArchive? _archive;
  Future<void> _previewTail = Future<void>.value();
  _Work? _work;
  _Failure? _failure;
  int _limit = _pageSize;

  bool get _current {
    if (!mounted ||
        !ActiveAccountOperation.fromWidget(ref, widget.lease).isCurrent) {
      return false;
    }
    final link = ref
        .read(instagramAccountProvider(widget.lease))
        .value
        ?.account;
    return link?.allowsImport ?? false;
  }

  List<InstagramArchivePost> get _shown =>
      _archive?.posts.take(_limit).toList() ?? const [];

  void _clearArchive() {
    final file = _file;
    setState(() {
      _file = null;
      _archive = null;
      _checkedKeys.clear();
      _published.clear();
      _accepted.clear();
      _failure = null;
      _limit = _pageSize;
    });
    if (file != null) unawaited(file.dispose());
  }

  @override
  void dispose() {
    final file = _file;
    _file = null;
    _archive = null;
    if (file != null) unawaited(file.dispose());
    super.dispose();
  }

  void _start(_Work work) => setState(() {
    _work = work;
    _failure = null;
  });

  void _finish() {
    if (mounted) setState(() => _work = null);
  }

  Future<void> _pick() async {
    if (!_current || _work != null) return;
    final picker = ref.read(instagramPostExportFilePickerProvider);
    final reporter = ref.read(errorReporterProvider);
    final did = widget.lease.session.account.did;
    _start(_Work.reading);
    InstagramPostExportFile? file;
    try {
      file = await picker();
      if (file == null || !_current) return;
      final archive = await _reader.inspect(file.archivePath);
      if (!_current) return;
      _clearArchive();
      setState(() {
        _file = file;
        _archive = archive;
      });
      file = null;
      await _check(_shown);
    } on InstagramPostArchiveException {
      if (_current) setState(() => _failure = _Failure.read);
    } on Object catch (error, stack) {
      reportConsumedFailure(
        error,
        stack,
        reporter: reporter,
        feature: 'InstagramPostImport',
        operation: 'archive.read',
        accountDid: did,
      );
      if (_current) setState(() => _failure = _Failure.read);
    } finally {
      await file?.dispose();
      _finish();
    }
  }

  Future<bool> _check(List<InstagramArchivePost> posts) async {
    final archive = _archive;
    bool reviewIsCurrent() => _current && _archive == archive;
    if (archive == null || !reviewIsCurrent()) return false;
    if (posts.isEmpty) return true;
    final reporter = ref.read(errorReporterProvider);
    final did = widget.lease.session.account.did;
    final keys = posts.map((post) => post.sourceKey).toList();
    _checkedKeys.removeAll(keys);
    setState(() => _work = _Work.checking);
    try {
      final repository = await ref.read(
        instagramMigrationRepositoryProvider(widget.lease).future,
      );
      if (!reviewIsCurrent()) return false;
      // AppView accepts at most 100 fingerprints in a lookup.
      for (var offset = 0; offset < keys.length; offset += 100) {
        final batch = keys.skip(offset).take(100).toList();
        final items = await repository.lookupPostImports(batch);
        if (!reviewIsCurrent()) return false;
        if (items.any(
          (item) => parseCraftskyPostUri(item.uri)?.did != did,
        )) {
          throw const DiagnosticFormatException(
            'Unexpected Instagram import post owner',
          );
        }
        final results = {for (final item in items) item.sourceKey: item};
        setState(() {
          _checkedKeys.addAll(batch);
          for (final key in batch) {
            final item = results[key];
            if (item == null) {
              _published.remove(key);
            } else {
              _published[key] = item;
            }
            if (_accepted[key]?.uri != item?.uri) _accepted.remove(key);
          }
        });
      }
      return true;
    } on Object catch (error, stack) {
      reportConsumedFailure(
        error,
        stack,
        reporter: reporter,
        feature: 'InstagramPostImport',
        operation: 'history.lookup',
        accountDid: did,
      );
      if (reviewIsCurrent()) setState(() => _failure = _Failure.lookup);
      return false;
    }
  }

  Future<void> _refresh({
    bool more = false,
    bool recheckVisible = false,
  }) async {
    if (!_current || _archive == null || _work != null) return;
    _start(_Work.checking);
    if (more) setState(() => _limit += _pageSize);
    try {
      final shown = _shown;
      final posts = recheckVisible
          ? shown
          : shown
                .where((post) => !_checkedKeys.contains(post.sourceKey))
                .toList();
      await _check(posts);
    } finally {
      _finish();
    }
  }

  Future<void> _create(
    InstagramArchivePost post,
    LocalPostDraftKind kind,
  ) async {
    final archive = _archive;
    final file = _file;
    if (archive == null || file == null || !_current || _work != null) return;
    final l10n = AppLocalizations.of(context);
    final reporter = ref.read(errorReporterProvider);
    final did = widget.lease.session.account.did;
    _start(_Work.checking);
    InstagramExtractedMedia? extracted;
    var unavailableCount = 0;
    try {
      if (!await _check([post])) {
        if (mounted && _current) {
          context.showError(l10n.instagramPostImportLookupError);
        }
        return;
      }
      if (!mounted || !_current) return;
      if (_published.containsKey(post.sourceKey)) return;

      setState(() => _work = _Work.preparing);
      final accepted = await openInstagramPostComposer(
        context,
        ref,
        owner: widget.lease,
        source: InstagramPostImportSource(
          sourceKey: post.sourceKey,
          createdAt: post.createdAt,
        ),
        caption: post.caption,
        kind: kind,
        loadMedia: () async {
          if (!mounted || !_current) return null;
          final indices = await chooseInstagramPostMedia(context, post);
          if (!_current || indices == null) return null;
          final media = extracted = await _reader.extract(
            archive,
            post,
            indices,
            temporaryDirectory: file.directory.path,
            allowPartial: true,
          );
          if (!_current) return null;
          unavailableCount = media.unavailableCount;
          // Selecting all available attachments can happen automatically.
          // Acknowledge anything omitted before creating the draft.
          if (indices.length ==
              post.media.where((item) => item.available).length) {
            unavailableCount += post.media.length - indices.length;
          }
          return (
            images: media.photos.map(XFile.new).toList(),
            video: media.video == null ? null : XFile(media.video!),
          );
        },
        confirmMediaRecovery: (media) async {
          if (!mounted || !_current) return false;
          final (:readyPhotos, :readyVideo, :failedPhotos, :videoFailure) =
              media;
          if (unavailableCount + failedPhotos == 0 && videoFailure == null) {
            return true;
          }
          final confirmed = await showCraftskyConfirmDialog(
            context,
            title: l10n.instagramPostImportMediaRecoveryTitle,
            message: [
              if (videoFailure != null)
                videoSelectionFailureMessage(l10n, videoFailure),
              l10n.instagramPostImportMediaRecoveryMessage,
            ].join('\n\n'),
            cancelLabel: l10n.languageCancel,
            confirmLabel: switch ((readyPhotos, readyVideo)) {
              (_, true) => l10n.instagramPostImportContinueWithVideo,
              (0, false) => l10n.instagramPostImportContinueWithoutMedia,
              _ => l10n.instagramPostImportContinueWithPhotos(readyPhotos),
            },
          );
          return _current && confirmed;
        },
        onMediaConsumed: () async {
          await extracted?.dispose();
          extracted = null;
        },
      );
      if (!_current) return;
      if (accepted != null) {
        setState(() {
          _accepted[post.sourceKey] = accepted;
          _published[post.sourceKey] = InstagramPublishedImport(
            sourceKey: post.sourceKey,
            uri: accepted.uri,
            isProject: accepted.project != null,
          );
        });
      }
      // This catches writes confirmed after the composer was closed, including
      // a competing import from the other client.
      await _check([post]);
    } on InstagramPostArchiveException {
      if (mounted && _current) {
        context.showError(l10n.instagramPostImportUnavailableMedia);
      }
    } on VideoSourceException catch (error) {
      if (mounted && _current) {
        context.showError(videoSelectionFailureMessage(l10n, error));
      }
    } on Object catch (error, stack) {
      reportConsumedFailure(
        error,
        stack,
        reporter: reporter,
        feature: 'InstagramPostImport',
        operation: 'composer.prepare',
        accountDid: did,
      );
      if (mounted && _current) {
        context.showError(l10n.instagramPostImportOpenError);
      }
    } finally {
      await extracted?.dispose();
      _finish();
    }
  }

  Future<Uint8List?> _preview(
    InstagramPostArchive archive,
    InstagramArchivePost post,
    bool Function() thumbnailIsMounted,
  ) {
    // Prepare at most one thumbnail at a time and drop queued work when the
    // list recycles a row before extraction has started.
    bool needsPreview() =>
        thumbnailIsMounted() && _current && _archive == archive;
    final next = _previewTail.then((_) async {
      final file = _file;
      if (file == null || !needsPreview()) return null;
      final bytes = await _reader.preview(
        archive,
        post,
        temporaryDirectory: file.directory.path,
        loadVideoFrame: (uri) async {
          if (!needsPreview()) return null;
          return await readLocalVideoFrameAt(uri, Duration.zero);
        },
      );
      return needsPreview() ? bytes : null;
    });
    _previewTail = next.then((_) {}, onError: (Object _, StackTrace _) {});
    return next;
  }

  Future<void> _view(InstagramPublishedImport item) async {
    final archive = _archive;
    if (!_current || archive == null || _work != null) return;
    final parts = parseCraftskyPostUri(item.uri);
    if (parts == null || parts.did != widget.lease.session.account.did) return;
    await PostThreadRoute(
      did: parts.did,
      rkey: parts.rkey,
      $extra: _accepted[item.sourceKey],
    ).push<void>(context);
    if (_archive == archive) await _refresh(recheckVisible: true);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final registry = ref.watch(sessionRegistryProvider);
    final account = ref.watch(instagramAccountProvider(widget.lease));
    final current = registry.value?.isCurrent(widget.lease) ?? false;
    if (account case AsyncData(:final value)
        when current &&
            value.account == null &&
            TickerMode.valuesOf(context).enabled &&
            ModalRoute.isCurrentOf(context) == true) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted || ModalRoute.isCurrentOf(context) != true) return;
        if (GoRouter.maybeOf(context)?.state.matchedLocation !=
            const InstagramPostImportRoute().location) {
          return;
        }
        if (!ActiveAccountOperation.fromWidget(ref, widget.lease).isCurrent) {
          return;
        }
        if (ref.read(instagramAccountProvider(widget.lease))
            case AsyncData(:final value) when value.account == null) {
          const InstagramMigrationRoute().go(context);
        }
      });
    }
    ref
      ..listen(sessionRegistryProvider, (_, next) {
        if (next.value?.isCurrent(widget.lease) != true && _file != null) {
          _clearArchive();
        }
      })
      ..listen(instagramAccountProvider(widget.lease), (_, next) {
        if (next.hasValue &&
            next.value?.account?.allowsImport != true &&
            _file != null) {
          _clearArchive();
        }
      });
    return Scaffold(
      appBar: AppBar(title: Text(l10n.instagramPostImportTitle)),
      body: SafeArea(
        child: switch ((current, account)) {
          (true, AsyncData(:final value))
              when value.account?.allowsImport == true =>
            _buildReview(),
          (true, AsyncData(:final value)) when value.account != null => Center(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    l10n.instagramImportAccountBlocked,
                    textAlign: TextAlign.center,
                  ),
                  TextButton(
                    onPressed: () => unawaited(
                      const InstagramAccountRoute().push<void>(context),
                    ),
                    child: Text(l10n.instagramManageAccount),
                  ),
                ],
              ),
            ),
          ),
          (true, AsyncLoading()) => const Center(
            child: StitchProgressIndicator(),
          ),
          (true, AsyncError()) => Center(
            child: TextButton(
              onPressed: () => unawaited(
                ref
                    .read(instagramAccountProvider(widget.lease).notifier)
                    .refresh(),
              ),
              child: Text(l10n.instagramRetry),
            ),
          ),
          _ => Center(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Text(
                l10n.instagramPostImportVerificationRequired,
                textAlign: TextAlign.center,
              ),
            ),
          ),
        },
      ),
    );
  }

  Widget _buildReview() {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final archive = _archive;
    final shown = _shown;
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 720),
        child: YarnRefreshScrollView(
          onRefresh: () => _refresh(recheckVisible: true),
          showSuccess: false,
          slivers: [
            SliverPadding(
              padding: EdgeInsets.all(spacing.sp4),
              sliver: SliverList.builder(
                itemCount: shown.length + 2,
                itemBuilder: (context, index) {
                  if (index == 0) return _buildHeader();
                  if (index == shown.length + 1) {
                    return archive != null &&
                            shown.length < archive.posts.length
                        ? Center(
                            child: ChunkyButton(
                              key: const Key('instagram-post-import-more'),
                              variant: ChunkyButtonVariant.secondary,
                              onPressed: _work == null
                                  ? () => _refresh(more: true)
                                  : null,
                              child: Text(l10n.instagramPostImportMore),
                            ),
                          )
                        : const SizedBox.shrink();
                  }
                  final post = shown[index - 1];
                  return Padding(
                    key: ValueKey(post),
                    padding: EdgeInsets.only(bottom: spacing.sp4),
                    child: InstagramPostImportRow(
                      post: post,
                      published: _published[post.sourceKey],
                      loadPreview: (isMounted) =>
                          _preview(archive!, post, isMounted),
                      onCreate: _work == null
                          ? (kind) => _create(post, kind)
                          : null,
                      onView: _work == null
                          ? (item) => unawaited(_view(item))
                          : null,
                    ),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildHeader() {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final archive = _archive;
    final file = _file;
    final message = switch (_work) {
      _Work.reading => l10n.instagramPostImportReading,
      _Work.checking => l10n.instagramPostImportChecking,
      _Work.preparing => l10n.instagramPostImportPreparing,
      null => null,
    };
    return Padding(
      padding: EdgeInsets.only(bottom: spacing.sp4),
      child: CraftskyCard(
        key: const Key('instagram-post-import-card'),
        padding: EdgeInsets.all(spacing.sp4),
        clipBehavior: Clip.none,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (archive == null) ...[
              Text(
                l10n.instagramPostImportInstructions,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
              SizedBox(height: spacing.sp3),
              FilledButton.icon(
                key: const Key('instagram-post-import-select-zip'),
                onPressed: _work == null ? _pick : null,
                icon: const Icon(CraftskyIconsBold.openFile),
                label: Text(l10n.instagramImportSelectJson),
              ),
              SizedBox(height: spacing.sp2),
              Text(
                l10n.instagramPostImportPrivacy,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ] else
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                spacing: spacing.sp3,
                children: [
                  Text(
                    file!.name,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: theme.textTheme.titleMedium,
                  ),
                  Text(l10n.instagramPostImportCount(archive.posts.length)),
                  if (archive.skipped > 0)
                    Text(l10n.instagramPostImportSkipped(archive.skipped)),
                  TextButton(
                    key: const Key('instagram-post-import-clear'),
                    onPressed: _work == null ? _clearArchive : null,
                    child: Text(l10n.instagramPostImportClear),
                  ),
                  if (archive.posts.isEmpty)
                    Text(l10n.instagramPostImportEmpty),
                ],
              ),
            if (message != null) ...[
              SizedBox(height: spacing.sp3),
              Semantics(
                liveRegion: true,
                child: Row(
                  spacing: spacing.sp3,
                  children: [
                    const StitchProgressIndicator(),
                    Flexible(child: Text(message)),
                  ],
                ),
              ),
            ],
            if (_failure case final failure?) ...[
              SizedBox(height: spacing.sp3),
              Semantics(
                liveRegion: true,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      switch (failure) {
                        _Failure.read => l10n.instagramPostImportReadError,
                        _Failure.lookup => l10n.instagramPostImportLookupError,
                      },
                      style: TextStyle(color: theme.colorScheme.error),
                    ),
                    if (failure == _Failure.lookup)
                      TextButton(
                        onPressed: _work == null ? _refresh : null,
                        child: Text(l10n.instagramRetry),
                      ),
                  ],
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
