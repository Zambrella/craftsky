import 'package:craftsky_app/router/router.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('login completion route carries only the short-lived browser code', () {
    final location = const AuthCompleteRoute(code: 'short-lived code').location;
    final uri = Uri.parse(location);

    expect(uri.path, '/auth/complete');
    expect(uri.queryParameters, {'code': 'short-lived code'});
    expect(location, isNot(contains('token')));
  });

  test('account-deletion reauth route never carries an auth bearer', () {
    final location = const AccountDeletionReauthCompleteRoute(
      jobId: 'job-123',
      proof: 'single-use-proof',
    ).location;
    final uri = Uri.parse(location);

    expect(uri.path, '/account-deletion/reauth-complete');
    expect(uri.queryParameters, {
      'job-id': 'job-123',
      'proof': 'single-use-proof',
    });
    expect(location, isNot(contains('token')));
    expect(location, isNot(contains('bearer')));
  });
}
