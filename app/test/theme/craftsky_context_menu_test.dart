import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_divider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  Future<void> pumpHarness(
    WidgetTester tester, {
    required Size size,
    required Widget child,
  }) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightThemeData,
        home: Scaffold(body: Center(child: child)),
      ),
    );
  }

  group('CraftskyContextMenuButton', () {
    testWidgets('opens a bottom sheet on compact screens', (tester) async {
      var tapped = false;

      await pumpHarness(
        tester,
        size: const Size(390, 844),
        child: CraftskyContextMenuButton(
          groups: [
            CraftskyContextMenuGroup(
              items: [
                CraftskyContextMenuItem(
                  text: 'Report',
                  icon: Icons.flag_outlined,
                  onPressed: () => tapped = true,
                ),
              ],
            ),
          ],
        ),
      );

      await tester.tap(find.byIcon(Icons.more_horiz));
      await tester.pumpAndSettle();

      expect(find.byType(BottomSheet), findsOneWidget);
      expect(find.text('Report'), findsOneWidget);

      await tester.tap(find.text('Report'));
      await tester.pumpAndSettle();

      expect(tapped, isTrue);
      expect(find.byType(BottomSheet), findsNothing);
    });

    testWidgets('opens a popup menu on wide screens', (tester) async {
      var tapped = false;

      await pumpHarness(
        tester,
        size: const Size(1200, 800),
        child: CraftskyContextMenuButton(
          groups: [
            CraftskyContextMenuGroup(
              items: [
                CraftskyContextMenuItem(
                  text: 'Share',
                  icon: Icons.ios_share,
                  onPressed: () => tapped = true,
                ),
              ],
            ),
          ],
        ),
      );

      await tester.tap(find.byIcon(Icons.more_horiz));
      await tester.pumpAndSettle();

      expect(find.byType(BottomSheet), findsNothing);
      expect(
        find.byType(PopupMenuItem<CraftskyContextMenuItem>),
        findsOneWidget,
      );
      expect(find.text('Share'), findsOneWidget);

      await tester.tap(find.text('Share'));
      await tester.pumpAndSettle();

      expect(tapped, isTrue);
    });

    testWidgets('opens a wide popup below a target near the top', (
      tester,
    ) async {
      await pumpHarness(
        tester,
        size: const Size(1200, 800),
        child: Align(
          alignment: Alignment.topCenter,
          child: Padding(
            padding: const EdgeInsets.only(top: 80),
            child: CraftskyContextMenuButton(
              groups: [
                CraftskyContextMenuGroup(
                  items: [
                    CraftskyContextMenuItem(
                      text: 'Share',
                      icon: Icons.ios_share,
                      onPressed: () {},
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      );

      final target = find.byIcon(Icons.more_horiz);
      final targetRect = tester.getRect(target);
      await tester.tap(target);
      await tester.pumpAndSettle();

      final menuItem = find.byType(
        PopupMenuItem<CraftskyContextMenuItem>,
      );
      expect(
        tester.getRect(menuItem).top,
        greaterThanOrEqualTo(targetRect.bottom),
      );
    });

    testWidgets('opens a wide popup above a target near the bottom', (
      tester,
    ) async {
      await pumpHarness(
        tester,
        size: const Size(1200, 800),
        child: Align(
          alignment: Alignment.bottomCenter,
          child: Padding(
            padding: const EdgeInsets.only(bottom: 80),
            child: CraftskyContextMenuButton(
              groups: [
                CraftskyContextMenuGroup(
                  items: [
                    CraftskyContextMenuItem(
                      text: 'Share',
                      icon: Icons.ios_share,
                      onPressed: () {},
                    ),
                    CraftskyContextMenuItem(
                      text: 'Report',
                      icon: Icons.flag_outlined,
                      onPressed: () {},
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      );

      final target = find.byIcon(Icons.more_horiz);
      final targetRect = tester.getRect(target);
      await tester.tap(target);
      await tester.pumpAndSettle();

      final menuItems = find.byType(
        PopupMenuItem<CraftskyContextMenuItem>,
      );
      final menuBottom = tester.getRect(menuItems.last).bottom;
      expect(menuBottom, lessThanOrEqualTo(targetRect.top));
    });

    testWidgets('anchors a wide popup in its nested content overlay', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(1200, 800);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);

      await tester.pumpWidget(
        MaterialApp(
          theme: AppTheme.lightThemeData,
          home: Scaffold(
            body: Row(
              children: [
                const SizedBox(width: 300),
                Expanded(
                  child: Navigator(
                    onGenerateRoute: (_) => MaterialPageRoute<void>(
                      builder: (_) => Scaffold(
                        body: Center(
                          child: CraftskyContextMenuButton(
                            groups: [
                              CraftskyContextMenuGroup(
                                items: [
                                  CraftskyContextMenuItem(
                                    text: 'Repost',
                                    icon: Icons.repeat,
                                    onPressed: () {},
                                  ),
                                ],
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      );

      final button = find.byIcon(Icons.more_horiz);
      await tester.tap(button);
      await tester.pumpAndSettle();

      final menuItem = find.byType(
        PopupMenuItem<CraftskyContextMenuItem>,
      );
      expect(
        (tester.getTopLeft(menuItem).dx - tester.getTopLeft(button).dx).abs(),
        lessThan(80),
      );
    });

    testWidgets('separates logical groups in compact sheet', (tester) async {
      await pumpHarness(
        tester,
        size: const Size(390, 844),
        child: CraftskyContextMenuButton(
          groups: [
            CraftskyContextMenuGroup(
              items: [
                CraftskyContextMenuItem(
                  text: 'Copy link',
                  icon: Icons.link,
                  onPressed: () {},
                ),
              ],
            ),
            CraftskyContextMenuGroup(
              items: [
                CraftskyContextMenuItem(
                  text: 'Block',
                  icon: Icons.block,
                  onPressed: () {},
                ),
              ],
            ),
          ],
        ),
      );

      await tester.tap(find.byIcon(Icons.more_horiz));
      await tester.pumpAndSettle();

      expect(find.text('Copy link'), findsOneWidget);
      expect(find.text('Block'), findsOneWidget);
      expect(find.byType(CraftskyDivider), findsOneWidget);
    });

    testWidgets('applies destructive item styling', (tester) async {
      await pumpHarness(
        tester,
        size: const Size(390, 844),
        child: CraftskyContextMenuButton(
          groups: [
            CraftskyContextMenuGroup(
              items: [
                CraftskyContextMenuItem(
                  text: 'Delete',
                  icon: Icons.delete_outline,
                  onPressed: () {},
                  style: CraftskyContextMenuItemStyle.destructive,
                ),
              ],
            ),
          ],
        ),
      );

      await tester.tap(find.byIcon(Icons.more_horiz));
      await tester.pumpAndSettle();

      final icon = tester.widget<Icon>(find.byIcon(Icons.delete_outline));
      final text = tester.widget<Text>(find.text('Delete'));

      expect(icon.color, BrandColors.red);
      expect(text.style?.color, BrandColors.red);
    });

    testWidgets('does not fire disabled item callbacks', (tester) async {
      var tapped = false;

      await pumpHarness(
        tester,
        size: const Size(390, 844),
        child: CraftskyContextMenuButton(
          groups: [
            CraftskyContextMenuGroup(
              items: [
                const CraftskyContextMenuItem(
                  text: 'Disabled',
                  icon: Icons.lock_outline,
                  onPressed: null,
                ),
                CraftskyContextMenuItem(
                  text: 'Enabled',
                  icon: Icons.check,
                  onPressed: () => tapped = true,
                ),
              ],
            ),
          ],
        ),
      );

      await tester.tap(find.byIcon(Icons.more_horiz));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Disabled'));
      await tester.pumpAndSettle();

      expect(tapped, isFalse);
      expect(find.byType(BottomSheet), findsOneWidget);
    });

    testWidgets('disabled menu button does not open a menu', (tester) async {
      await pumpHarness(
        tester,
        size: const Size(390, 844),
        child: CraftskyContextMenuButton(
          enabled: false,
          groups: [
            CraftskyContextMenuGroup(
              items: [
                CraftskyContextMenuItem(
                  text: 'Share',
                  icon: Icons.ios_share_outlined,
                  onPressed: () {},
                ),
              ],
            ),
          ],
        ),
      );

      await tester.tap(find.byIcon(Icons.more_horiz));
      await tester.pumpAndSettle();

      expect(find.byType(BottomSheet), findsNothing);
      expect(find.text('Share'), findsNothing);
    });
  });
}
