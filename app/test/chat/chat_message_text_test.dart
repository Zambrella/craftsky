import 'package:craftsky_app/chat/widgets/chat_message_text.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:material_ui/material_ui.dart';

void main() {
  testWidgets('UT-008 original message links stay usable without a preview', (
    tester,
  ) async {
    const link = 'https://example.test/private-path?q=private-query';
    Uri? opened;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: ChatMessageText(text: link, onLink: (uri) => opened = uri),
        ),
      ),
    );
    expect(opened, isNull);
    expect(find.text(link), findsOneWidget);
    await tester.tap(find.text(link));
    await tester.pump();
    expect(opened?.toString(), link);
    expect(tester.takeException(), isNull);
  });
}
