import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/data/mock_facet_suggestion_repository.dart';
import 'package:craftsky_app/shared/rich_text/providers/facet_suggestion_providers.dart';
import 'package:craftsky_app/shared/rich_text/widgets/facet_autocomplete_editor.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('FacetAutocompleteEditor', () {
    testWidgets(
      'AT-003 shows mention suggestions and inserts selected handle',
      (tester) async {
        final controller = FacetTextEditingController();
        final focusNode = FocusNode();
        addTearDown(controller.dispose);
        addTearDown(focusNode.dispose);

        await tester.pumpWidget(
          _wrap(
            overrides: [
              facetAutocompleteDebounceProvider.overrideWithValue(
                Duration.zero,
              ),
              accountSuggestionRepositoryProvider.overrideWithValue(
                const MockAccountSuggestionRepository(
                  accounts: [
                    AccountSuggestion(
                      did: 'did:plc:alicia',
                      handle: 'alicia.craftsky.social',
                      displayName: 'Alicia',
                      avatar: 'https://example.com/alicia.jpg',
                      isCraftskyProfile: true,
                      viewerIsFollowing: false,
                    ),
                    AccountSuggestion(
                      did: 'did:plc:mallory',
                      handle: 'alice.elsewhere.example',
                      displayName: 'Mallory',
                      avatar: 'https://example.com/mallory.jpg',
                      isCraftskyProfile: false,
                      viewerIsFollowing: true,
                    ),
                    AccountSuggestion(
                      did: 'did:plc:alice',
                      handle: 'alice.craftsky.social',
                      displayName: 'Alice',
                      avatar: 'https://example.com/alice.jpg',
                      isCraftskyProfile: true,
                      viewerIsFollowing: true,
                    ),
                  ],
                ),
              ),
            ],
            child: FacetAutocompleteEditor(
              label: 'Body',
              controller: controller,
              focusNode: focusNode,
            ),
          ),
        );

        await tester.enterText(find.byType(TextField), '@ali');
        await tester.pumpAndSettle();

        expect(find.text('Alice'), findsOneWidget);
        expect(find.text('@alice.craftsky.social'), findsOneWidget);
        expect(find.text('Alicia'), findsOneWidget);
        expect(find.text('@alicia.craftsky.social'), findsOneWidget);
        expect(find.text('Mallory'), findsNothing);
        expect(find.text('@alice.elsewhere.example'), findsNothing);
        expect(
          find.byWidgetPredicate(
            (widget) =>
                widget is Semantics &&
                widget.properties.label == 'Avatar for Alice',
          ),
          findsOneWidget,
        );

        final aliceTop = tester.getTopLeft(find.text('Alice')).dy;
        final aliciaTop = tester.getTopLeft(find.text('Alicia')).dy;
        expect(aliceTop, lessThan(aliciaTop));

        await tester.tap(find.text('@alice.craftsky.social'));
        await tester.pump();

        expect(controller.text, '@alice.craftsky.social ');
        expect(controller.selection.baseOffset, controller.text.length);
        expect(focusNode.hasFocus, isTrue);
      },
    );

    testWidgets(
      'AT-004 shows hashtag suggestions with counts and inserts canonical tag',
      (tester) async {
        final controller = FacetTextEditingController();
        final focusNode = FocusNode();
        addTearDown(controller.dispose);
        addTearDown(focusNode.dispose);

        await tester.pumpWidget(
          _wrap(
            overrides: [
              facetAutocompleteDebounceProvider.overrideWithValue(
                Duration.zero,
              ),
              hashtagSuggestionRepositoryProvider.overrideWithValue(
                const MockHashtagSuggestionRepository(
                  hashtags: [
                    HashtagSuggestion(tag: 'SockKAL', postsLast28Days: 128),
                    HashtagSuggestion(tag: 'sockmending', postsLast28Days: 12),
                  ],
                ),
              ),
            ],
            child: FacetAutocompleteEditor(
              label: 'Body',
              controller: controller,
              focusNode: focusNode,
            ),
          ),
        );

        await tester.enterText(find.byType(TextField), '#sock');
        await tester.pumpAndSettle();

        expect(find.text('#SockKAL'), findsOneWidget);
        expect(find.text('128 posts'), findsOneWidget);
        expect(find.text('#sockmending'), findsOneWidget);
        expect(find.text('12 posts'), findsOneWidget);
        expect(find.textContaining('in the last 28 days'), findsNothing);
        expect(find.byIcon(Icons.show_chart), findsNWidgets(2));

        await tester.tap(find.text('#SockKAL'));
        await tester.pump();

        expect(controller.text, '#SockKAL ');
        expect(controller.selection.baseOffset, controller.text.length);
        expect(focusNode.hasFocus, isTrue);
      },
    );

    testWidgets(
      'IR-001 styles active mention and hashtag tokens with theme primary '
      'color',
      (tester) async {
        final controller = FacetTextEditingController();
        final focusNode = FocusNode();
        addTearDown(controller.dispose);
        addTearDown(focusNode.dispose);

        await tester.pumpWidget(
          _wrap(
            overrides: const [],
            child: FacetAutocompleteEditor(
              label: 'Body',
              controller: controller,
              focusNode: focusNode,
            ),
          ),
        );

        await tester.enterText(find.byType(TextField), 'Hello @ali');
        final context = tester.element(find.byType(TextField));
        final theme = Theme.of(context);
        final mentionSpan = controller.buildTextSpan(
          context: context,
          style: theme.textTheme.bodyLarge,
          withComposing: false,
        );

        expect(
          _spanForText(mentionSpan, '@ali')?.style?.color,
          theme.colorScheme.primary,
        );

        await tester.enterText(find.byType(TextField), 'Casting on #sock');
        final hashtagSpan = controller.buildTextSpan(
          context: context,
          style: theme.textTheme.bodyLarge,
          withComposing: false,
        );

        expect(
          _spanForText(hashtagSpan, '#sock')?.style?.color,
          theme.colorScheme.primary,
        );

        await tester.enterText(
          find.byType(TextField),
          'Hello @ali and #sock done',
        );
        final persistentSpan = controller.buildTextSpan(
          context: context,
          style: theme.textTheme.bodyLarge,
          withComposing: false,
        );

        expect(
          _spanForText(persistentSpan, '@ali')?.style?.color,
          theme.colorScheme.primary,
        );
        expect(
          _spanForText(persistentSpan, '#sock')?.style?.color,
          theme.colorScheme.primary,
        );
      },
    );

    testWidgets('styles valid web URLs with theme primary color', (
      tester,
    ) async {
      final controller = FacetTextEditingController();
      final focusNode = FocusNode();
      addTearDown(controller.dispose);
      addTearDown(focusNode.dispose);

      await tester.pumpWidget(
        _wrap(
          overrides: const [],
          child: FacetAutocompleteEditor(
            label: 'Body',
            controller: controller,
            focusNode: focusNode,
          ),
        ),
      );

      await tester.enterText(find.byType(TextField), 'See craftsky.social');
      final context = tester.element(find.byType(TextField));
      final theme = Theme.of(context);
      final validUrlSpan = controller.buildTextSpan(
        context: context,
        style: theme.textTheme.bodyLarge,
        withComposing: false,
      );

      expect(
        _spanForText(validUrlSpan, 'craftsky.social')?.style?.color,
        theme.colorScheme.primary,
      );

      await tester.enterText(find.byType(TextField), 'See craftsky');
      final partialUrlSpan = controller.buildTextSpan(
        context: context,
        style: theme.textTheme.bodyLarge,
        withComposing: false,
      );

      expect(_spanForText(partialUrlSpan, 'craftsky')?.style?.color, isNull);
    });

    testWidgets('positions suggestions below the active token start', (
      tester,
    ) async {
      final controller = FacetTextEditingController();
      final focusNode = FocusNode();
      addTearDown(controller.dispose);
      addTearDown(focusNode.dispose);

      await tester.pumpWidget(
        _wrap(
          overrides: [
            facetAutocompleteDebounceProvider.overrideWithValue(Duration.zero),
            accountSuggestionRepositoryProvider.overrideWithValue(
              const MockAccountSuggestionRepository(
                accounts: [
                  AccountSuggestion(
                    did: 'did:plc:alice',
                    handle: 'alice.craftsky.social',
                    displayName: 'Alice',
                    avatar: null,
                    isCraftskyProfile: true,
                    viewerIsFollowing: false,
                  ),
                ],
              ),
            ),
          ],
          child: SizedBox(
            width: 480,
            child: FacetAutocompleteEditor(
              label: 'Body',
              controller: controller,
              focusNode: focusNode,
            ),
          ),
        ),
      );

      const text = 'Leading words @ali';
      await tester.enterText(find.byType(TextField), text);
      await tester.pumpAndSettle();

      final tokenStart = _tokenStartGlobalOffset(tester, text.indexOf('@'));
      final cardTopLeft = tester.getTopLeft(find.byType(Card));

      expect(cardTopLeft.dx, closeTo(tokenStart.dx, 1));
      expect(cardTopLeft.dy, greaterThan(tokenStart.dy));
      expect(tester.getRect(find.byType(Card)).width, closeTo(300, 1));
    });

    testWidgets('scales suggestion width with the current text scaler', (
      tester,
    ) async {
      final controller = FacetTextEditingController();
      final focusNode = FocusNode();
      addTearDown(controller.dispose);
      addTearDown(focusNode.dispose);

      await tester.pumpWidget(
        _wrap(
          textScaler: TextScaler.linear(1.5),
          overrides: [
            facetAutocompleteDebounceProvider.overrideWithValue(Duration.zero),
            accountSuggestionRepositoryProvider.overrideWithValue(
              const MockAccountSuggestionRepository(
                accounts: [
                  AccountSuggestion(
                    did: 'did:plc:alice',
                    handle: 'alice.craftsky.social',
                    displayName: 'Alice',
                    avatar: null,
                    isCraftskyProfile: true,
                    viewerIsFollowing: false,
                  ),
                ],
              ),
            ),
          ],
          child: SizedBox(
            width: 560,
            child: FacetAutocompleteEditor(
              label: 'Body',
              controller: controller,
              focusNode: focusNode,
            ),
          ),
        ),
      );

      await tester.enterText(find.byType(TextField), '@ali');
      await tester.pumpAndSettle();

      expect(tester.getRect(find.byType(Card)).width, closeTo(450, 1));
    });

    testWidgets('shifts suggestions left when needed to avoid overflow', (
      tester,
    ) async {
      await _withSurfaceSize(
        tester,
        const Size(400, 600),
        () async {
          final controller = FacetTextEditingController();
          final focusNode = FocusNode();
          addTearDown(controller.dispose);
          addTearDown(focusNode.dispose);

          await tester.pumpWidget(
            _wrap(
              overrides: [
                facetAutocompleteDebounceProvider.overrideWithValue(
                  Duration.zero,
                ),
                accountSuggestionRepositoryProvider.overrideWithValue(
                  const MockAccountSuggestionRepository(
                    accounts: [
                      AccountSuggestion(
                        did: 'did:plc:alice',
                        handle: 'alice.craftsky.social',
                        displayName: 'Alice',
                        avatar: null,
                        isCraftskyProfile: true,
                        viewerIsFollowing: false,
                      ),
                    ],
                  ),
                ),
              ],
              child: SizedBox(
                width: 352,
                child: FacetAutocompleteEditor(
                  label: 'Body',
                  controller: controller,
                  focusNode: focusNode,
                ),
              ),
            ),
          );

          const text = 'Long leading words before @ali';
          await tester.enterText(find.byType(TextField), text);
          await tester.pumpAndSettle();

          final tokenStart = _tokenStartGlobalOffset(tester, text.indexOf('@'));
          final cardRect = tester.getRect(find.byType(Card));

          expect(cardRect.left, lessThan(tokenStart.dx));
          expect(cardRect.right, lessThanOrEqualTo(392));
          expect(cardRect.left, closeTo(392 - cardRect.width, 1));
        },
      );
    });

    testWidgets('closes suggestions when the text field loses focus', (
      tester,
    ) async {
      final controller = FacetTextEditingController();
      final focusNode = FocusNode();
      addTearDown(controller.dispose);
      addTearDown(focusNode.dispose);

      await tester.pumpWidget(
        _wrap(
          overrides: [
            facetAutocompleteDebounceProvider.overrideWithValue(Duration.zero),
            accountSuggestionRepositoryProvider.overrideWithValue(
              const MockAccountSuggestionRepository(
                accounts: [
                  AccountSuggestion(
                    did: 'did:plc:alice',
                    handle: 'alice.craftsky.social',
                    displayName: 'Alice',
                    avatar: null,
                    isCraftskyProfile: true,
                    viewerIsFollowing: false,
                  ),
                ],
              ),
            ),
          ],
          child: FacetAutocompleteEditor(
            label: 'Body',
            controller: controller,
            focusNode: focusNode,
          ),
        ),
      );

      await tester.enterText(find.byType(TextField), '@ali');
      await tester.pumpAndSettle();

      expect(find.byType(Card), findsOneWidget);

      focusNode.unfocus();
      await tester.pumpAndSettle();

      expect(find.byType(Card), findsNothing);
    });
  });
}

