import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_labels.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/shared/image/basic_image_editor.dart';
import 'package:craftsky_app/shared/media/image_pick_error_reporting.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:craftsky_app/theme/craftsky_field_scaffold.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:material_ui/material_ui.dart';

typedef EventImagePicker = Future<PreparedProfileImage?> Function(
  ImageSource source,
  void Function(Uint8List bytes) onPreviewReady,
);

class EventFormSubmission {
  const new({
    required this.draft,
    required this.pendingImage,
    required this.imageAlt,
  });

  final BusinessEventDraft draft;
  final PreparedProfileImage? pendingImage;
  final String imageAlt;
}

class EventForm extends ConsumerStatefulWidget {
  const new({
    required this.onChanged,
    this.initial,
    this.pickImage,
    this.editImage,
    this.enabled = true,
    super.key,
  });

  final BusinessEventDraft? initial;
  final VoidCallback onChanged;
  final EventImagePicker? pickImage;
  final BasicImageEditor? editImage;
  final bool enabled;

  @override
  ConsumerState<EventForm> createState() => EventFormState();
}

class EventFormState extends ConsumerState<EventForm> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _name;
  late final FocusNode _startFocus;
  late final FocusNode _endFocus;
  late final TextEditingController _summary;
  late final TextEditingController _venue;
  late final TextEditingController _eventUri;
  late final TextEditingController _registrationUri;
  late final TextEditingController _alt;
  late Set<String> _roles;
  late String _mode;
  late String _status;
  late String _timeZone;
  late bool _isAllDay;
  DateTime? _start;
  DateTime? _end;
  late BusinessImageDraft _image;
  PreparedProfileImage? _pendingImage;
  Uint8List? _preview;
  String? _draftError;
  String? _uploadError;
  bool _preparingImage = false;

  @override
  void initState() {
    super.initState();
    final initial = widget.initial;
    _name = TextEditingController(text: initial?.name);
    _startFocus = FocusNode();
    _endFocus = FocusNode();
    _start = _civilTime(initial?.startsAt);
    _end = _civilTime(initial?.endsAt);
    _summary = TextEditingController(text: initial?.summary);
    _venue = TextEditingController(text: initial?.venueName);
    _eventUri = TextEditingController(text: initial?.eventUri);
    _registrationUri = TextEditingController(text: initial?.registrationUri);
    _image = initial?.image ?? const MissingBusinessImageDraft();
    _alt = TextEditingController(text: _image.alt);
    _roles = {...?initial?.roles};
    _mode = initial?.mode ?? 'in-person';
    _status = initial?.status ?? 'scheduled';
    _timeZone = initial?.timeZone ?? 'UTC';
    _isAllDay = initial?.isAllDay ?? false;
  }

  @override
  void dispose() {
    for (final controller in [
      _name,
      _summary,
      _venue,
      _eventUri,
      _registrationUri,
      _alt,
    ]) {
      controller.dispose();
    }
    _startFocus.dispose();
    _endFocus.dispose();
    super.dispose();
  }

  EventFormSubmission? submit() {
    final formValid = _formKey.currentState?.validate() ?? false;
    final draft = _buildDraft();
    final errors = draft?.validate(ref.read(businessTimeZoneServiceProvider));
    final l10n = AppLocalizations.of(context);
    setState(() {
      if (errors?.contains(EventDraftError.timeDoesNotExist) ?? false) {
        _draftError = l10n.businessEventTimeDoesNotExist;
      } else {
        _draftError = draft == null || (errors?.isNotEmpty ?? true)
            ? l10n.businessEventValidationError
            : null;
      }
    });
    if (!formValid || draft == null || errors!.isNotEmpty || _preparingImage) {
      return null;
    }
    return EventFormSubmission(
      draft: draft,
      pendingImage: _pendingImage,
      imageAlt: _alt.text,
    );
  }

  void acceptUploadedImage(
    PreparedProfileImage pending,
    UploadedBusinessImageDraft uploaded,
  ) {
    if (!identical(_pendingImage, pending)) return;
    setState(() {
      _image = uploaded;
      _pendingImage = null;
      _preview = pending.bytes;
      _uploadError = null;
    });
  }

  void showImageUploadError() {
    if (!mounted) return;
    setState(() {
      _uploadError = AppLocalizations.of(context).businessEventUploadError;
    });
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return Form(
      key: _formKey,
      onChanged: _notifyChanged,
      child: LayoutBuilder(
        builder: (context, constraints) {
          final wide = constraints.maxWidth >= 720;
          return SingleChildScrollView(
            padding: EdgeInsets.all(spacing.sp4),
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 880),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    CraftskyTextFormField(
                      textFieldKey: const ValueKey('event-name'),
                      controller: _name,
                      enabled: widget.enabled,
                      maxLength: businessEventNameLimit,
                      maxBytes: RecordTextLimits.eventName.bytes,
                      textCapitalization: TextCapitalization.words,
                      label: l10n.businessEventNameLabel,
                      hintText: l10n.businessEventNameHint,
                      required: true,
                      validator: (value) =>
                          value == null || value.trim().isEmpty
                          ? l10n.businessEventNameRequired
                          : null,
                    ),
                    SizedBox(height: spacing.sp5),
                    SwitchListTile(
                      key: const ValueKey('event-all-day'),
                      contentPadding: EdgeInsets.zero,
                      title: Text(l10n.businessEventAllDay),
                      value: _isAllDay,
                      onChanged: widget.enabled ? _setAllDay : null,
                    ),
                    SizedBox(height: spacing.sp5),
                    _responsivePair(
                      wide,
                      _boundaryField(
                        key: const ValueKey('event-start'),
                        value: _start,
                        focusNode: _startFocus,
                        label: l10n.businessEventStartLabel,
                        onSelected: (value) => _start = value,
                      ),
                      _boundaryField(
                        key: const ValueKey('event-end'),
                        value: _end,
                        pickerInitialValue: _end ?? _start,
                        mustBeAfter: _start,
                        focusNode: _endFocus,
                        label: l10n.businessEventEndLabel,
                        onSelected: (value) => _end = value,
                      ),
                    ),
                    if (_isAllDay) ...[
                      SizedBox(height: spacing.sp2),
                      Text(
                        l10n.businessEventAllDayHint,
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                    ],
                    SizedBox(height: spacing.sp5),
                    CraftskySingleSelectInput<String>(
                      key: const ValueKey('event-time-zone'),
                      label: l10n.businessEventTimeZoneLabel,
                      searchHintText: l10n.businessEventTimeZoneHint,
                      clearable: false,
                      value: _timeZone,
                      enabled: widget.enabled,
                      options: ref
                          .read(businessTimeZoneServiceProvider)
                          .names
                          .map(
                            (zone) =>
                                CraftskySelectOption(value: zone, label: zone),
                          )
                          .toList(),
                      onChanged: widget.enabled
                          ? (value) => setState(() {
                              if (value == null) return;
                              _timeZone = value;
                              _notifyChanged();
                            })
                          : null,
                    ),
                    SizedBox(height: spacing.sp5),
                    FormField<Set<String>>(
                      initialValue: _roles,
                      enabled: widget.enabled,
                      autovalidateMode: AutovalidateMode.onUserInteraction,
                      validator: (roles) => roles == null || roles.isEmpty
                          ? l10n.businessEventRolesRequired
                          : null,
                      builder: (field) =>
                          CraftskySearchableMultiSelectInput<String>(
                            keyPrefix: 'event-roles',
                            label: l10n.businessEventRolesLabel,
                            required: true,
                            searchHintText: l10n.businessEventRolesSearchHint,
                            enabled: widget.enabled,
                            errorText: field.errorText,
                            helperText: l10n.businessEventRolesHint,
                            maxSelected: 4,
                            maxSelectedErrorText: l10n
                                .projectComposerMultiSelectMaxSelectedError(4),
                            values: _roles.toList(),
                            options: [
                              for (final role in {
                                ...businessEventRoles,
                                ..._roles,
                              })
                                CraftskySelectOption(
                                  value: role,
                                  label: BusinessLabels.eventRole(
                                    BusinessOpenValue(
                                      value: role,
                                      known: businessEventRoles.contains(role),
                                    ),
                                    l10n,
                                  ),
                                ),
                            ],
                            onChanged: (values) => setState(() {
                              _roles = values.toSet();
                              field.didChange(_roles);
                              _notifyChanged();
                            }),
                          ),
                    ),
                    SizedBox(height: spacing.sp5),
                    _responsivePair(
                      wide,
                      _catalogField(
                        key: const ValueKey('event-mode'),
                        value: _mode,
                        label: l10n.businessEventModeLabel,
                        hint: l10n.businessEventModeHint,
                        values: businessEventModes,
                        itemLabel: (value) => BusinessLabels.eventMode(
                          BusinessOpenValue(
                            value: value,
                            known: businessEventModes.contains(value),
                          ),
                          l10n,
                        ),
                        onChanged: (value) => _mode = value,
                      ),
                      _catalogField(
                        key: const ValueKey('event-status'),
                        value: _status,
                        label: l10n.businessEventStatusLabel,
                        hint: l10n.businessEventStatusHint,
                        values: businessEventStatuses,
                        itemLabel: (value) => BusinessLabels.eventStatus(
                          BusinessOpenValue(
                            value: value,
                            known: businessEventStatuses.contains(value),
                          ),
                          l10n,
                        ),
                        onChanged: (value) => _status = value,
                      ),
                    ),
                    SizedBox(height: spacing.sp5),
                    CraftskyMultilineTextFormField(
                      textFieldKey: const ValueKey('event-summary'),
                      controller: _summary,
                      enabled: widget.enabled,
                      maxLength: businessEventSummaryLimit,
                      maxBytes: RecordTextLimits.eventSummary.bytes,
                      maxLines: 4,
                      label: l10n.businessEventSummaryLabel,
                      hintText: l10n.businessEventSummaryHint,
                    ),
                    SizedBox(height: spacing.sp5),
                    CraftskyTextFormField(
                      textFieldKey: const ValueKey('event-venue'),
                      controller: _venue,
                      enabled: widget.enabled,
                      maxLength: businessEventVenueLimit,
                      maxBytes: RecordTextLimits.eventVenue.bytes,
                      textCapitalization: TextCapitalization.words,
                      label: l10n.businessEventVenueLabel,
                      hintText: l10n.businessEventVenueHint,
                    ),
                    SizedBox(height: spacing.sp5),
                    CraftskyTextFormField(
                      textFieldKey: const ValueKey('event-uri'),
                      controller: _eventUri,
                      enabled: widget.enabled,
                      keyboardType: TextInputType.url,
                      label: l10n.businessEventUriLabel,
                      hintText: l10n.businessEventUriHint,
                    ),
                    SizedBox(height: spacing.sp5),
                    CraftskyTextFormField(
                      textFieldKey: const ValueKey('event-registration-uri'),
                      controller: _registrationUri,
                      enabled: widget.enabled,
                      keyboardType: TextInputType.url,
                      label: l10n.businessEventRegistrationUriLabel,
                      hintText: l10n.businessEventRegistrationUriHint,
                    ),
                    SizedBox(height: spacing.sp5),
                    ComposerImageAttachmentSection(
                      imagesState: ComposerImagesState(
                        images: [?_attachmentDraft],
                      ),
                      enabled: widget.enabled && !_preparingImage,
                      maxImages: 1,
                      useDetailsRoute: true,
                      keyPrefix: 'event',
                      imageUrlFor: (_) => _image.previewUrl,
                      onAddImages: () => _pickImage(ImageSource.gallery),
                      onTakePhoto: () => _pickImage(ImageSource.camera),
                      onAltTextChanged: (_, value) {
                        setState(() => _alt.text = value);
                        _notifyChanged();
                      },
                      onRemove: (_) => _removeImage(),
                      onReplace: (_, source) => _pickImage(source),
                      onReplaceUnavailable: (_, source) => _pickImage(source),
                      onEditImage: (_) => _editImage(),
                      onReorder: (_, _) {},
                      validationErrorText: _uploadError,
                    ),
                    if (_draftError != null)
                      Text(
                        _draftError!,
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.error,
                        ),
                      ),
                    SizedBox(
                      key: const Key('event-editor-bottom-safe-space'),
                      height:
                          spacing.sp9 + MediaQuery.paddingOf(context).bottom,
                    ),
                  ],
                ),
              ),
            ),
          );
        },
      ),
    );
  }

  Widget _boundaryField({
    required Key key,
    required DateTime? value,
    required FocusNode focusNode,
    required String label,
    required ValueChanged<DateTime> onSelected,
    DateTime? pickerInitialValue,
    DateTime? mustBeAfter,
  }) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final formatted = value != null && _isAllDay
        ? MaterialLocalizations.of(context).formatMediumDate(value)
        : _formatDateTime(context, value);
    final hint = _isAllDay
        ? l10n.businessEventDateHint
        : l10n.businessEventDateTimeHint;
    return FormField<DateTime>(
      key: ValueKey((key, _isAllDay)),
      initialValue: value,
      enabled: widget.enabled,
      validator: (selected) {
        if (selected == null) return l10n.businessEventTimeInvalid;
        if (mustBeAfter != null && !selected.isAfter(mustBeAfter)) {
          return l10n.businessEventEndAfterStart;
        }
        return null;
      },
      builder: (field) => CraftskyFieldScaffold(
        label: label,
        focusNode: focusNode,
        enabled: widget.enabled,
        required: true,
        errorText: field.errorText,
        semanticValue: formatted,
        semanticHint: hint,
        child: InkWell(
          key: key,
          focusNode: focusNode,
          canRequestFocus: widget.enabled,
          onTap: widget.enabled
              ? () async {
                  final selected = await _pickDateTime(
                    pickerInitialValue ?? value,
                  );
                  if (selected == null || !field.mounted) return;
                  setState(() => onSelected(selected));
                  field.didChange(selected);
                }
              : null,
          child: InputDecorator(
            isFocused: focusNode.hasFocus,
            decoration: InputDecoration(
              enabled: widget.enabled,
              contentPadding: EdgeInsets.symmetric(
                horizontal: spacing.sp3,
                vertical: spacing.sp3,
              ),
            ),
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    formatted ?? hint,
                    style: formatted == null
                        ? Theme.of(context).textTheme.bodyLarge?.copyWith(
                            color: Theme.of(
                              context,
                            ).colorScheme.onSurfaceVariant,
                          )
                        : Theme.of(context).textTheme.bodyLarge,
                  ),
                ),
                const Icon(CraftskyIconsBold.eventDatePicker),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Future<DateTime?> _pickDateTime(DateTime? current) async {
    final now = DateTime.now();
    final initial = current ?? DateTime(now.year, now.month, now.day, now.hour);
    final date = await showDatePicker(
      context: context,
      initialDate: initial,
      firstDate: DateTime(initial.year - 100),
      lastDate: DateTime(initial.year + 100),
    );
    if (date == null || !mounted) return null;
    if (_isAllDay) return DateTime.utc(date.year, date.month, date.day);
    final time = await showTimePicker(
      context: context,
      initialTime: TimeOfDay.fromDateTime(initial),
    );
    if (time == null || !mounted) return null;
    return DateTime.utc(
      date.year,
      date.month,
      date.day,
      time.hour,
      time.minute,
    );
  }

  static DateTime? _civilTime(DateTime? value) => value == null
      ? null
      : DateTime.utc(
          value.year,
          value.month,
          value.day,
          value.hour,
          value.minute,
          value.second,
        );

  Widget _catalogField({
    required Key key,
    required String value,
    required String label,
    required String hint,
    required Set<String> values,
    required String Function(String value) itemLabel,
    required ValueChanged<String> onChanged,
  }) {
    final items = {...values, if (!values.contains(value)) value};
    return CraftskySingleSelectInput<String>(
      key: key,
      label: label,
      searchHintText: hint,
      clearable: false,
      value: value,
      enabled: widget.enabled,
      options: alphabetizedSelectOptions(
        items.map(
          (item) => CraftskySelectOption(value: item, label: itemLabel(item)),
        ),
      ),
      onChanged: widget.enabled
          ? (next) => setState(() {
              if (next == null) return;
              onChanged(next);
              _notifyChanged();
            })
          : null,
    );
  }

  void _setAllDay(bool value) {
    setState(() {
      _isAllDay = value;
      if (value) {
        if (_start case final start?) {
          _start = DateTime.utc(start.year, start.month, start.day);
        }
        if (_end case final end?) {
          _end = DateTime.utc(end.year, end.month, end.day);
        }
        if (_start case final start?
            when _end == null || !_end!.isAfter(start)) {
          _end = DateTime.utc(start.year, start.month, start.day + 1);
        }
      }
    });
    _notifyChanged();
  }

  Widget _responsivePair(bool wide, Widget first, Widget second) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return wide
        ? Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(child: first),
              SizedBox(width: spacing.sp4),
              Expanded(child: second),
            ],
          )
        : Column(
            children: [
              first,
              SizedBox(height: spacing.sp5),
              second,
            ],
          );
  }

  BusinessEventDraft? _buildDraft() {
    final start = _start;
    final end = _end;
    if (start == null || end == null) return null;
    final image = switch (_image) {
      final ExistingBusinessImageDraft value => value.withAlt(_alt.text),
      final UploadedBusinessImageDraft value => value.withAlt(_alt.text),
      _ => _image,
    };
    return BusinessEventDraft(
      name: _name.text,
      startsAt: start,
      endsAt: end,
      roles: _roles.toList(),
      mode: _mode,
      status: _status,
      timeZone: _timeZone,
      isAllDay: _isAllDay,
      summary: _optional(_summary.text),
      venueName: _optional(_venue.text),
      eventUri: _optional(_eventUri.text),
      registrationUri: _optional(_registrationUri.text),
      image: image,
    );
  }

  ComposerImageDraft? get _attachmentDraft {
    final pending = _pendingImage;
    final bytes = _preview ?? _image.previewBytes;
    if (pending == null && !_image.hasImage && bytes == null) return null;

    final businessRatio = switch (_image) {
      ExistingBusinessImageDraft(:final aspectRatio) => aspectRatio,
      UploadedBusinessImageDraft(:final aspectRatio) => aspectRatio,
      _ => null,
    };
    final ratio = pending == null
        ? businessRatio == null
              ? null
              : CreatePostImageAspectRatio(
                  width: businessRatio.width,
                  height: businessRatio.height,
                )
        : CreatePostImageAspectRatio(
            width: pending.width,
            height: pending.height,
          );
    final phase = pending != null
        ? ImageReady(
            bytes: pending.bytes,
            mimeType: pending.mimeType,
            width: pending.width,
            height: pending.height,
            sha256: '',
          )
        : _preparingImage
        ? const ImagePreparing()
        : ImageUploaded(
            UploadedDraftImage(
              cid: switch (_image) {
                ExistingBusinessImageDraft(:final cid) => cid,
                UploadedBusinessImageDraft(:final cid) => cid,
                _ => '',
              },
              mime: switch (_image) {
                ExistingBusinessImageDraft(:final mime) => mime,
                UploadedBusinessImageDraft(:final mime) => mime,
                _ => 'image/jpeg',
              },
              size: switch (_image) {
                ExistingBusinessImageDraft(:final size) => size,
                UploadedBusinessImageDraft(:final size) => size,
                _ => bytes?.length ?? 0,
              },
              aspectRatio: ratio,
            ),
          );
    return ComposerImageDraft(
      id: 'image',
      fileName: 'event-image',
      mimeType: pending?.mimeType ?? 'image/jpeg',
      altText: _alt.text,
      phase: phase,
      previewBytes: bytes,
      previewAspectRatio: ratio,
    );
  }

  Future<void> _pickImage(ImageSource source) async {
    final ownership = _captureOwnership();
    final reporter = ref.read(errorReporterProvider);
    final previousPending = _pendingImage;
    final previousPreview = _preview;
    final picker =
        widget.pickImage ??
        (source, onPreviewReady) => ref
            .read(profileImagePickerProvider)
            .pickAndPrepare(source: source, onPreviewReady: onPreviewReady);
    setState(() {
      _preparingImage = true;
      _uploadError = null;
    });
    try {
      final result = await picker(source, (bytes) {
        if (_isCurrent(ownership)) setState(() => _preview = bytes);
      });
      if (!_isCurrent(ownership)) return;
      if (result != null) {
        setState(() {
          _pendingImage = result;
          _preview = result.bytes;
          _notifyChanged();
        });
      }
    } on Object catch (error, stackTrace) {
      unawaited(
        reportImagePickFailure(
          reporter: reporter,
          error: error,
          stackTrace: stackTrace,
          operation: ImagePickOperation.businessEvent,
          accountDid: ownership?.session.account.did.value,
        ),
      );
      if (!_isCurrent(ownership)) return;
      setState(() {
        _pendingImage = previousPending;
        _preview = previousPreview;
        _uploadError = imagePickFailureMessage(
          error,
          AppLocalizations.of(context),
          fallback: AppLocalizations.of(context).businessEventUploadError,
        );
      });
    } finally {
      if (mounted) {
        setState(() {
          _preparingImage = false;
          if (!_isCurrent(ownership)) _preview = null;
        });
      }
    }
  }

  Future<void> _editImage() async {
    final image = _pendingImage;
    if (image == null || _preparingImage || !widget.enabled) return;
    final ownership = _captureOwnership();
    final reporter = ref.read(errorReporterProvider);
    final media = ref.read(profileImageMediaServiceProvider);
    setState(() {
      _preparingImage = true;
      _uploadError = null;
    });
    bool current() =>
        mounted && _isCurrent(ownership) && identical(_pendingImage, image);
    try {
      final request = BasicImageEditRequest(
        bytes: image.bytes,
        mimeType: image.mimeType,
        mode: BasicImageEditMode.composer,
      );
      final output =
          await (widget.editImage?.call(request) ??
              showBasicImageEditor(context, request, ownership: ownership));
      if (output == null || !current()) return;
      final prepared = await media
          .prepareImage(
            bytes: output.bytes,
            fileName: output.mimeType == 'image/png'
                ? 'edited.png'
                : 'edited.jpg',
            mimeType: output.mimeType,
          )
          .future;
      if (!current()) return;
      final check = media.validatePreparedUploadBytes(
        originalBytes: output.bytes.length,
        preparedBytes: prepared.bytes.length,
        width: prepared.width,
        height: prepared.height,
      );
      if (!check.canUpload) {
        throw const ProfileImagePickException(expectedRejection: true);
      }
      setState(() {
        _pendingImage = PreparedProfileImage(
          bytes: prepared.bytes,
          mimeType: prepared.mimeType,
          width: prepared.width,
          height: prepared.height,
        );
        _preview = prepared.bytes;
        _notifyChanged();
      });
    } on Object catch (error, stackTrace) {
      unawaited(
        reportImagePickFailure(
          reporter: reporter,
          error: error,
          stackTrace: stackTrace,
          operation: ImagePickOperation.businessEvent,
          accountDid: ownership?.session.account.did.value,
        ),
      );
      if (current()) {
        setState(
          () => _uploadError = AppLocalizations.of(context).imageEditFailed,
        );
      }
    } finally {
      if (mounted) setState(() => _preparingImage = false);
    }
  }

  void _removeImage() {
    setState(() {
      _pendingImage = null;
      _image = const RemovedBusinessImageDraft();
      _preview = null;
      _notifyChanged();
    });
  }

  bool _isCurrent(ActiveAccountLease? ownership) =>
      ActiveAccountOperation.fromWidget(ref, ownership).isCurrent;

  ActiveAccountLease? _captureOwnership() {
    return ref.read(sessionRegistryProvider).value?.activeLease;
  }

  void _notifyChanged() {
    widget.onChanged();
  }
}

String? _formatDateTime(BuildContext context, DateTime? value) {
  if (value == null) return null;
  final material = MaterialLocalizations.of(context);
  final date = material.formatMediumDate(value);
  final time = material.formatTimeOfDay(
    TimeOfDay.fromDateTime(value),
    alwaysUse24HourFormat: MediaQuery.alwaysUse24HourFormatOf(context),
  );
  return '$date, $time';
}

String? _optional(String value) => value.trim().isEmpty ? null : value.trim();
