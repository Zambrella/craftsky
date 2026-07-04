import 'package:craftsky_app/shared/observability/sentry_config.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('SentryConfig', () {
    test('is disabled when DSN is absent or empty', () {
      expect(
        SentryConfig.fromValues(environment: 'production').enabled,
        isFalse,
      );
      expect(
        SentryConfig.fromValues(dsn: '', environment: 'staging').enabled,
        isFalse,
      );
    });

    test('is enabled for staging and production when DSN is present', () {
      expect(
        SentryConfig.fromValues(
          dsn: 'https://public@example.sentry.io/1',
          environment: 'staging',
        ).enabled,
        isTrue,
      );
      expect(
        SentryConfig.fromValues(
          dsn: 'https://public@example.sentry.io/1',
          environment: 'production',
        ).enabled,
        isTrue,
      );
    });

    test('requires explicit local opt-in outside staging and production', () {
      expect(
        SentryConfig.fromValues(
          dsn: 'https://public@example.sentry.io/1',
          environment: 'development',
        ).enabled,
        isFalse,
      );
      expect(
        SentryConfig.fromValues(
          dsn: 'https://public@example.sentry.io/1',
          environment: 'development',
          localOptIn: true,
        ).enabled,
        isTrue,
      );
    });

    test('preserves release and dist values for enabled configs', () {
      final config = SentryConfig.fromValues(
        dsn: 'https://public@example.sentry.io/1',
        environment: 'production',
        release: 'craftsky@1.0.0+1',
        dist: '1',
      );

      expect(config.release, 'craftsky@1.0.0+1');
      expect(config.dist, '1');
    });
  });
}
