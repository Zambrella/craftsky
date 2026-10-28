import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/misc.dart' show ProviderFamily;

/// An accepted edit can replace a visible row, but cannot insert a publication
/// into another list. Keep this distinct from ordinary create overlays.
final class PostEditOverlay {
  const new(
    this.post, {
    this.revisionBound,
    this.publicationLists = const {},
    this.quote,
  });
  final Post post;
  final String? revisionBound;

  /// Latest observed partial view; never used to fabricate a full post.
  final QuotePreviewPost? quote;

  QuotePreviewPost get preview =>
      quote ??
      QuotePreviewPost(
        uri: post.uri.value,
        cid: post.cid.value,
        text: post.text,
        author: post.author,
        createdAt: post.createdAt,
        sponsored: post.sponsored,
        images: post.images,
        project: post.project,
        external: post.external,
        editedAt: post.editedAt,
        recordRevision: post.recordRevision,
      );

  String? get quoteRevisionBound => _laterRevision(
    revisionBound ?? post.recordRevision,
    quote?.recordRevision,
  );

  bool quoteAgreesWith(PdsRecordProjection value) => PdsPostEditReconciliation(
    uri: post.uri.value,
    acceptedCid: preview.cid.value,
    recordRevision: quoteRevisionBound,
  ).agrees(value);

  PostEditOverlay observedQuote(QuotePreviewPost value) => PostEditOverlay(
    post,
    revisionBound: revisionBound,
    publicationLists: publicationLists,
    quote: value,
  );

  /// Only inherited from an existing optimistic create, never from an edit.
  final Set<Object> publicationLists;

  bool agreesWith(Object? value) =>
      value is PdsRecordProjection &&
      PdsPostEditReconciliation(
        uri: post.uri.value,
        acceptedCid: post.cid.value,
        recordRevision: revisionBound ?? post.recordRevision,
      ).agrees(value);

  PostEditOverlay observed(Post value) {
    final previous = revisionBound ?? post.recordRevision;
    final revision = _laterRevision(
      value.recordRevision,
      quote?.cid == value.cid ? quote?.recordRevision : null,
    );
    final bound = _laterRevision(previous, revision);
    return PostEditOverlay(
      value,
      revisionBound: bound,
      publicationLists: publicationLists,
      quote: quoteAgreesWith(postRecordProjection(value)) ? null : quote,
    );
  }
}

String? _laterRevision(String? current, String? incoming) =>
    incoming != null && (current == null || incoming.compareTo(current) > 0)
    ? incoming
    : current;

/// Apply confirmed edits to already-rendered rows while public reads wait.
final ProviderFamily<Post, Post> postRecordPresentationProvider = Provider
    .autoDispose
    .family<Post, Post>(
      (ref, post) {
        ref
          ..watch(_postRecordOverlayRevisionProvider)
          ..watch(
            sessionRegistryProvider.select((value) => value.value?.activeLease),
          );
        // Input may already contain a local overlay. It is not a fresh public
        // observation and must not clear that overlay merely by rendering it.
        return applyPostRecordOverlay(ref, post, reconcile: false) ?? post;
      },
    );

final _postRecordOverlayRevisionProvider =
    NotifierProvider<_PostRecordOverlayRevision, int>(
      _PostRecordOverlayRevision.new,
    );

class _PostRecordOverlayRevision extends Notifier<int> {
  @override
  int build() => 0;

  void changed() => state++;
}

void notifyPostRecordOverlays(Ref ref) =>
    ref.read(_postRecordOverlayRevisionProvider.notifier).changed();

void listenForPostEdits(Ref ref, void Function() update) {
  ref.listen(_postRecordOverlayRevisionProvider, (_, _) => update());
}

List<Post> applyLoadedPostEdits(Ref ref, Iterable<Post> posts) => [
  for (final post in posts)
    ?applyPostRecordOverlay(ref, post, reconcile: false),
];

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

