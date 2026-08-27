// Separate controller actions keep the privacy assertions explicit.
// ignore_for_file: cascade_invocations

import 'dart:io';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/feed/models/link_preview.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'UT-020 Flutter preview failures expose no sensitive telemetry path',
    () async {
      const canary = 'https://private.example/path?token=secret';
      final reporter = _RecordingErrorReporter();
      final container = ProviderContainer.test(
        overrides: [
          linkPreviewRepositoryProvider.overrideWithValue(
            const _FailingRepository(canary),
          ),
          errorReporterProvider.overrideWithValue(reporter),
        ],
      );
      addTearDown(container.dispose);
      final provider = linkPreviewControllerProvider(
        'privacy',
        AccountKey('did:plc:alice'),
      );
      final subscription = container.listen(provider, (_, _) {});
      addTearDown(subscription.close);
      final controller = container.read(provider.notifier);

      controller.updateText('$canary ');
      await Future<void>.delayed(Duration.zero);
      await Future<void>.delayed(Duration.zero);

      expect(controller.available, isEmpty);
      expect(controller.state.toString(), isNot(contains(canary)));
      expect(reporter.captured, isEmpty);
      expect(reporter.messages, isEmpty);
      expect(reporter.breadcrumbs, isEmpty);
      for (final path in [
        'lib/feed/composer/link_preview_controller.dart',
        'lib/feed/data/post_api_client.dart',
        'lib/feed/widgets/composer_link_preview_carousel.dart',
        'lib/feed/widgets/external_card.dart',
        'lib/feed/widgets/post_composer_sheet.dart',
        'lib/scheduled_posts/data/scheduled_post_repository.dart',
        'lib/scheduled_posts/services/scheduled_composer_media.dart',
      ]) {
        final source = File(path).readAsStringSync();
        for (final forbidden in [
          'Sentry',
          'captureException',
          'captureMessage',
          'addBreadcrumb',
          'analytics',
        ]) {
          expect(source, isNot(contains(forbidden)), reason: path);
        }
      }
    },
  );
}

final class _RecordingErrorReporter implements ErrorReporter {
  final captured = <Object>[];
  final messages = <String>[];
  final breadcrumbs = <SafeBreadcrumb>[];

  @override
  bool get enabled => true;

  @override
  void addBreadcrumb(SafeBreadcrumb breadcrumb) => breadcrumbs.add(breadcrumb);

  @override
  Future<String?> captureException(
    Object error, {
    required ReportContext context,
    StackTrace? stackTrace,
  }) async {
    captured.add(error);
    return 'event';
  }

  @override
  Future<void> captureMessage(
    String message, {
    required ReportContext context,
  }) async {
    messages.add(message);
  }
}

final class _FailingRepository implements LinkPreviewRepository {
  const _FailingRepository(this.canary);

  final String canary;

  @override
  Future<LinkPreview> fetch(Uri url, CancelToken cancelToken) async {
    throw Exception(canary);
  }
}
