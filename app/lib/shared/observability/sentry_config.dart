final class SentryConfig {
  const SentryConfig._({
    required this.dsn,
    required this.environment,
    required this.release,
    required this.dist,
    required this.localOptIn,
  });

  factory SentryConfig.fromEnvironment() => SentryConfig.fromValues(
    dsn: const String.fromEnvironment('SENTRY_DSN'),
    environment: const String.fromEnvironment('SENTRY_ENVIRONMENT'),
    release: const String.fromEnvironment('SENTRY_RELEASE'),
    dist: const String.fromEnvironment('SENTRY_DIST'),
    localOptIn: _sentryLocalOptInFromEnvironment(),
  );

  factory SentryConfig.fromValues({
    String? dsn,
    String? environment,
    String? release,
    String? dist,
    bool localOptIn = false,
  }) {
    return SentryConfig._(
      dsn: _emptyToNull(dsn),
      environment: _emptyToNull(environment)?.toLowerCase() ?? 'development',
      release: _emptyToNull(release),
      dist: _emptyToNull(dist),
      localOptIn: localOptIn,
    );
  }

  final String? dsn;
  final String environment;
  final String? release;
  final String? dist;
  final bool localOptIn;

  bool get enabled {
    if (dsn == null) return false;
    return switch (environment) {
      'production' || 'staging' => true,
      _ => localOptIn,
    };
  }

  SentryFeatureOptions get options => const SentryFeatureOptions();
}

final class SentryFeatureOptions {
  const SentryFeatureOptions({
    this.sendDefaultPii = false,
    this.enableLogs = true,
    this.tracingEnabled = false,
    this.profilingEnabled = false,
    this.metricsEnabled = false,
    this.sessionReplayEnabled = false,
  });

  final bool sendDefaultPii;
  final bool enableLogs;
  final bool tracingEnabled;
  final bool profilingEnabled;
  final bool metricsEnabled;
  final bool sessionReplayEnabled;
}

String? _emptyToNull(String? value) {
  final trimmed = value?.trim();
  return trimmed == null || trimmed.isEmpty ? null : trimmed;
}

bool _sentryLocalOptInFromEnvironment() =>
    const bool.fromEnvironment('SENTRY_LOCAL_OPT_IN');
