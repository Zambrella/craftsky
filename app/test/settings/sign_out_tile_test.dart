import 'dart:async';

import 'package:craftsky_app/auth/providers/auth_controller.dart';
import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeAuthController extends AuthController {
  int signOutCalls = 0;
  @override
  FutureOr<void> build() => null;
  @override
  Future<void> signOut() async {
    signOutCalls++;
  }
}

void main() {
  testWidgets('tapping the tile calls AuthController.signOut', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authControllerProvider.overrideWith(_FakeAuthController.new),
        ],
        child: const MaterialApp(home: Material(child: SignOutTile())),
      ),
    );
    await tester.tap(find.byType(SignOutTile));
    await tester.pump();

    final fake = tester.container().read(authControllerProvider.notifier)
        as _FakeAuthController;
    expect(fake.signOutCalls, 1);
  });
}
