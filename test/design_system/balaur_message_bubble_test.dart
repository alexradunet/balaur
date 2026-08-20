import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('aligns a Household Member message to the right', (tester) async {
    await tester.pumpWidget(
      _TestApp(
        child: BalaurMessageBubble(
          content: 'Add milk.',
          role: BalaurMessageBubbleRole.householdMember,
        ),
      ),
    );

    final align = tester.widget<Align>(
      find.descendant(
        of: find.byType(BalaurMessageBubble),
        matching: find.byType(Align),
      ),
    );

    expect(align.alignment, Alignment.centerRight);
  });

  testWidgets('shows the streaming state', (tester) async {
    await tester.pumpWidget(
      const _TestApp(
        child: BalaurMessageBubble(
          content: '',
          role: BalaurMessageBubbleRole.agent,
          status: BalaurMessageBubbleStatus.streaming,
        ),
      ),
    );

    expect(find.bySemanticsLabel('Agent response in progress'), findsOneWidget);
    expect(find.text('thinking…'), findsOneWidget);
  });

  testWidgets('uses the compact portrait below 640 pixels', (tester) async {
    tester.view.devicePixelRatio = 1;
    tester.view.physicalSize = const Size(360, 800);
    addTearDown(tester.view.reset);

    await tester.pumpWidget(
      const _TestApp(
        child: BalaurMessageBubble(
          content: 'Add milk.',
          role: BalaurMessageBubbleRole.householdMember,
        ),
      ),
    );

    expect(tester.widget<BalaurAvatar>(find.byType(BalaurAvatar)).size, 50);
  });

  testWidgets('shows stopped and failed states', (tester) async {
    await tester.pumpWidget(
      const _TestApp(
        child: Column(
          children: [
            BalaurMessageBubble(
              content: 'I added milk and',
              role: BalaurMessageBubbleRole.agent,
              status: BalaurMessageBubbleStatus.stopped,
            ),
            BalaurMessageBubble(
              content: 'I could not update the Shopping List.',
              role: BalaurMessageBubbleRole.agent,
              status: BalaurMessageBubbleStatus.failed,
            ),
          ],
        ),
      ),
    );

    expect(find.text('Stopped'), findsOneWidget);
    expect(find.text('Failed'), findsOneWidget);
  });
}

class _TestApp extends StatelessWidget {
  const _TestApp({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      theme: BalaurTheme.light(),
      home: Scaffold(body: child),
    );
  }
}