Post? applyPostRecordOverlay(Ref ref, Post source, {bool reconcile = true}) {
  final post = _applyQuotedPostEdit(ref, source, reconcile: reconcile);
  final controller = ref.read(pdsRecordOperationControllerProvider);
  final scope = postRecordMutationScope(ref, post);
  final overlay = controller.overlayFor(scope);
  if (overlay == null) return post;
  if (overlay.optimisticValue case final PostEditOverlay edited) {
    if (post.isProtected) return post;
    if (edited.agreesWith(postRecordProjection(post))) {
      if (reconcile) {
        controller.reconcile(scope, postRecordProjection(post));
        final current = edited.observed(post);
        controller.updateRetainedValue(
          scope,
          value: current,
          agrees: current.agreesWith,
        );
        if (current.post != edited.post) {
          // Public reads may run during provider construction. Publish the
          // new presentation after construction, without treating cached
          // cards as fresh observations or issuing another network request.
          scheduleMicrotask(() {
            if (ref.mounted) notifyPostRecordOverlays(ref);
          });
        }
      }
      return post;
    }
  }
  if (reconcile && controller.reconcile(scope, postRecordProjection(post))) {
    return post;
  }
  final optimistic = overlay.optimisticValue;
  if (optimistic case PostEditOverlay(post: final edited)) {
    if (post.isProtected) return post;
    return post.copyWith(
      cid: edited.cid,
      text: edited.text,
      tags: edited.tags,
      facets: edited.facets,
      langs: edited.langs,
      sponsored: edited.sponsored,
      images: edited.images,
      video: edited.video,
      external: edited.external,
      project: edited.project,
      editedAt: edited.editedAt,
      recordRevision: edited.recordRevision,
    );
  }
  return optimistic is Post ? optimistic : null;
}

Post _applyQuotedPostEdit(Ref ref, Post post, {required bool reconcile}) {
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
  if (overlay?.optimisticValue case final PostEditOverlay edited) {
    if (edited.post.isProtected) return post;
    final projection = PdsRecordProjection(
      uri: quoted.uri.value,
      cid: quoted.cid.value,
      recordRevision: quoted.recordRevision,
      content: const {},
    );
    if (edited.quoteAgreesWith(projection)) {
      final revision = quoted.recordRevision;
      final previous = edited.quoteRevisionBound;
      final advancesRevision =
          revision != null &&
          (previous == null || revision.compareTo(previous) > 0);
      if (reconcile && (quoted.cid != edited.preview.cid || advancesRevision)) {
        final current = edited.observedQuote(quoted);
        ref
            .read(pdsRecordOperationControllerProvider)
            .updateRetainedValue(
              scope,
              value: current,
              agrees: current.agreesWith,
            );
        scheduleMicrotask(() {
          if (ref.mounted) notifyPostRecordOverlays(ref);
        });
      }
      return post;
    }
    final latest = edited.preview;
    return post.copyWith(
      quoteView: view!.copyWith(
        post: quoted.copyWith(
          cid: latest.cid,
          text: latest.text,
          sponsored: latest.sponsored,
          images: latest.images,
          project: latest.project,
          external: latest.external,
          editedAt: latest.editedAt,
          recordRevision: latest.recordRevision,
        ),
      ),
    );
  }
  return post;
}

Iterable<Post> optimisticPostRecordOverlays(Ref ref, {Object? listKey}) sync* {
  final controller = ref.read(pdsRecordOperationControllerProvider);
  for (final overlay in controller.activeOverlays) {
    final optimistic = overlay.optimisticValue;
    if (optimistic is Post) yield optimistic;
    if (optimistic is PostEditOverlay &&
        optimistic.publicationLists.contains(listKey)) {
      yield optimistic.post;
    }
  }
}

List<Post> applyPostRecordListOverlays(
  Ref ref,
  Iterable<Post> authoritative, {
  required bool Function(Post post) includes,
  bool reconcile = true,
  Object? listKey,
}) {
  if (reconcile) observePostListMembership(ref, authoritative, listKey);
  final posts = <Post>[
    for (final post in authoritative)
      ?applyPostRecordOverlay(ref, post, reconcile: reconcile),
  ];
  final seen = posts.map((post) => post.uri).toSet();
  for (final optimistic in optimisticPostRecordOverlays(
    ref,
    listKey: listKey,
  )) {
    if (includes(optimistic) && seen.add(optimistic.uri)) {
      recordPublicationList(ref, optimistic, listKey);
      posts.insert(0, optimistic);
    }
  }
  return posts;
}

/// Only fresh collection responses acknowledge membership, never detail reads
/// or cached rows. Each previously visible list converges independently.
void observePostListMembership(Ref ref, Iterable<Post> posts, Object? listKey) {
  if (listKey == null) return;
  final controller = ref.read(pdsRecordOperationControllerProvider);
  for (final post in posts) {
    final overlay = controller.overlayFor(postRecordMutationScope(ref, post));
    if (overlay?.optimisticValue case final PostEditOverlay edited) {
      if (edited.publicationLists.contains(listKey) &&
          (post.isProtected || edited.agreesWith(postRecordProjection(post)))) {
        edited.publicationLists.remove(listKey);
      }
    }
  }
}

void recordPublicationList(Ref ref, Post post, Object? listKey) {
  if (listKey == null) return;
  final overlay = ref
      .read(pdsRecordOperationControllerProvider)
      .overlayFor(postRecordMutationScope(ref, post));
  if (overlay?.optimisticValue is Post) overlay!.publicationLists.add(listKey);
}
