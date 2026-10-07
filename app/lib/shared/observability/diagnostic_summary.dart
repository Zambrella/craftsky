import 'package:at_primitives/at_identifier.dart'
    show InvalidDidError, InvalidHandleError;
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/active_account_identity_provider.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/providers/business_event_detail_provider.dart';
import 'package:craftsky_app/business/providers/profile_business_events_provider.dart';
import 'package:craftsky_app/settings/models/settings_identity.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/widgets/post_summary.dart';

// Summary calls are explicit. They never serialize models, UI labels, free
// prose, URLs, cursors, viewer relationships, errors or session generations.
extension AuthDiagnosticSummary on AuthState {
  DiagnosticWorkflow diagnosticSummary({bool publicWorkflow = false}) =>
      ModelDiagnosticSummary(
        model: DiagnosticModel.auth,
        state: this is SignedIn
            ? DiagnosticModelState.signedIn
            : DiagnosticModelState.signedOut,
        record: publicWorkflow && this is SignedIn
            ? PublicRecordContext(
                actorDid: (this as SignedIn).did.value,
                handle: (this as SignedIn).handle.value,
              )
            : null,
      );
}

extension SettingsDiagnosticSummary on SettingsIdentity {
  DiagnosticWorkflow diagnosticSummary({bool publicWorkflow = false}) =>
      ModelDiagnosticSummary(
        model: DiagnosticModel.settingsIdentity,
        state: DiagnosticModelState.available,
        record: publicWorkflow ? _settingsReferences(this) : null,
      );
}

extension IdentityDiagnosticSummary on ActiveAccountIdentity {
  DiagnosticWorkflow diagnosticSummary({bool publicWorkflow = false}) =>
      ModelDiagnosticSummary(
        model: DiagnosticModel.activeIdentity,
        state: DiagnosticModelState.available,
        record: publicWorkflow
            ? PublicRecordContext(
                actorDid: profile.did.value,
                handle: profile.handle.value,
              )
            : null,
      );
}

extension PostDiagnosticSummary on PostSummaryData {
  DiagnosticWorkflow diagnosticSummary({bool publicWorkflow = false}) =>
      ModelDiagnosticSummary(
        model: DiagnosticModel.post,
        state: state == PostSummaryState.visible
            ? DiagnosticModelState.visible
            : DiagnosticModelState.unavailable,
        record:
            publicWorkflow &&
                state == PostSummaryState.visible &&
                author != null
            ? PublicRecordContext(
                targetDid: author!.did.value,
                handle: author!.handle.value,
              )
            : null,
        hasImage: image != null,
      );
}

extension EventDiagnosticSummary on BusinessEvent {
  DiagnosticWorkflow diagnosticSummary({bool publicWorkflow = false}) =>
      ModelDiagnosticSummary(
        model: DiagnosticModel.businessEvent,
        state: switch (status.value) {
          'scheduled' => DiagnosticModelState.scheduled,
          'canceled' => DiagnosticModelState.canceled,
          'completed' => DiagnosticModelState.completed,
          _ => DiagnosticModelState.unknown,
        },
        record: publicWorkflow
            ? PublicRecordContext(
                targetDid: did.value,
                recordUri: uri.value,
                cid: cid.value,
                nsid: 'social.craftsky.business.event',
                recordKey: rkey.value,
              )
            : null,
      );
}

extension EventDetailDiagnosticSummary on BusinessEventDetailState {
  DiagnosticWorkflow diagnosticSummary({bool publicWorkflow = false}) =>
      switch (this) {
        BusinessEventDetailAvailable(:final event) => event.diagnosticSummary(
          publicWorkflow: publicWorkflow,
        ),
        _ => const ModelDiagnosticSummary(
          model: DiagnosticModel.businessEvent,
          state: DiagnosticModelState.unavailable,
        ),
      };
}

extension EventListDiagnosticSummary on BusinessEventListState {
  DiagnosticWorkflow diagnosticSummary({bool publicWorkflow = false}) =>
      ModelDiagnosticSummary(
        model: DiagnosticModel.businessEventList,
        state: DiagnosticModelState.available,
        itemCount: items.length,
        hasMore: hasMore,
      );
}

PublicRecordContext _settingsReferences(SettingsIdentity identity) {
  String? did;
  String? handle;
  try {
    did = Did.parse(identity.avatarSeed).value;
    // The identifier SDK uses Error subclasses for validation failures.
    // ignore: avoid_catching_errors
  } on InvalidDidError {
    /* no guessed identity */
  }
  try {
    handle = Handle.parse(
      identity.handleLabel.replaceFirst(RegExp('^@'), ''),
    ).value;
    // The identifier SDK uses Error subclasses for validation failures.
    // ignore: avoid_catching_errors
  } on InvalidHandleError {
    /* UI placeholder is not a handle */
  }
  return PublicRecordContext(actorDid: did, handle: handle);
}
