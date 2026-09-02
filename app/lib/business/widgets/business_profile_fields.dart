import 'dart:convert';

import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_labels.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_form_builder_select_fields.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:sealed_countries/sealed_countries.dart';

abstract final class BusinessProfileFieldNames {
  static const types = 'businessTypes';
  static const offerings = 'businessOfferings';
  static const tagline = 'businessTagline';
  static const hours = 'businessHours';
  static const serviceArea = 'businessServiceArea';
  static const country = 'businessCountry';
  static const locality = 'businessLocality';
  static const actionType = 'businessActionType';
  static const actionDestination = 'businessActionDestination';
}

class BusinessProfileFields extends StatelessWidget {
  const BusinessProfileFields({
    required this.initial,
    required this.enabled,
    super.key,
  });

  final BusinessDeclarationDraft initial;
  final bool enabled;

  static BusinessDeclarationDraft draftFrom(
    Map<String, dynamic> values,
    BusinessDeclarationDraft baseline,
  ) {
    final selectedTypes =
        (values[BusinessProfileFieldNames.types] as List<String>? ?? const [])
            .toSet();
    final selectedOfferings =
        (values[BusinessProfileFieldNames.offerings] as List<String>? ??
                const [])
            .toSet();
    final country = _trimmedOrNull(
      values[BusinessProfileFieldNames.country] as String?,
    )?.toUpperCase();
    final locality = _trimmedOrNull(
      values[BusinessProfileFieldNames.locality] as String?,
    );
    final actionType = _trimmedOrNull(
      values[BusinessProfileFieldNames.actionType] as String?,
    );
    final destination = _trimmedOrNull(
      values[BusinessProfileFieldNames.actionDestination] as String?,
    );

    return BusinessDeclarationDraft(
      expectedCid: baseline.expectedCid,
      businessTypes: [
        for (final value in BusinessLabels.businessTypes)
          if (selectedTypes.contains(value.value)) value,
        ...baseline.businessTypes.where((value) => !value.known),
      ],
      offerings: [
        for (final value in BusinessLabels.offerings)
          if (selectedOfferings.contains(value.value)) value,
        ...baseline.offerings.where((value) => !value.known),
      ],
      tagline: _trimmedOrNull(
        values[BusinessProfileFieldNames.tagline] as String?,
      ),
      hoursNote: _trimmedOrNull(
        values[BusinessProfileFieldNames.hours] as String?,
      ),
      serviceArea: _trimmedOrNull(
        values[BusinessProfileFieldNames.serviceArea] as String?,
      ),
      location: country == null
          ? null
          : BusinessLocation(country: country, locality: locality),
      primaryAction: actionType == null || destination == null
          ? null
          : BusinessAction(type: actionType, destination: destination),
      products: baseline.products,
    );
  }

