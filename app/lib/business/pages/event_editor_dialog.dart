import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_event_mutation_controller.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/business/widgets/event_form.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/router/responsive_modal_navigation.dart';
import 'package:craftsky_app/shared/media/blob_api_client_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

typedef EventDraftSubmit = Future<bool> Function(BusinessEventDraft draft);
typedef EventImageUpload =
    Future<UploadedImageBlob> Function(PreparedProfileImage image);

Future<void> showEventEditorSheet(
  BuildContext context, {
  BusinessEvent? event,
}) {
  return responsiveModalNavigator(context).push<void>(
    MaterialPageRoute<void>(
      fullscreenDialog: true,
      builder: (_) => EventEditorDialog(event: event),
    ),
  );
}

class EventEditorDialog extends ConsumerStatefulWidget {
  const EventEditorDialog({
    this.event,
    this.initialDraft,
    this.onSubmit,
    this.pickImage,
    this.uploadImage,
    this.confirmDiscard,
    super.key,
  }) : assert(
         event == null || initialDraft == null,
         'Provide either an event or an initial draft, not both.',
       );

  final BusinessEvent? event;
  final BusinessEventDraft? initialDraft;
  final EventDraftSubmit? onSubmit;
  final EventImagePicker? pickImage;
  final EventImageUpload? uploadImage;
  final Future<bool> Function()? confirmDiscard;

  @override
  ConsumerState<EventEditorDialog> createState() => _EventEditorDialogState();
}

