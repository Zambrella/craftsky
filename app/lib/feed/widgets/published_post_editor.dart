import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/feed/composer/post_edit_origin.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_edit_seed_provider.dart';
import 'package:craftsky_app/feed/widgets/post_composer_sheet.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/projects/widgets/project_composer_sheet.dart';
import 'package:craftsky_app/router/responsive_modal_navigation.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';
import 'package:uuid/uuid.dart';

Future<Post?> showPublishedPostEditor(
  BuildContext context, {
  required Post post,
  required ActiveAccountLease ownership,
}) => responsiveModalNavigator(context).push<Post?>(
  FullscreenModalRoute<Post?>(
    builder: (_) => _PublishedPostEditor(post: post, ownership: ownership),
  ),
);

final class _PublishedPostEditor extends ConsumerStatefulWidget {
  const new({required this.post, required this.ownership});
  final Post post;
  final ActiveAccountLease ownership;

  @override
  ConsumerState<_PublishedPostEditor> createState() =>
      _PublishedPostEditorState();
}

final class _PublishedPostEditorState
    extends ConsumerState<_PublishedPostEditor> {
  // Separate openings, including conflict review, never invalidate a live
  // editor's recoverable input or reuse its older cached base.
  final String _requestId = const Uuid().v4();

  @override
  Widget build(BuildContext context) {
    final registry = ref.watch(sessionRegistryProvider);
    if (registry.isLoading) return _status(context, loading: true);
    if (registry.value?.activeLease != widget.ownership) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted && Navigator.of(context).canPop()) {
          Navigator.of(context).pop();
        }
      });
      return const SizedBox.shrink();
    }
    final provider = postEditSeedProvider(
      widget.ownership,
      widget.post.author.did,
      widget.post.rkey,
      requestId: _requestId,
    );
    final state = ref.watch(provider);
    return switch (state) {
      AsyncData(:final value) =>
        value.post.project == null
            ? PostComposerSheet(
                key: ValueKey(value.baseCid),
                publishedEdit: PostEditOrigin(
                  seed: value,
                  ownership: widget.ownership,
                ),
              )
            : ProjectComposerSheet(
                key: ValueKey(value.baseCid),
                publishedEdit: PostEditOrigin(
                  seed: value,
                  ownership: widget.ownership,
                ),
              ),
      AsyncLoading() => _status(context, loading: true),
      _ => _status(
        context,
        loading: false,
        retry: () => ref.invalidate(provider),
      ),
    };
  }

  Widget _status(
    BuildContext context, {
    required bool loading,
    VoidCallback? retry,
  }) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.postEditTitle)),
      body: Center(
        child: loading
            ? const StitchProgressIndicator(size: 32)
            : Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(l10n.postEditLoadError),
                  TextButton(onPressed: retry, child: Text(l10n.retryButton)),
                ],
              ),
      ),
    );
  }
}
