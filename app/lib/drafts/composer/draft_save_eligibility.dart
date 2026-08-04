/// Composer origins relevant to first-release local draft eligibility.
enum DraftComposerKind { standard, project, quote, reply }

/// Deliberate differences from the composer's initial editable state.
class DraftComposerChanges {
  const DraftComposerChanges({
    this.textChanged = false,
    this.attachmentChanged = false,
    this.altTextChanged = false,
    this.languageChanged = false,
    this.projectFieldChanged = false,
    this.scheduleChanged = false,
  });

  final bool textChanged;
  final bool attachmentChanged;
  final bool altTextChanged;
  final bool languageChanged;
  final bool projectFieldChanged;
  final bool scheduleChanged;

  bool get isMeaningful =>
      textChanged ||
      attachmentChanged ||
      altTextChanged ||
      languageChanged ||
      projectFieldChanged ||
      scheduleChanged;
}

/// Pure policy for whether the current composer may be explicitly saved.
abstract final class DraftSaveEligibility {
  static bool canSave({
    required DraftComposerKind kind,
    required DraftComposerChanges changes,
    required bool mediaReady,
  }) {
    final eligibleKind =
        kind == DraftComposerKind.standard || kind == DraftComposerKind.project;
    return eligibleKind && changes.isMeaningful && mediaReady;
  }
}