TextSpan? _spanForText(TextSpan root, String text) {
  if (root.text == text) {
    return root;
  }
  final children = root.children;
  if (children == null) {
    return null;
  }
  for (final child in children) {
    if (child is TextSpan) {
      final match = _spanForText(child, text);
      if (match != null) {
        return match;
      }
    }
  }
  return null;
}

Offset _tokenStartGlobalOffset(WidgetTester tester, int tokenStart) {
  final editableRoot = tester.renderObject<RenderObject>(
    find.byType(EditableText),
  );
  final renderEditable = _findRenderEditable(editableRoot)!;
  final caret = renderEditable.getLocalRectForCaret(
    TextPosition(offset: tokenStart),
  );
  return renderEditable.localToGlobal(caret.bottomLeft);
}

RenderEditable? _findRenderEditable(RenderObject root) {
  if (root is RenderEditable) {
    return root;
  }
  RenderEditable? match;
  root.visitChildren((child) {
    match ??= _findRenderEditable(child);
  });
  return match;
}

Future<void> _withSurfaceSize(
  WidgetTester tester,
  Size size,
  Future<void> Function() body,
) async {
  final previousSize = tester.view.physicalSize;
  final previousDevicePixelRatio = tester.view.devicePixelRatio;
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(() {
    tester.view.physicalSize = previousSize;
    tester.view.devicePixelRatio = previousDevicePixelRatio;
  });
  await body();
}

Widget _wrap({
  required Widget child,
  required List<dynamic> overrides,
  TextScaler? textScaler,
}) {
  return ProviderScope(
    overrides: List.from(overrides),
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      home: Scaffold(
        body: Builder(
          builder: (context) => MediaQuery(
            data: MediaQuery.of(context).copyWith(
              textScaler: textScaler ?? TextScaler.noScaling,
            ),
            child: Padding(padding: const EdgeInsets.all(24), child: child),
          ),
        ),
      ),
    ),
  );
}
