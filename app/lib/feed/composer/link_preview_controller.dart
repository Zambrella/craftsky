import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/feed/composer/link_preview_candidate.dart';
import 'package:craftsky_app/feed/composer/link_preview_candidates.dart';
import 'package:craftsky_app/feed/models/link_preview.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'link_preview_controller.g.dart';

// The interface keeps the controller independent from the AppView API client.
// ignore: one_member_abstracts
abstract interface class LinkPreviewRepository {
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken);
}

final class ApiLinkPreviewRepository implements LinkPreviewRepository {
  const ApiLinkPreviewRepository(this._fetch);

  final Future<LinkPreview> Function(String, {CancelToken? cancelToken}) _fetch;

  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) =>
      _fetch(url.toString(), cancelToken: cancelToken);
}

@Riverpod(keepAlive: true)
LinkPreviewRepository linkPreviewRepository(Ref ref) =>
    ApiLinkPreviewRepository(ref.watch(postApiClientProvider).fetchLinkPreview);

final class LinkPreviewSessionState {
  const LinkPreviewSessionState({
    this.candidates = const [],
    this.selectedIdentity,
    this.inFlightIdentity,
    this.suppressed = false,
    this.dismissed = false,
    this.canUndoDismiss = false,
  });

  final List<LinkPreviewCandidate> candidates;
  final Uri? selectedIdentity;
  final Uri? inFlightIdentity;
  final bool suppressed;
  final bool dismissed;
  final bool canUndoDismiss;

  LinkPreviewSessionState copyWith({
    List<LinkPreviewCandidate>? candidates,
    Object? selectedIdentity = _unset,
    Object? inFlightIdentity = _unset,
    bool? suppressed,
    bool? dismissed,
    bool? canUndoDismiss,
  }) => LinkPreviewSessionState(
    candidates: candidates ?? this.candidates,
    selectedIdentity: selectedIdentity == _unset
        ? this.selectedIdentity
        : selectedIdentity as Uri?,
    inFlightIdentity: inFlightIdentity == _unset
        ? this.inFlightIdentity
        : inFlightIdentity as Uri?,
    suppressed: suppressed ?? this.suppressed,
    dismissed: dismissed ?? this.dismissed,
    canUndoDismiss: canUndoDismiss ?? this.canUndoDismiss,
  );
}

const _unset = Object();

final class SelectedLinkPreview {
  const SelectedLinkPreview({required this.candidate, required this.preview});

  final LinkPreviewCandidate candidate;
  final LinkPreview preview;

  Uri get navigationUri => candidate.navigationUri(preview.url);
}

@riverpod
class LinkPreviewController extends _$LinkPreviewController {
  final _successes = <Uri, LinkPreview>{};
  final _failures = <Uri>{};
  CancelToken? _cancelToken;
  var _generation = 0;
  var _dismissGeneration = 0;
  var _submitInvalidated = false;
  var _text = '';
  Uri? _seededIdentity;

  @override
  LinkPreviewSessionState build(String composerId, AccountKey accountKey) {
    ref.onDispose(() => _cancelActive(updateState: false));
    return const LinkPreviewSessionState();
  }

  SelectedLinkPreview? get selected {
    final identity = state.selectedIdentity;
    if (identity == null) return null;
    final preview = _successes[identity];
    if (preview == null) return null;
    final candidate = state.candidates
        .where((candidate) => candidate.identity == identity)
        .firstOrNull;
    return candidate == null
        ? null
        : SelectedLinkPreview(candidate: candidate, preview: preview);
  }

  List<SelectedLinkPreview> get available => [
    for (final candidate in state.candidates)
      if (_successes[candidate.identity] case final preview?)
        SelectedLinkPreview(candidate: candidate, preview: preview),
  ];

  bool get dismissed => state.dismissed;

  void seed(SelectedLinkPreview selection) {
    final identity = selection.candidate.identity;
    _seededIdentity = identity;
    _successes[identity] = selection.preview;
    state = state.copyWith(
      candidates: [
        selection.candidate,
        for (final candidate in state.candidates)
          if (candidate.identity != identity) candidate,
      ],
      selectedIdentity: identity,
    );
  }

