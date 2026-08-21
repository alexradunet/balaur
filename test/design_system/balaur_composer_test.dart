import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('Enter sends a trimmed draft and clears the field', (
    tester,
  ) async {
    final messages = <String>[];
    await tester.pumpWidget(
      _TestApp(child: BalaurComposer(onSend: messages.add)),
    );

    final field = find.byKey(const Key('chat-composer'));
    await tester.enterText(field, '  Add milk.  ');
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pump();

    expect(messages, ['Add milk.']);
    expect(tester.widget<TextField>(field).controller?.text, isEmpty);
  });

  testWidgets('Shift and Enter add a line without sending', (tester) async {
    final messages = <String>[];
    await tester.pumpWidget(
      _TestApp(child: BalaurComposer(onSend: messages.add)),
    );

    final field = find.byKey(const Key('chat-composer'));
    await tester.enterText(field, 'First line');
    await tester.sendKeyDownEvent(LogicalKeyboardKey.shiftLeft);
    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.sendKeyUpEvent(LogicalKeyboardKey.shiftLeft);
    await tester.pump();

    expect(messages, isEmpty);
    expect(tester.widget<TextField>(field).controller?.text, 'First line\n');
  });

  testWidgets('dialogue choices replace the draft and accept number keys', (
    tester,
  ) async {
    int? selected;
    await tester.pumpWidget(
      _TestApp(
        child: BalaurComposer.choices(
          choices: const [
            BalaurDialogueChoice(label: 'Keep it.'),
            BalaurDialogueChoice(label: 'Not this.'),
          ],
          onPick: (index) => selected = index,
        ),
      ),
    );

    expect(find.byKey(const Key('chat-composer')), findsNothing);
    expect(find.byType(BalaurAvatar), findsOneWidget);
    expect(find.text('Keep it.'), findsOneWidget);

    await tester.sendKeyEvent(LogicalKeyboardKey.digit2);
    await tester.pump();

    expect(selected, 1);
  });

  testWidgets('responding disables the draft and provides Stop', (
    tester,
  ) async {
    var stopped = false;
    await tester.pumpWidget(
      _TestApp(
        child: BalaurComposer(
          onSend: (_) {},
          responding: true,
          onStop: () => stopped = true,
        ),
      ),
    );

    final field = tester.widget<TextField>(
      find.byKey(const Key('chat-composer')),
    );
    expect(field.enabled, isFalse);
    expect(find.byKey(const Key('send-button')), findsNothing);

    await tester.tap(find.byKey(const Key('stop-button')));
    await tester.pump();

    expect(stopped, isTrue);
  });

  testWidgets('disabled state prevents sending', (tester) async {
    await tester.pumpWidget(
      _TestApp(child: BalaurComposer(onSend: (_) {}, enabled: false)),
    );

    final field = tester.widget<TextField>(
      find.byKey(const Key('chat-composer')),
    );
    final send = tester.widget<BalaurButton>(
      find.byKey(const Key('send-button')),
    );

    expect(field.enabled, isFalse);
    expect(send.onPressed, isNull);
  });
}

class _TestApp extends StatelessWidget {
  const _TestApp({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      theme: BalaurTheme.light(),
      home: Scaffold(
        body: Center(child: SizedBox(width: 700, child: child)),
      ),
    );
  }
}
