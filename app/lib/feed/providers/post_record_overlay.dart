import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// An accepted edit can replace a visible row, but cannot insert a publication
/// into another list. Keep this distinct from ordinary create overlays.
final class PostEditOverlay {
  const new(this.post);
  final Post post;
}

PdsMutationScope postRecordMutationScope(Ref ref, Post post) =>
    postRecordMutationScopeForUri(
      ref,
      post.uri.toString(),
      post.author.did.toString(),
    );

PdsMutationScope postRecordMutationScopeForUri(
  Ref ref,
  String uri,
  String ownerDid,
) {
  final lease =
      captureActiveAccountOperation(ref)?.session ??
      AccountSessionLease(
        account: AccountKey(ownerDid),
        sessionGeneration: 0,
      );
  return PdsMutationScope(lease: lease, identity: 'post:$uri');
}

PdsMutationScope postCreateOperationScope(
  ActiveAccountLease? ownership,
) => PdsMutationScope(
  lease:
      ownership?.session ??
      AccountSessionLease(
        account: AccountKey('did:plc:local-test'),
        sessionGeneration: 0,
      ),
  identity: 'post-create',
);

PdsRecordProjection postRecordProjection(Post? post) => PdsRecordProjection(
  uri: post?.uri.toString() ?? '',
  cid: post?.cid.toString() ?? '',
  recordRevision: post?.recordRevision,
  content: post == null
      ? const {}
      : <String, Object?>{
          'text': post.text,
          'langs': post.langs,
          'sponsored': post.sponsored,
        },
);

Post? applyPostRecordOverlay(Ref ref, Post source) {
  final post = _applyQuotedPostEdit(ref, source);
  final controller = ref.read(pdsRecordOperationControllerProvider);
  final scope = postRecordMutationScope(ref, post);
  final overlay = controller.overlayFor(scope);
  if (overlay == null) return post;
  if (controller.reconcile(scope, postRecordProjection(post))) return post;
  final optimistic = overlay.optimisticValue;
  if (optimistic case PostEditOverlay(post: final edited)) {
    if (post.isProtected) return post;
    return post.copyWith(
      cid: edited.cid,
      text: edited.text,
      facets: edited.facets,
      langs: edited.langs,
      sponsored: edited.sponsored,
      images: edited.images,
      video: edited.video,
      external: edited.external,
      project: edited.project,
      editedAt: edited.editedAt,
      editTimeSource: edited.editTimeSource,
      recordRevision: edited.recordRevision,
    );
  }
  return optimistic is Post ? optimistic : null;
}

Post _applyQuotedPostEdit(Ref ref, Post post) {
  final view = post.quoteView;
  final quoted = view?.post;
  if (post.isProtected || view?.state != 'visible' || quoted == null) {
    return post;
  }
  final scope = postRecordMutationScopeForUri(
    ref,
    quoted.uri.value,
    quoted.author.did.value,
  );
  final overlay = ref
      .read(pdsRecordOperationControllerProvider)
      .overlayFor(scope);
  if (overlay?.optimisticValue case PostEditOverlay(post: final edited)) {
    if (edited.isProtected ||
        overlay!.agreesWith(
          PdsRecordProjection(
            uri: quoted.uri.value,
            cid: quoted.cid.value,
            recordRevision: quoted.recordRevision,
            content: const {},
          ),
        )) {
      return post;
    }
    return post.copyWith(
      quoteView: view!.copyWith(
        post: quoted.copyWith(
          cid: edited.cid,
          text: edited.text,
          sponsored: edited.sponsored,
          images: edited.images,
          project: edited.project,
          external: edited.external,
          editedAt: edited.editedAt,
          editTimeSource: edited.editTimeSource,
          recordRevision: edited.recordRevision,
        ),
      ),
    );
  }
  return post;
}

Iterable<Post> optimisticPostRecordOverlays(Ref ref) sync* {
  final controller = ref.read(pdsRecordOperationControllerProvider);
  for (final overlay in controller.activeOverlays) {
    final optimistic = overlay.optimisticValue;
    if (optimistic is Post) yield optimistic;
  }
}

List<Post> applyPostRecordListOverlays(
  Ref ref,
  Iterable<Post> authoritative, {
  required bool Function(Post post) includes,
}) {
  final posts = <Post>[
    for (final post in authoritative) ?applyPostRecordOverlay(ref, post),
  ];
  final seen = posts.map((post) => post.uri).toSet();
  for (final optimistic in optimisticPostRecordOverlays(ref)) {
    if (includes(optimistic) && seen.add(optimistic.uri)) {
      posts.insert(0, optimistic);
    }
  }
  return posts;
}
