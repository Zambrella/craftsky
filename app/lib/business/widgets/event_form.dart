import 'dart:typed_data';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_labels.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/services/business_time_zone_service.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

typedef EventImagePicker =
    Future<ProfileImagePickResult?> Function(
      void Function(Uint8List bytes) onPreviewReady,
    );

class EventForm extends ConsumerStatefulWidget {
  const EventForm({
    required this.onChanged,
    this.initial,
    this.pickImage,
    this.enabled = true,
    super.key,
  });

  final BusinessEventDraft? initial;
  final ValueChanged<BusinessEventDraft> onChanged;
  final EventImagePicker? pickImage;
  final bool enabled;

  @override
  ConsumerState<EventForm> createState() => EventFormState();
}

class EventFormState extends ConsumerState<EventForm> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _name;
  late final TextEditingController _start;
  late final TextEditingController _end;
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
  late BusinessImageDraft _image;
  Uint8List? _preview;
  String? _draftError;
  String? _uploadError;
  bool _uploading = false;

  @override
  void initState() {
    super.initState();
    final initial = widget.initial;
    _name = TextEditingController(text: initial?.name);
    _start = TextEditingController(text: _formatLocal(initial?.startsAt));
    _end = TextEditingController(text: _formatLocal(initial?.endsAt));
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
      _start,
      _end,
      _summary,
      _venue,
      _eventUri,
      _registrationUri,
      _alt,
    ]) {
      controller.dispose();
    }
    super.dispose();
  }

  BusinessEventDraft? submit() {
    final formValid = _formKey.currentState?.validate() ?? false;
    final draft = _buildDraft();
    final errors = draft?.validate(ref.read(businessTimeZoneServiceProvider));
    setState(() {
      _draftError = draft == null || (errors?.isNotEmpty ?? true)
          ? AppLocalizations.of(context).businessEventValidationError
          : null;
    });
    if (!formValid || draft == null || errors!.isNotEmpty || _uploading) {
      return null;
    }
    return draft;
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Form(
      key: _formKey,
      onChanged: _notifyChanged,
      child: LayoutBuilder(
        builder: (context, constraints) {
          final wide = constraints.maxWidth >= 720;
          return SingleChildScrollView(
            padding: const EdgeInsets.all(20),
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 880),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    TextFormField(
                      key: const ValueKey('event-name'),
                      controller: _name,
                      enabled: widget.enabled,
                      maxLength: businessEventNameLimit,
                      decoration: InputDecoration(
                        labelText: l10n.businessEventNameLabel,
                      ),
                      validator: (value) =>
                          value == null || value.trim().isEmpty
                          ? l10n.businessEventNameRequired
                          : null,
                    ),
                    const SizedBox(height: 8),
                    _responsivePair(
                      wide,
                      _boundaryField(
                        key: const ValueKey('event-start'),
                        controller: _start,
                        label: l10n.businessEventStartLabel,
                      ),
                      _boundaryField(
                        key: const ValueKey('event-end'),
                        controller: _end,
                        label: l10n.businessEventEndLabel,
                      ),
                    ),
                    SwitchListTile(
                      contentPadding: EdgeInsets.zero,
                      title: Text(l10n.businessEventAllDay),
                      value: _isAllDay,
                      onChanged: widget.enabled
                          ? (value) => setState(() {
                              _isAllDay = value;
                              _notifyChanged();
                            })
                          : null,
                    ),
                    DropdownButtonFormField<String>(
                      key: const ValueKey('event-time-zone'),
                      isExpanded: true,
                      initialValue: _timeZone,
                      decoration: InputDecoration(
                        labelText: l10n.businessEventTimeZoneLabel,
                      ),
                      items: ref
                          .read(businessTimeZoneServiceProvider)
                          .names
                          .map(
                            (zone) => DropdownMenuItem(
                              value: zone,
                              child: Text(
                                zone,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                            ),
                          )
                          .toList(),
                      onChanged: widget.enabled
                          ? (value) => setState(() {
                              _timeZone = value!;
                              _notifyChanged();
                            })
                          : null,
                    ),
                    const SizedBox(height: 18),
                    Text(
                      l10n.businessEventRolesLabel,
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    Wrap(
                      spacing: 8,
                      children: [
                        for (final role in {
                          ...businessEventRoles,
                          ..._roles.where(
                            (role) => !businessEventRoles.contains(role),
                          ),
                        })
                          FilterChip(
                            label: Text(
                              BusinessLabels.eventRole(
                                BusinessOpenValue(
                                  value: role,
                                  known: businessEventRoles.contains(role),
                                ),
                                l10n,
                              ),
                            ),
                            selected: _roles.contains(role),
                            onSelected: widget.enabled
                                ? (selected) => setState(() {
                                    selected
                                        ? _roles.add(role)
                                        : _roles.remove(role);
                                    _notifyChanged();
                                  })
                                : null,
                          ),
                      ],
                    ),
                    if (_roles.isEmpty)
                      Text(
                        l10n.businessEventRolesRequired,
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.error,
                        ),
                      ),
                    const SizedBox(height: 12),
                    _responsivePair(
                      wide,
                      _catalogField(
                        key: const ValueKey('event-mode'),
                        value: _mode,
                        label: l10n.businessEventModeLabel,
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
                    TextFormField(
                      key: const ValueKey('event-summary'),
                      controller: _summary,
                      enabled: widget.enabled,
                      maxLength: businessEventSummaryLimit,
                      maxLines: 4,
                      decoration: InputDecoration(
                        labelText: l10n.businessEventSummaryLabel,
                      ),
                    ),
                    TextFormField(
                      key: const ValueKey('event-venue'),
                      controller: _venue,
                      enabled: widget.enabled,
                      maxLength: businessEventVenueLimit,
                      decoration: InputDecoration(
                        labelText: l10n.businessEventVenueLabel,
                      ),
                    ),
                    TextFormField(
                      key: const ValueKey('event-uri'),
                      controller: _eventUri,
                      enabled: widget.enabled,
                      keyboardType: TextInputType.url,
                      decoration: InputDecoration(
                        labelText: l10n.businessEventUriLabel,
                      ),
                    ),
                    TextFormField(
                      key: const ValueKey('event-registration-uri'),
                      controller: _registrationUri,
                      enabled: widget.enabled,
                      keyboardType: TextInputType.url,
                      decoration: InputDecoration(
                        labelText: l10n.businessEventRegistrationUriLabel,
                      ),
                    ),
                    if (_preview != null)
                      SizedBox(
                        height: 180,
                        child: Image.memory(_preview!, fit: BoxFit.cover),
                      )
                    else if (_image.hasImage)
                      const SizedBox(
                        height: 96,
                        child: Icon(Icons.image_outlined, size: 56),
                      ),
                    if (_uploading)
                      LinearProgressIndicator(
                        semanticsLabel: l10n.businessImageUploading,
                      ),
                    Wrap(
                      spacing: 8,
                      children: [
                        TextButton.icon(
                          onPressed: widget.enabled && !_uploading
                              ? _pickImage
                              : null,
                          icon: const Icon(Icons.add_photo_alternate_outlined),
                          label: Text(
                            _image.hasImage
                                ? l10n.businessEventReplaceImage
                                : l10n.businessEventAddImage,
                          ),
                        ),
                        if (_image.hasImage)
                          TextButton.icon(
                            onPressed: widget.enabled && !_uploading
                                ? () => setState(() {
                                    _image = const RemovedBusinessImageDraft();
                                    _preview = null;
                                    _notifyChanged();
                                  })
                                : null,
                            icon: const Icon(Icons.delete_outline),
                            label: Text(l10n.businessEventRemoveImage),
                          ),
                      ],
                    ),
                    TextField(
                      key: const ValueKey('event-image-alt'),
                      controller: _alt,
                      enabled: widget.enabled && _image.hasImage,
                      maxLength: businessImageAltLimit,
                      decoration: InputDecoration(
                        labelText: l10n.businessEventImageDescriptionLabel,
                      ),
                      onChanged: (_) => _notifyChanged(),
                    ),
                    if (_uploadError != null)
                      Text(
                        _uploadError!,
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.error,
                        ),
                      ),
                    if (_draftError != null)
                      Text(
                        _draftError!,
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.error,
                        ),
                      ),
                    const SizedBox(height: 32),
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
    required TextEditingController controller,
    required String label,
  }) {
    final l10n = AppLocalizations.of(context);
    return TextFormField(
      key: key,
      controller: controller,
      enabled: widget.enabled,
      decoration: InputDecoration(
        labelText: label,
        hintText: l10n.businessEventTimeHint,
      ),
      validator: (value) => _parseLocal(value ?? '') == null
          ? l10n.businessEventTimeInvalid
          : null,
    );
  }

  Widget _catalogField({
    required Key key,
    required String value,
    required String label,
    required Set<String> values,
    required String Function(String value) itemLabel,
    required ValueChanged<String> onChanged,
  }) {
    final items = {...values, if (!values.contains(value)) value};
    return DropdownButtonFormField<String>(
      key: key,
      isExpanded: true,
      initialValue: value,
      decoration: InputDecoration(labelText: label),
      items: items
          .map(
            (item) => DropdownMenuItem(
              value: item,
              child: Text(itemLabel(item)),
            ),
          )
          .toList(),
      onChanged: widget.enabled
          ? (next) => setState(() {
              onChanged(next!);
              _notifyChanged();
            })
          : null,
    );
  }

  Widget _responsivePair(bool wide, Widget first, Widget second) => wide
      ? Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(child: first),
            const SizedBox(width: 16),
            Expanded(child: second),
          ],
        )
      : Column(
          children: [first, const SizedBox(height: 12), second],
        );

  BusinessEventDraft? _buildDraft() {
    final start = _parseLocal(_start.text);
    final end = _parseLocal(_end.text);
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

  Future<void> _pickImage() async {
    final ownership = _captureOwnership();
    final picker =
        widget.pickImage ??
        (onPreviewReady) => ref
            .read(profileImagePickerProvider)
            .pickAndUpload(onPreviewReady: onPreviewReady);
    setState(() {
      _uploading = true;
      _uploadError = null;
      _preview = null;
    });
    try {
      final result = await picker((bytes) {
        if (_isCurrent(ownership)) setState(() => _preview = bytes);
      });
      if (!_isCurrent(ownership)) return;
      if (result != null) {
        setState(() {
          _image = UploadedBusinessImageDraft.fromUpload(
            result.uploaded,
            alt: _alt.text,
            previewBytes: result.previewBytes,
          );
          _preview = result.previewBytes;
          _notifyChanged();
        });
      }
    } on Object {
      if (!_isCurrent(ownership)) return;
      setState(() {
        _preview = null;
        _uploadError = AppLocalizations.of(context).businessEventUploadError;
      });
    } finally {
      if (mounted) {
        setState(() {
          _uploading = false;
          if (!_isCurrent(ownership)) _preview = null;
        });
      }
    }
  }

  bool _isCurrent(ActiveAccountLease? ownership) {
    if (!mounted) return false;
    if (ownership == null) return true;
    return ref.read(sessionRegistryProvider).value?.isCurrent(ownership) ??
        false;
  }

  ActiveAccountLease? _captureOwnership() {
    return ref.read(sessionRegistryProvider).value?.activeLease;
  }

  void _notifyChanged() {
    final draft = _buildDraft();
    if (draft != null) widget.onChanged(draft);
  }
}

DateTime? _parseLocal(String value) => DateTime.tryParse(value.trim());

String _formatLocal(DateTime? value) {
  if (value == null) return '';
  String two(int part) => part.toString().padLeft(2, '0');
  return '${value.year.toString().padLeft(4, '0')}-${two(value.month)}-'
      '${two(value.day)} ${two(value.hour)}:${two(value.minute)}';
}

String? _optional(String value) => value.trim().isEmpty ? null : value.trim();
