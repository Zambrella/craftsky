import 'package:craftsky_app/languages/data/language_preferences_api_client.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http_mock_adapter/http_mock_adapter.dart';

void main() {
  const preferences = LanguagePreferences(
    primaryLanguage: 'fr',
    contentLanguages: ['fr', 'en'],
  );

  test('uses the fixed private preference routes without selectors', () async {
    final dio = Dio(BaseOptions(baseUrl: 'https://appview.example.com'));
    DioAdapter(dio: dio)
      ..onGet(
        '/v1/languages/preferences',
        (server) => server.reply(200, preferences.toJson()),
      )
      ..onPut(
        '/v1/languages/preferences',
        (server) => server.reply(200, preferences.toJson()),
        data: preferences.toJson(),
      )
      ..onPost(
        '/v1/languages/preferences/initialize',
        (server) => server.reply(200, preferences.toJson()),
        data: preferences.toJson(),
      );
    final client = LanguagePreferencesApiClient(dio);

    expect(await client.get(), preferences);
    expect(await client.replace(preferences), preferences);
    expect(await client.initialize(preferences), preferences);
  });
}
