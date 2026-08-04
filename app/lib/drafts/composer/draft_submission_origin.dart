import 'package:craftsky_app/drafts/models/local_post_draft.dart';

/// Tracks the authoritative revision of an already-saved composer origin.
final class DraftSubmissionOrigin {
  DraftSubmissionOrigin(this._draft);

  LocalPostDraft? _draft;

  LocalPostDraft? get draft => _draft;

  void acceptSnapshot(LocalPostDraft saved) {
    final current = _draft;
    if (current == null ||
        saved.id != current.id ||
        saved.owner != current.owner ||
        saved.revision <= current.revision) {
      throw StateError('invalid local-draft recovery snapshot');
    }
    _draft = saved;
  }
}
