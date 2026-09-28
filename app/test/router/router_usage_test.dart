import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

import '../test_support/source_scan.dart';

void main() {
  test('application navigation uses typed route methods where possible', () {
    const rawNavigationExceptions = {
      'lib/router/router.dart': 'defines routes and redirect locations',
      'lib/router/app_shell.dart': 'adapts dynamic destinations and branches',
      'lib/auth/providers/account_boundary_provider.dart':
          'runs without a widget context',
      'lib/notifications/services/notification_navigation.dart':
          'runs above the Router inherited widget',
      'lib/notifications/providers/notification_runtime_provider.dart':
          'runs without a widget context',
    };
    final violations = <String>[];

    for (final file in scanDartSources('lib')) {
      final path = file.path;
      if (rawNavigationExceptions.containsKey(path)) continue;

      final source = file.source;
      if (RegExp(
            r'\b(?:context|router)\.(?:go|push|replace)\(',
          ).hasMatch(source) ||
          RegExp(
            r'GoRouter\.(?:of|maybeOf)\([^)]*\)\??\.(?:go|push|replace)',
          ).hasMatch(source)) {
        violations.add(path);
      }
    }

    expect(
      violations,
      isEmpty,
      reason:
          'Use Route(args).go(context), Route(args).push(context), or '
          'Route(args).replace(context). Add an exception only when no widget '
          'context or typed route operation can preserve the required '
          'behavior.',
    );
  });

  test('UT-009 known profile navigation does not use mutable handles', () {
    final violations = <String>[];

    for (final file in scanDartSources('lib')) {
      final path = file.path;
      final source = file.source;
      if (source.contains('UserProfileRoute(handle:') ||
          source.contains('showUserProfileCard(context, handleOrDid:') ||
          source.contains("'/profile/:handle'") ||
          source.contains("'${r'$'}{RouteLocations.profile}/:handle'")) {
        violations.add(path);
      }
    }

    expect(
      violations,
      isEmpty,
      reason:
          'Known identities must navigate by DID. Handles are display values '
          'or explicit /profiles/@handle alias input only.',
    );
  });

  test('UT-018 route declarations separate canonical DIDs from aliases', () {
    final source = File('lib/router/router.dart').readAsStringSync();

    expect(
      source,
      allOf(
        contains(r"path: '${RouteLocations.profiles}/:did'"),
        contains('final Did did;'),
        contains(r"path: '${RouteLocations.profiles}/@:handle'"),
        contains('final Handle handle;'),
        contains('return UserProfileRoute(did: profile.did).location;'),
      ),
    );
    expect(source, isNot(contains('UserProfileRoute({required this.handle')));
    expect(
      source,
      isNot(contains(r"path: '${RouteLocations.profile}/:handle'")),
    );
  });
}
