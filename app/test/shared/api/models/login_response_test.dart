import 'dart:convert';

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/shared/api/models/login_response.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test('parses snake_case JSON from the server', () {
    const json = '{"auth_url":"https://pds.example.com/authorize?request_uri=x"}';
    final parsed = LoginResponseMapper.fromJson(json);
    expect(parsed.authUrl, 'https://pds.example.com/authorize?request_uri=x');
  });

  test('serialises back to snake_case JSON', () {
    const original = LoginResponse(authUrl: 'https://pds.example.com/a');
    final roundTrip = jsonDecode(original.toJson()) as Map<String, dynamic>;
    expect(roundTrip.keys.single, 'auth_url');
    expect(roundTrip['auth_url'], 'https://pds.example.com/a');
  });
}