  void refreshSeed(SelectedLinkPreview selection) {
    final identity = selection.candidate.identity;
    final remainsCandidate = state.candidates.any(
      (candidate) => candidate.identity == identity,
    );
    if (!remainsCandidate || !_successes.containsKey(identity)) return;
    _successes[identity] = selection.preview;
    state = state.copyWith(candidates: List.of(state.candidates));
  }

  void updateText(String text) {
    if (text != _text) {
      _text = text;
      _submitInvalidated = false;
    }
    final candidates = deriveLinkPreviewCandidates(
      text,
      retainedIdentity: _seededIdentity,
    );
    final identities = {for (final candidate in candidates) candidate.identity};
    if (state.inFlightIdentity case final active?
        when !identities.contains(active)) {
      _cancelActive();
    }
    final selected = identities.contains(state.selectedIdentity)
        ? state.selectedIdentity
        : null;
    state = state.copyWith(
      candidates: candidates,
      selectedIdentity: selected,
    );
    _selectEarliestSuccess();
    _startNext();
  }

  void setSuppressed({required bool value}) {
    if (state.suppressed == value) return;
    if (value) _cancelActive();
    state = state.copyWith(suppressed: value);
    if (!value) _startNext();
  }

  int dismiss() {
    if (state.dismissed) return _dismissGeneration;
    _cancelActive();
    _dismissGeneration += 1;
    state = state.copyWith(dismissed: true, canUndoDismiss: true);
    return _dismissGeneration;
  }

  void undoDismiss() {
    if (!state.dismissed || !state.canUndoDismiss) return;
    state = state.copyWith(dismissed: false, canUndoDismiss: false);
    _startNext();
  }

  void expireUndo(int dismissGeneration) {
    if (dismissGeneration != _dismissGeneration || !state.canUndoDismiss) {
      return;
    }
    state = state.copyWith(canUndoDismiss: false);
  }

  void selectNext() => _moveSelection(1);

  void selectPrevious() => _moveSelection(-1);

  SelectedLinkPreview? snapshotForSubmit() {
    final selection = state.suppressed || state.dismissed ? null : selected;
    _submitInvalidated = true;
    _cancelActive();
    return selection;
  }

  void _moveSelection(int delta) {
    final available = [
      for (final candidate in state.candidates)
        if (_successes.containsKey(candidate.identity)) candidate.identity,
    ];
    if (available.isEmpty) return;
    final current = available.indexWhere(
      (identity) => identity == state.selectedIdentity,
    );
    final index = current < 0 ? 0 : (current + delta) % available.length;
    state = state.copyWith(selectedIdentity: available[index]);
  }

  void _selectEarliestSuccess() {
    if (state.selectedIdentity != null) return;
    for (final candidate in state.candidates) {
      if (_successes.containsKey(candidate.identity)) {
        state = state.copyWith(selectedIdentity: candidate.identity);
        return;
      }
    }
  }

  void _startNext() {
    if (_submitInvalidated ||
        state.suppressed ||
        state.dismissed ||
        state.inFlightIdentity != null) {
      return;
    }
    for (final candidate in state.candidates) {
      if (_successes.containsKey(candidate.identity) ||
          _failures.contains(candidate.identity)) {
        continue;
      }
      final token = CancelToken();
      final generation = ++_generation;
      _cancelToken = token;
      state = state.copyWith(inFlightIdentity: candidate.identity);
      unawaited(_fetch(candidate, token, generation));
      return;
    }
  }

  Future<void> _fetch(
    LinkPreviewCandidate candidate,
    CancelToken token,
    int generation,
  ) async {
    try {
      final preview = await ref
          .read(linkPreviewRepositoryProvider)
          .fetch(candidate.transportUri, token);
      if (!ref.mounted || generation != _generation || token.isCancelled) {
        return;
      }
      _successes[candidate.identity] = preview;
    } on Object {
      if (!ref.mounted || generation != _generation || token.isCancelled) {
        return;
      }
      _failures.add(candidate.identity);
    }
    if (!ref.mounted || generation != _generation) return;
    _cancelToken = null;
    state = state.copyWith(inFlightIdentity: null);
    _selectEarliestSuccess();
    _startNext();
  }

  void _cancelActive({bool updateState = true}) {
    _generation++;
    _cancelToken?.cancel();
    _cancelToken = null;
    if (updateState && ref.mounted && state.inFlightIdentity != null) {
      state = state.copyWith(inFlightIdentity: null);
    }
  }
}
