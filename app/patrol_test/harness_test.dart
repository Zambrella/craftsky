import 'package:flutter/material.dart';
import 'package:patrol/patrol.dart';

void main() {
  patrolTest('Patrol can render Flutter and control the device', ($) async {
    await $.pumpWidgetAndSettle(
      const MaterialApp(
        home: Scaffold(
          body: Text('Patrol ready', key: ValueKey('patrol-harness-ready')),
        ),
      ),
    );
    await $.platform.mobile.pressHome();
    await $.platform.mobile.openApp();
    await $(const ValueKey<String>('patrol-harness-ready')).waitUntilVisible();
  });
}
