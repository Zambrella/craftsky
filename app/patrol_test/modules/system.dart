import 'dart:io';

import 'package:craftsky_app/keys.dart';
import 'package:patrol/patrol.dart';

final class System {
  new(this.$);

  final PatrolIntegrationTester $;

  Future<void> chooseFixturePhotos() async {
    if (Platform.isIOS) {
      // Match actual photo tiles, excluding the picker's onboarding artwork.
      for (var index = 0; index < 2; index++) {
        await $.platform.ios.tap(
          IOSSelector(
            elementType: IOSElementType.image,
            identifier: 'PXGGridLayout-Info',
            instance: index,
          ),
        );
      }
      final selection = await $.platform.ios.getNativeViews(
        const IOSSelector(
          elementType: IOSElementType.image,
          identifier: 'PXGGridLayout-Info',
          isSelected: true,
        ),
      );
      if (selection.roots.length != 2) {
        throw StateError(
          'The iOS photo picker must select exactly two photos.',
        );
      }
      await $.platform.ios.tap(
        const IOSSelector(
          elementType: IOSElementType.button,
          identifier: 'Add',
        ),
      );
      await $(keys.composerImages.page('composer')).waitUntilVisible();
      return;
    }
    if (await $.platform.mobile.isPermissionDialogVisible()) {
      await $.platform.mobile.grantPermissionWhenInUse();
    }
    await $.platform.android.tap(
      const AndroidSelector(contentDescription: 'Collections'),
    );
    await $.platform.android.tap(
      const AndroidSelector(text: 'From this device'),
    );
    await $.platform.android.tap(const AndroidSelector(text: 'CraftSkyE2E'));
    await _waitForFixtureSelection(0);
    for (var selected = 1; selected <= 2; selected++) {
      await $.platform.android.tap(
        const AndroidSelector(
          contentDescriptionStartsWith: 'Photo taken on',
          instance: 0,
        ),
      );
      // A native tap completing does not guarantee the picker has updated its
      // accessibility tree. Wait for the selection before querying the next
      // unselected photo, otherwise the same tile can be toggled off again.
      await _waitForFixtureSelection(selected);
    }
    await $.platform.android.tap(const AndroidSelector(text: 'Done'));
    await $(keys.composerImages.page('composer')).waitUntilVisible();
  }

  Future<void> _waitForFixtureSelection(int expected) async {
    final deadline = DateTime.now().add(const Duration(seconds: 15));
    var selected = 0;
    var unselected = 0;
    var doneVisible = false;
    do {
      final tree = await $.platform.android.getNativeViews(null);
      final views = tree.roots.expand(_flatten).toList();
      doneVisible = views.any((view) => view.text == 'Done');
      selected = views
          .where(
            (view) =>
                view.contentDescription?.startsWith(
                  'Selected Photo taken on',
                ) ??
                false,
          )
          .length;
      unselected = views
          .where(
            (view) =>
                view.contentDescription?.startsWith('Photo taken on') ?? false,
          )
          .length;
      final countMatches =
          expected == 0 ||
          views.any(
            (view) =>
                view.contentDescription ==
                '$expected photos or videos selected',
          );
      if (selected == expected &&
          selected + unselected == 2 &&
          countMatches &&
          (expected != 2 || doneVisible)) {
        return;
      }
      await Future<void>.delayed(const Duration(milliseconds: 250));
    } while (DateTime.now().isBefore(deadline));
    throw StateError(
      'Photo picker expected $expected selected fixtures out of 2; '
      'found $selected selected and $unselected unselected. '
      'Done visible: $doneVisible. '
      'Inspect the CraftSkyE2E album on the emulator.',
    );
  }

  Iterable<AndroidNativeView> _flatten(AndroidNativeView view) sync* {
    yield view;
    for (final child in view.children) {
      yield* _flatten(child);
    }
  }
}
