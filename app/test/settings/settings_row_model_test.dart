import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('canonical Settings sections and rows remain in the agreed order', () {
    expect(
      settingsSections.map((section) => section.id),
      [
        SettingsSectionId.preferences,
        SettingsSectionId.connections,
        SettingsSectionId.discovery,
        SettingsSectionId.general,
      ],
    );
    expect(settingsSections[0].rows.map((row) => row.id), [
      SettingsRowId.appearance,
      SettingsRowId.customisation,
      SettingsRowId.languages,
      SettingsRowId.notifications,
    ]);
    expect(settingsSections[1].rows.map((row) => row.id), [
      SettingsRowId.growth,
      SettingsRowId.followers,
      SettingsRowId.following,
      SettingsRowId.mutedAccounts,
      SettingsRowId.blockedAccounts,
    ]);
    expect(settingsSections[2].rows.map((row) => row.id), [
      SettingsRowId.findPeopleFromInstagram,
    ]);
    expect(settingsSections[3].rows.map((row) => row.id), [
      SettingsRowId.account,
      SettingsRowId.about,
    ]);
  });

  test('row kinds expose only their truthful trailing icon', () {
    const disclosure = SettingsRowDescriptor(
      id: SettingsRowId.account,
      kind: SettingsRowKind.disclosure,
    );
    const external = SettingsRowDescriptor(
      id: SettingsRowId.terms,
      kind: SettingsRowKind.externalLink,
    );

    expect(disclosure.trailingIcon(TextDirection.ltr), Icons.chevron_right);
    expect(disclosure.trailingIcon(TextDirection.rtl), Icons.chevron_left);
    expect(external.trailingIcon(TextDirection.ltr), Icons.open_in_new);
    expect(external.trailingIcon(TextDirection.rtl), Icons.open_in_new);

    for (final kind in [
      SettingsRowKind.action,
      SettingsRowKind.destructiveAction,
      SettingsRowKind.readOnly,
    ]) {
      final row = SettingsRowDescriptor(id: SettingsRowId.version, kind: kind);
      expect(row.trailingIcon(TextDirection.ltr), isNull);
      expect(row.trailingIcon(TextDirection.rtl), isNull);
    }
  });
}
