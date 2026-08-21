import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';

void main() {
  testWidgets('keeps a Household Member portrait on the right edge', (
    tester,
  ) async {
    await tester.pumpWidget(
      const _TestApp(
        child: BalaurMessageBubble(
          content: 'Add milk.',
          role: BalaurMessageBubbleRole.householdMember,
        ),
      ),
    );

    final portrait = find
        .ancestor(
          of: find.byType(BalaurAvatar),
          matching: find.byType(BalaurSurface),
        )
        .first;
    final bubbleRect = tester.getRect(find.byType(BalaurMessageBubble));
    final portraitRect = tester.getRect(portrait);

    expect(portraitRect.right, closeTo(bubbleRect.right, 0.1));
  });

  testWidgets('uses the message text style for the streaming state', (
    tester,
  ) async {
    await tester.pumpWidget(
      const _TestApp(
        child: Column(
          children: [
            BalaurMessageBubble(
              content: 'Normal message.',
              role: BalaurMessageBubbleRole.householdMember,
            ),
            BalaurMessageBubble(
              content: '',
              role: BalaurMessageBubbleRole.agent,
              status: BalaurMessageBubbleStatus.streaming,
            ),
          ],
        ),
      ),
    );

    final message = tester.widget<SelectableText>(find.byType(SelectableText));
    final indicator = tester.widget<Text>(find.text('thinking…'));

    expect(find.bySemanticsLabel('Agent response in progress'), findsOneWidget);
    expect(indicator.style, message.style);
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

  testWidgets('formats agent Markdown', (tester) async {
    const content = '''
## Shopping plan

Use **milk** and `notes.md`.''';

    await tester.pumpWidget(
      const _TestApp(
        child: BalaurMessageBubble(
          content: content,
          role: BalaurMessageBubbleRole.agent,
        ),
      ),
    );

    final markdown = tester.widget<MarkdownBody>(find.byType(MarkdownBody));

    expect(markdown.data, content);
    expect(markdown.selectable, isTrue);
    expect(find.text('Shopping plan'), findsOneWidget);
    expect(
      find.text('Use milk and notes.md.', findRichText: true),
      findsOneWidget,
    );
  });

  testWidgets('keeps Household Member Markdown as plain text', (tester) async {
    await tester.pumpWidget(
      const _TestApp(
        child: BalaurMessageBubble(
          content: '**Buy milk.**',
          role: BalaurMessageBubbleRole.householdMember,
        ),
      ),
    );

    expect(find.byType(MarkdownBody), findsNothing);
    expect(find.text('**Buy milk.**'), findsOneWidget);
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