class _EventEditorDialogState extends ConsumerState<EventEditorDialog> {
  GlobalKey<EventFormState> _formKey = GlobalKey<EventFormState>();
  late BusinessEvent? _event;
  late BusinessEventDraft? _initial;
  late final UnsavedWorkGuard _unsavedGuard;
  UnsavedWorkRegistration? _unsavedRegistration;
  AccountSessionLease? _unsavedOwner;
  bool _dirty = false;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _event = widget.event;
    _initial =
        widget.initialDraft ??
        (_event == null
            ? null
            : BusinessEventDraft.fromEvent(
                _event!,
                timeZones: ref.read(businessTimeZoneServiceProvider),
              ));
    _unsavedGuard = ref.read(unsavedWorkGuardProvider);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _ensureUnsavedRegistration();
  }

  @override
  void dispose() {
    _unsavedGuard.unregister(_unsavedRegistration);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final mutation = ref.watch(businessEventMutationControllerProvider);
    final conflict = mutation.status == EventMutationStatus.conflict;
    final submitLabel = _event == null
        ? l10n.businessEventCreateTitle
        : l10n.businessEventSave;
    return PopScope<Object?>(
      canPop: !_dirty || _saving,
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        if (await _confirmDiscard() && context.mounted) {
          Navigator.of(context).pop();
        }
      },
      child: Scaffold(
        backgroundColor: swatches.paper,
        appBar: AppBar(
          leading: CloseButton(onPressed: _requestClose),
          title: Text(
            _event == null
                ? l10n.businessEventCreateTitle
                : l10n.businessEventEditTitle,
          ),
        ),
        body: Column(
          children: [
            if (conflict)
              MaterialBanner(
                content: Text(l10n.businessEventConflict),
                actions: [
                  TextButton(
                    onPressed: _reloadConflict,
                    child: Text(l10n.businessEventReload),
                  ),
                ],
              )
            else if (mutation.status == EventMutationStatus.error &&
                mutation.validationErrors.isEmpty)
              MaterialBanner(
                content: Text(l10n.businessEventSaveError),
                actions: const [SizedBox.shrink()],
              ),
            Expanded(
              child: Stack(
                children: [
                  EventForm(
                    key: _formKey,
                    initial: _initial,
                    pickImage: widget.pickImage,
                    enabled: !_saving && !conflict,
                    onChanged: () {
                      if (!_dirty) setState(() => _dirty = true);
                    },
                  ),
                  PositionedDirectional(
                    start: spacing.sp4,
                    end: spacing.sp4,
                    bottom: 0,
                    child: SafeArea(
                      top: false,
                      minimum: EdgeInsets.only(bottom: spacing.sp4),
                      child: ChunkyButton(
                        key: const ValueKey('event-submit'),
                        onPressed: _saving || conflict ? null : _submit,
                        style: ButtonStyle(
                          minimumSize: WidgetStatePropertyAll(
                            Size.fromHeight(spacing.sp7),
                          ),
                        ),
                        child: _saving
                            ? SizedBox.square(
                                dimension: spacing.sp5,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                  semanticsLabel: l10n.businessSaving,
                                ),
                              )
                            : Text(submitLabel),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _submit() async {
    final submission = _formKey.currentState?.submit();
    if (submission == null) return;
    setState(() => _saving = true);
    var draft = submission.draft;
    final pendingImage = submission.pendingImage;
    if (pendingImage != null) {
      final ownership = ref.read(sessionRegistryProvider).value?.activeLease;
      try {
        final uploaded =
            await (widget.uploadImage?.call(pendingImage) ??
                ref
                    .read(blobApiClientProvider)
                    .uploadImage(
                      bytes: pendingImage.bytes,
                      mimeType: pendingImage.mimeType,
                    ));
        if (!_isCurrent(ownership)) {
          if (mounted) setState(() => _saving = false);
          return;
        }
        final image = UploadedBusinessImageDraft.fromUpload(
          uploaded,
          alt: submission.imageAlt,
          aspectRatio: BusinessImageAspectRatio(
            width: pendingImage.width,
            height: pendingImage.height,
          ),
          previewBytes: pendingImage.bytes,
        );
        _formKey.currentState?.acceptUploadedImage(pendingImage, image);
        draft = draft.copyWith(image: image);
      } on Object {
        _formKey.currentState?.showImageUploadError();
        if (mounted) setState(() => _saving = false);
        return;
      }
    }
    final success = await (widget.onSubmit?.call(draft) ?? _mutate(draft));
    if (!mounted) return;
    setState(() {
      _saving = false;
      if (success) _dirty = false;
    });
    if (success && Navigator.of(context).canPop()) {
      Navigator.of(context).pop();
    }
  }

  bool _isCurrent(ActiveAccountLease? ownership) {
    if (!mounted) return false;
    if (ownership == null) return true;
    return ref.read(sessionRegistryProvider).value?.isCurrent(ownership) ??
        false;
  }

  Future<void> _requestClose() async {
    if (!_dirty || await _confirmDiscard()) {
      if (mounted && Navigator.of(context).canPop()) {
        Navigator.of(context).pop();
      }
    }
  }

  Future<bool> _mutate(BusinessEventDraft draft) {
    final controller = ref.read(
      businessEventMutationControllerProvider.notifier,
    );
    final event = _event;
    return event == null
        ? controller.create(draft)
        : controller.update(event, draft);
  }

  Future<void> _reloadConflict() async {
    final event = await ref
        .read(businessEventMutationControllerProvider.notifier)
        .reloadConflict();
    if (event != null && mounted) {
      setState(() {
        _event = event;
        _initial = BusinessEventDraft.fromEvent(
          event,
          timeZones: ref.read(businessTimeZoneServiceProvider),
        );
        _formKey = GlobalKey<EventFormState>();
        _dirty = false;
      });
    }
  }

  Future<bool> _confirmDiscard() async {
    final injected = widget.confirmDiscard;
    if (injected != null) return injected();
    final l10n = AppLocalizations.of(context);
    return showCraftskyConfirmDialog(
      context,
      title: l10n.businessEventDiscardTitle,
      message: l10n.businessEventDiscardMessage,
      confirmLabel: l10n.businessEventDiscard,
      cancelLabel: l10n.businessEventKeepEditing,
    );
  }

  void _ensureUnsavedRegistration() {
    final owner = ref.read(sessionRegistryProvider).value?.activeLease?.session;
    if (owner == null || owner == _unsavedOwner) return;
    _unsavedOwner = owner;
    _unsavedRegistration = _unsavedGuard.replace(
      _unsavedRegistration,
      owner: owner,
      isDirty: () => mounted && _dirty,
      confirmAndClose: () async {
        if (!mounted || !await _confirmDiscard() || !mounted) return false;
        Navigator.of(context).pop();
        await Future<void>.delayed(Duration.zero);
        return true;
      },
    );
  }
}
