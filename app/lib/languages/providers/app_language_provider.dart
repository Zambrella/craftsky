import 'package:craftsky_app/app_dependencies.dart';
import 'package:flutter/widgets.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'app_language_provider.g.dart';

@Riverpod(keepAlive: true)
class AppLanguage extends _$AppLanguage {
  static const _key = 'app_language';

  @override
  Locale build() {
    final stored = ref.watch(sharedPreferencesProvider).getString(_key);
    return Locale(stored == 'en' ? stored! : 'en');
  }

  Future<void> select(Locale locale) async {
    if (locale.languageCode != 'en') {
      throw ArgumentError.value(locale, 'locale', 'Unsupported App language');
    }
    await ref.read(sharedPreferencesProvider).setString(_key, 'en');
    if (ref.mounted) state = locale;
  }
}