  static String? _trimmedOrNull(String? value) {
    final trimmed = value?.trim();
    return trimmed == null || trimmed.isEmpty ? null : trimmed;
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final knownTypes = initial.businessTypes
        .where((value) => value.known)
        .map((value) => value.value)
        .toList();
    final knownOfferings = initial.offerings
        .where((value) => value.known)
        .map((value) => value.value)
        .toList();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Divider(height: spacing.sp8),
        Text(
          l10n.editProfileBusinessHeading,
          style: Theme.of(
            context,
          ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w800),
        ),
        SizedBox(height: spacing.sp1),
        Text(l10n.editProfileBusinessHelper),
        SizedBox(height: spacing.sp5),
        CraftskyFormBuilderMultiSelectField<String>(
          name: BusinessProfileFieldNames.types,
          label: l10n.editProfileBusinessTypesLabel,
          helperText: l10n.editProfileBusinessTypesHelper,
          initialValue: knownTypes,
          enabled: enabled,
          maxSelected: 5,
          maxSelectedErrorText: l10n.editProfileBusinessTypesLimit,
          options: [
            for (final value in BusinessLabels.businessTypes)
              CraftskySelectOption(
                value: value.value,
                label: BusinessLabels.openValue(value, l10n),
              ),
          ],
        ),
        SizedBox(height: spacing.sp5),
        CraftskyFormBuilderMultiSelectField<String>(
          name: BusinessProfileFieldNames.offerings,
          label: l10n.editProfileBusinessOfferingsLabel,
          helperText: l10n.editProfileBusinessOfferingsHelper,
          initialValue: knownOfferings,
          enabled: enabled,
          maxSelected: 10,
          maxSelectedErrorText: l10n.editProfileBusinessOfferingsLimit,
          options: [
            for (final value in BusinessLabels.offerings)
              CraftskySelectOption(
                value: value.value,
                label: BusinessLabels.openValue(value, l10n),
              ),
          ],
        ),
        SizedBox(height: spacing.sp5),
        _TextField(
          name: BusinessProfileFieldNames.tagline,
          label: l10n.editProfileBusinessTaglineLabel,
          initialValue: initial.tagline,
          maxLines: 2,
          enabled: enabled,
          validator: _bounded(
            l10n.editProfileBusinessTaglineTooLong,
            graphemes: 100,
            bytes: 1000,
          ),
        ),
        SizedBox(height: spacing.sp5),
        _TextField(
          name: BusinessProfileFieldNames.hours,
          label: l10n.editProfileBusinessHoursLabel,
          initialValue: initial.hoursNote,
          maxLines: 4,
          enabled: enabled,
          validator: _bounded(
            l10n.editProfileBusinessHoursTooLong,
            graphemes: 300,
            bytes: 3000,
          ),
        ),
        SizedBox(height: spacing.sp5),
        _TextField(
          name: BusinessProfileFieldNames.serviceArea,
          label: l10n.editProfileBusinessServiceAreaLabel,
          initialValue: initial.serviceArea,
          maxLines: 3,
          enabled: enabled,
          validator: _bounded(
            l10n.editProfileBusinessServiceAreaTooLong,
            graphemes: 200,
            bytes: 2000,
          ),
        ),
        SizedBox(height: spacing.sp5),
        _TextField(
          name: BusinessProfileFieldNames.country,
          label: l10n.editProfileBusinessCountryLabel,
          initialValue: initial.location?.country,
          textCapitalization: TextCapitalization.characters,
          enabled: enabled,
          validator: (value) {
            final country = value?.trim();
            if (country == null || country.isEmpty) return null;
            return WorldCountry.maybeFromCodeShort(country.toUpperCase()) ==
                    null
                ? l10n.editProfileBusinessCountryInvalid
                : null;
          },
        ),
        SizedBox(height: spacing.sp5),
        _TextField(
          name: BusinessProfileFieldNames.locality,
          label: l10n.editProfileBusinessLocalityLabel,
          initialValue: initial.location?.locality,
          enabled: enabled,
          validator: _bounded(
            l10n.editProfileBusinessLocalityTooLong,
            graphemes: 100,
            bytes: 1000,
          ),
        ),
        SizedBox(height: spacing.sp5),
        CraftskyFormBuilderDropdownField<String>(
          name: BusinessProfileFieldNames.actionType,
          label: l10n.editProfileBusinessActionLabel,
          initialValue: initial.primaryAction?.type,
          enabled: enabled,
          options: [
            CraftskySelectOption(
              value: '',
              label: l10n.editProfileBusinessActionNone,
            ),
            for (final action in BusinessLabels.actions)
              CraftskySelectOption(
                value: action,
                label: BusinessLabels.action(action, l10n),
              ),
          ],
        ),
        SizedBox(height: spacing.sp5),
        _TextField(
          name: BusinessProfileFieldNames.actionDestination,
          label: l10n.editProfileBusinessActionDestinationLabel,
          initialValue: initial.primaryAction?.destination,
          keyboardType: TextInputType.url,
          enabled: enabled,
          validator: (value) {
            final destination = value?.trim();
            final actionType =
                FormBuilder.of(
                      context,
                    )?.instantValue[BusinessProfileFieldNames.actionType]
                    as String?;
            if (actionType == null || actionType.isEmpty) {
              return destination == null || destination.isEmpty
                  ? null
                  : l10n.editProfileBusinessActionDestinationInvalid;
            }
            if (destination == null || destination.isEmpty) {
              return l10n.editProfileBusinessActionDestinationInvalid;
            }
            final valid = actionType == 'email'
                ? _validMailto(destination)
                : _validHttps(destination);
            return valid
                ? null
                : l10n.editProfileBusinessActionDestinationInvalid;
          },
        ),
      ],
    );
  }

  FormFieldValidator<String> _bounded(
    String message, {
    required int graphemes,
    required int bytes,
  }) => (value) {
    if (value == null || value.isEmpty) return null;
    return value.characters.length > graphemes ||
            utf8.encode(value).length > bytes
        ? message
        : null;
  };

  static bool _validHttps(String value) {
    if (utf8.encode(value).length > 2048) return false;
    final uri = Uri.tryParse(value);
    return uri != null &&
        uri.scheme == 'https' &&
        uri.host.isNotEmpty &&
        uri.userInfo.isEmpty;
  }

  static bool _validMailto(String value) {
    if (!value.startsWith('mailto:') ||
        value.contains(RegExp(r'[\s\x00-\x1F\x7F%,;]'))) {
      return false;
    }
    final uri = Uri.tryParse(value);
    if (uri == null || uri.hasQuery || uri.hasFragment) return false;
    final address = value.substring('mailto:'.length);
    if (address.codeUnits.any((unit) => unit > 0x7f)) return false;
    if (ascii.encode(address).length > 320) return false;
    final at = address.indexOf('@');
    if (at <= 0 || at != address.lastIndexOf('@')) return false;
    final local = address.substring(0, at);
    final domain = address.substring(at + 1);
    if (!RegExp(r"^[A-Za-z0-9.!#$&'*+/=?^_`{|}~-]+$").hasMatch(local)) {
      return false;
    }
    return RegExp(
      r'^(?=.{1,253}$)(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}'
      r'[A-Za-z0-9])?\.)+[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}'
      r'[A-Za-z0-9])?$',
    ).hasMatch(domain);
  }
}

class _TextField extends StatelessWidget {
  const _TextField({
    required this.name,
    required this.label,
    required this.enabled,
    this.initialValue,
    this.validator,
    this.maxLines = 1,
    this.keyboardType,
    this.textCapitalization = TextCapitalization.none,
  });

  final String name;
  final String label;
  final String? initialValue;
  final bool enabled;
  final FormFieldValidator<String>? validator;
  final int maxLines;
  final TextInputType? keyboardType;
  final TextCapitalization textCapitalization;

  @override
  Widget build(BuildContext context) => CraftskyFormTextField(
    name: name,
    label: label,
    initialValue: initialValue ?? '',
    enabled: enabled,
    validator: validator,
    maxLines: maxLines,
    keyboardType: keyboardType,
    textCapitalization: textCapitalization,
  );
}
