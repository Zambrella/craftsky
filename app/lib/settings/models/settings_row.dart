import 'package:flutter/material.dart';

enum SettingsSectionId { preferences, connections, discovery, general }

enum SettingsRowId {
  switchAccount,
  customisation,
  languages,
  notifications,
  growth,
  followers,
  following,
  mutedAccounts,
  blockedAccounts,
  findPeopleFromInstagram,
  account,
  about,
  terms,
  privacyPolicy,
  clearImageCache,
  version,
  deleteAccount,
  signOut,
}

enum SettingsRowKind {
  disclosure,
  externalLink,
  action,
  destructiveAction,
  readOnly,
}

final class SettingsRowDescriptor {
  const SettingsRowDescriptor({required this.id, required this.kind});

  final SettingsRowId id;
  final SettingsRowKind kind;

  IconData? trailingIcon(TextDirection direction) => switch (kind) {
    SettingsRowKind.disclosure =>
      direction == TextDirection.ltr ? Icons.chevron_right : Icons.chevron_left,
    SettingsRowKind.externalLink => Icons.open_in_new,
    SettingsRowKind.action ||
    SettingsRowKind.destructiveAction ||
    SettingsRowKind.readOnly => null,
  };
}

final class SettingsSectionDescriptor {
  const SettingsSectionDescriptor({required this.id, required this.rows});

  final SettingsSectionId id;
  final List<SettingsRowDescriptor> rows;
}

const settingsSections = <SettingsSectionDescriptor>[
  SettingsSectionDescriptor(
    id: SettingsSectionId.preferences,
    rows: [
      SettingsRowDescriptor(
        id: SettingsRowId.customisation,
        kind: SettingsRowKind.disclosure,
      ),
      SettingsRowDescriptor(
        id: SettingsRowId.languages,
        kind: SettingsRowKind.disclosure,
      ),
      SettingsRowDescriptor(
        id: SettingsRowId.notifications,
        kind: SettingsRowKind.disclosure,
      ),
    ],
  ),
  SettingsSectionDescriptor(
    id: SettingsSectionId.connections,
    rows: [
      SettingsRowDescriptor(
        id: SettingsRowId.growth,
        kind: SettingsRowKind.disclosure,
      ),
      SettingsRowDescriptor(
        id: SettingsRowId.followers,
        kind: SettingsRowKind.disclosure,
      ),
      SettingsRowDescriptor(
        id: SettingsRowId.following,
        kind: SettingsRowKind.disclosure,
      ),
      SettingsRowDescriptor(
        id: SettingsRowId.mutedAccounts,
        kind: SettingsRowKind.disclosure,
      ),
      SettingsRowDescriptor(
        id: SettingsRowId.blockedAccounts,
        kind: SettingsRowKind.disclosure,
      ),
    ],
  ),
  SettingsSectionDescriptor(
    id: SettingsSectionId.discovery,
    rows: [
      SettingsRowDescriptor(
        id: SettingsRowId.findPeopleFromInstagram,
        kind: SettingsRowKind.disclosure,
      ),
    ],
  ),
  SettingsSectionDescriptor(
    id: SettingsSectionId.general,
    rows: [
      SettingsRowDescriptor(
        id: SettingsRowId.account,
        kind: SettingsRowKind.disclosure,
      ),
      SettingsRowDescriptor(
        id: SettingsRowId.about,
        kind: SettingsRowKind.disclosure,
      ),
    ],
  ),
];
