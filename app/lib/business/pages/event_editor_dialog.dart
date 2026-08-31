import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/providers/business_event_mutation_controller.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/business/widgets/event_form.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

typedef EventDraftSubmit = Future<bool> Function(BusinessEventDraft draft);

class EventEditorDialog extends ConsumerStatefulWidget {
  const EventEditorDialog({
    this.event,
    this.initialDraft,
    this.onSubmit,
    this.pickImage,
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
        appBar: AppBar(
          leading: CloseButton(onPressed: _requestClose),
          title: Text(
            _event == null
                ? l10n.businessEventCreateTitle
                : l10n.businessEventEditTitle,
          ),
          actions: [
            Padding(
              padding: const EdgeInsets.only(right: 12),
              child: IconButton.filled(
                key: const ValueKey('event-submit'),
                tooltip: submitLabel,
                onPressed: _saving || conflict ? null : _submit,
                icon: _saving
                    ? SizedBox.square(
                        dimension: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          semanticsLabel: l10n.businessSaving,
                        ),
                      )
                    : Icon(_event == null ? Icons.add : Icons.save_outlined),
              ),
            ),
          ],
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
              child: EventForm(
                key: _formKey,
                initial: _initial,
                pickImage: widget.pickImage,
                enabled: !_saving && !conflict,
                onChanged: (_) {
                  if (!_dirty) setState(() => _dirty = true);
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _submit() async {
    final draft = _formKey.currentState?.submit();
    if (draft == null) return;
    setState(() => _saving = true);
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
    return await showDialog<bool>(
          context: context,
          builder: (context) => AlertDialog(
            title: Text(l10n.businessEventDiscardTitle),
            content: Text(l10n.businessEventDiscardMessage),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: Text(l10n.businessEventKeepEditing),
              ),
              FilledButton(
                onPressed: () => Navigator.pop(context, true),
                child: Text(l10n.businessEventDiscard),
              ),
            ],
          ),
        ) ??
        false;
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
