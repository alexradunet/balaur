import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/material.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_reference_screen.stories.g.dart';

const component = ComponentMeta(name: 'Balaur screens', path: 'Screens');
const meta = Meta(BalaurReferenceScreen.new);

final $Chat = _Story(
  name: 'Chat view',
  args: _Args(
    screen: EnumArg(
      BalaurReferenceScreenKind.chat,
      values: BalaurReferenceScreenKind.values,
    ),
  ),
);

final $Tasks = _Story(
  name: 'Tasks page',
  args: _Args(
    screen: EnumArg(
      BalaurReferenceScreenKind.tasks,
      values: BalaurReferenceScreenKind.values,
    ),
  ),
);

final $Memory = _Story(
  name: 'Memory page',
  args: _Args(
    screen: EnumArg(
      BalaurReferenceScreenKind.memory,
      values: BalaurReferenceScreenKind.values,
    ),
  ),
);

final $Life = _Story(
  name: 'Life dashboard',
  args: _Args(
    screen: EnumArg(
      BalaurReferenceScreenKind.life,
      values: BalaurReferenceScreenKind.values,
    ),
  ),
);

final $Profile = _Story(
  name: 'Profile page',
  args: _Args(
    screen: EnumArg(
      BalaurReferenceScreenKind.profile,
      values: BalaurReferenceScreenKind.values,
    ),
  ),
);

enum BalaurReferenceScreenKind { chat, tasks, memory, life, profile }

class BalaurReferenceScreen extends StatelessWidget {
  const BalaurReferenceScreen({super.key, required this.screen});

  final BalaurReferenceScreenKind screen;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final compact = constraints.maxWidth < 700;
        return ColoredBox(
          color: BalaurColors.of(context).background,
          child: Column(
            children: [
              BalaurTopbar(
                links: const ['chat', 'tasks', 'memory', 'life', 'profile'],
                active: screen.name,
                onNavigate: _select,
                onToggleTheme: _noop,
              ),
              Expanded(
                child: SingleChildScrollView(
                  padding: EdgeInsets.symmetric(
                    horizontal: compact ? 16 : 40,
                    vertical: 26,
                  ),
                  child: Center(
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(maxWidth: 980),
                      child: switch (screen) {
                        BalaurReferenceScreenKind.chat =>
                          const _ChatReference(),
                        BalaurReferenceScreenKind.tasks => _TasksReference(
                          compact: compact,
                        ),
                        BalaurReferenceScreenKind.memory => _MemoryReference(
                          compact: compact,
                        ),
                        BalaurReferenceScreenKind.life => _LifeReference(
                          compact: compact,
                        ),
                        BalaurReferenceScreenKind.profile => _ProfileReference(
                          compact: compact,
                        ),
                      },
                    ),
                  ),
                ),
              ),
              Container(
                padding: EdgeInsets.symmetric(
                  horizontal: compact ? 12 : 40,
                  vertical: 12,
                ),
                decoration: BoxDecoration(
                  color: BalaurColors.of(context).background,
                  border: Border(
                    top: BorderSide(
                      color: BalaurColors.of(context).outline,
                      width: 2,
                    ),
                  ),
                ),
                child: Center(
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 980),
                    child: BalaurComposer(
                      onSend: _send,
                      placeholder: _placeholder(screen),
                      sendLabel: screen == BalaurReferenceScreenKind.life
                          ? 'Log'
                          : 'Send',
                      tools: const [
                        BalaurComposerTool(
                          iconName: 'scroll',
                          tooltip: 'Attach a scroll',
                          onPressed: _noop,
                        ),
                        BalaurComposerTool(
                          iconName: 'tome',
                          tooltip: 'Add from memory',
                          onPressed: _noop,
                        ),
                        BalaurComposerTool(
                          iconName: 'lens',
                          tooltip: 'Recall a thread',
                          onPressed: _noop,
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }

  static String _placeholder(BalaurReferenceScreenKind screen) {
    return switch (screen) {
      BalaurReferenceScreenKind.chat => 'Speak; I am listening.',
      BalaurReferenceScreenKind.tasks => 'Add a task to the book…',
      BalaurReferenceScreenKind.memory => 'Tell Balaur what to keep for you…',
      BalaurReferenceScreenKind.life =>
        'Log a measure, or ask about your week…',
      BalaurReferenceScreenKind.profile => 'Tell Balaur what should change…',
    };
  }
}

class _ChatReference extends StatelessWidget {
  const _ChatReference();

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const _PageTitle(
          eyebrow: 'The hearth is lit',
          title: 'What shall we weigh today?',
        ),
        const SizedBox(height: 22),
        const BalaurRecapCard(
          summary:
              'You asked me to keep the garden and your note format in view.',
          when: 'earlier today',
          points: [
            'Garden — tomatoes and peppers, watered at dusk',
            'Notes exported as Markdown',
          ],
        ),
        const SizedBox(height: 24),
        const BalaurMessageBubble(
          content: 'I am here. The hearth is lit. What shall we weigh today?',
          role: BalaurMessageBubbleRole.agent,
        ),
        const SizedBox(height: 20),
        const BalaurMessageBubble(
          content: 'Remind me to water the tomatoes every two days, in the evenings.',
          role: BalaurMessageBubbleRole.householdMember,
        ),
        const SizedBox(height: 20),
        const BalaurToolRow(
          tool: 'task',
          message: 'kept Water the tomatoes · every 2 days · 18:00',
          iconName: 'scroll',
        ),
        const SizedBox(height: 20),
        const BalaurMessageBubble(
          content: 'Every second evening at 18:00, then. It is written.',
          role: BalaurMessageBubbleRole.agent,
        ),
      ],
    );
  }
}

class _TasksReference extends StatelessWidget {
  const _TasksReference({required this.compact});

  final bool compact;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const _PageTitle(
          eyebrow: 'The book of promises',
          title: 'What waits for your hands.',
        ),
        const SizedBox(height: 18),
        BalaurTabs(
          items: const ['list', 'calendar', 'timeline'],
          active: 'list',
          onSelect: _select,
        ),
        const SizedBox(height: 18),
        const BalaurNudgeBanner(
          message: 'The evening comes, and the tomatoes thirst. Will you tend them now?',
          when: '18:00',
          onDone: _noop,
          onSnooze: _noop,
          onTomorrow: _noop,
        ),
        const SizedBox(height: 24),
        _SectionHeading(
          label: 'On the book',
          color: BalaurColors.of(context).gold,
        ),
        const SizedBox(height: 12),
        Wrap(
          spacing: 16,
          runSpacing: 16,
          children: [
            SizedBox(
              width: compact ? double.infinity : 360,
              child: const BalaurTaskCard(
                title: 'Water the tomatoes',
                dueLine: 'due today 18:00',
                recurrence: 'every 2 days',
                onDone: _noop,
                onSnooze: _noop,
                onDrop: _noop,
              ),
            ),
            SizedBox(
              width: compact ? double.infinity : 360,
              child: const BalaurTaskCard(
                title: 'Mend the deer fence',
                dueLine: 'overdue · yesterday',
                overdue: true,
                onDone: _noop,
                onSnooze: _noop,
                onDrop: _noop,
              ),
            ),
          ],
        ),
      ],
    );
  }
}

class _MemoryReference extends StatelessWidget {
  const _MemoryReference({required this.compact});

  final bool compact;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const _PageTitle(
          eyebrow: 'Remembers what matters',
          title: 'What I keep for you.',
        ),
        const SizedBox(height: 18),
        BalaurTabs(
          items: const ['all', 'people', 'preferences', 'projects'],
          active: 'all',
          onSelect: _select,
        ),
        const SizedBox(height: 24),
        _SectionHeading(
          label: 'Awaiting your word',
          color: BalaurColors.of(context).gold,
        ),
        const SizedBox(height: 12),
        SizedBox(
          width: compact ? double.infinity : 420,
          child: const BalaurKnowledgeCard(
            title: 'Prefers Markdown exports',
            kind: BalaurKnowledgeKind.preference,
            status: BalaurKnowledgeStatus.proposed,
            body: 'Asked twice for notes as .md files.',
            whenToUse: 'when exporting notes',
            importance: 3,
            onApprove: _noop,
            onDismiss: _noop,
          ),
        ),
        const SizedBox(height: 26),
        _SectionHeading(
          label: 'Kept in memory',
          color: BalaurColors.of(context).teal,
        ),
        const SizedBox(height: 12),
        Wrap(
          spacing: 16,
          runSpacing: 16,
          children: [
            SizedBox(
              width: compact ? double.infinity : 360,
              child: const BalaurKnowledgeCard(
                title: 'Ana — sister',
                kind: BalaurKnowledgeKind.person,
                body: 'Birthday May 3rd. Lives in Cluj.',
                importance: 4,
                usedCount: 6,
                onArchive: _noop,
              ),
            ),
            SizedBox(
              width: compact ? double.infinity : 360,
              child: const BalaurKnowledgeCard(
                title: 'Garden at dusk',
                kind: BalaurKnowledgeKind.project,
                body: 'Tomatoes and peppers. Water every second evening.',
                importance: 3,
                usedCount: 3,
                onArchive: _noop,
              ),
            ),
          ],
        ),
      ],
    );
  }
}

class _LifeReference extends StatelessWidget {
  const _LifeReference({required this.compact});

  final bool compact;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const _PageTitle(
          eyebrow: 'Tuesday · 14 May',
          title: 'Your week, kept.',
        ),
        const SizedBox(height: 18),
        const BalaurAlert(
          title: 'Running on llama-3.1',
          message: 'A local head. Your measures stay on this box.',
        ),
        const SizedBox(height: 24),
        _SectionHeading(
          label: 'Measures',
          color: BalaurColors.of(context).gold,
        ),
        const SizedBox(height: 12),
        Wrap(
          spacing: 16,
          runSpacing: 16,
          children: [
            SizedBox(
              width: compact ? double.infinity : 280,
              child: const BalaurStatCard(
                iconName: 'gem',
                label: 'Weight',
                value: '81.2',
                unit: 'kg',
                delta: '0.6 this week',
                deltaTone: BalaurStatDeltaTone.down,
                values: [83, 82.6, 82.1, 82.4, 81.9, 81.6, 81.2],
              ),
            ),
            SizedBox(
              width: compact ? double.infinity : 280,
              child: const BalaurStatCard(
                iconName: 'flame',
                label: 'Steps',
                value: '8,210',
                delta: '12% vs avg',
                deltaTone: BalaurStatDeltaTone.up,
                values: [6100, 7200, 5400, 8900, 7600, 9100, 8210],
              ),
            ),
            SizedBox(
              width: compact ? double.infinity : 280,
              child: const BalaurStatCard(
                iconName: 'hourglass',
                label: 'Sleep',
                value: '7.1',
                unit: 'h',
                delta: 'steady',
                values: [6.8, 7.2, 6.5, 7.4, 7.0, 7.3, 7.1],
              ),
            ),
          ],
        ),
        const SizedBox(height: 24),
        _SectionHeading(label: 'Today', color: BalaurColors.of(context).teal),
        const SizedBox(height: 12),
        const BalaurCard(
          child: Column(
            children: [
              BalaurDayEntry(
                time: '06:30',
                title: 'Fed the hens',
                detail: 'daily · streak 12',
              ),
              BalaurDayEntry(
                time: '13:00',
                title: 'Logged weight — 81.2 kg',
                detail: 'life log',
                tone: BalaurDayEntryTone.teal,
              ),
              BalaurDayEntry(
                time: '18:00',
                title: 'Water the tomatoes',
                detail: 'every 2 days · due',
                tone: BalaurDayEntryTone.ember,
                last: true,
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _ProfileReference extends StatelessWidget {
  const _ProfileReference({required this.compact});

  final bool compact;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const _PageTitle(
          eyebrow: 'One companion, many heads',
          title: 'Your place at the hearth.',
        ),
        const SizedBox(height: 24),
        const BalaurCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('YOUR NAME'),
              SizedBox(height: 10),
              BalaurTextField(
                label: 'Household member',
                placeholder: 'Alex',
                hint: 'Balaur uses this name in your private Conversation.',
              ),
            ],
          ),
        ),
        const SizedBox(height: 24),
        _SectionHeading(
          label: 'Your soul',
          color: BalaurColors.of(context).indigo,
        ),
        const SizedBox(height: 12),
        Wrap(
          spacing: 10,
          runSpacing: 10,
          children: [
            for (var index = 1; index <= (compact ? 8 : 16); index++)
              BalaurSurface(
                material: BalaurSurfaceMaterial.wood,
                borderColor: index == 5
                    ? BalaurColors.of(context).goldDeep
                    : BalaurColors.of(context).outline,
                padding: const EdgeInsets.all(4),
                child: BalaurAvatar(
                  image: BalaurAssets.soulAvatar(index),
                  kind: BalaurAvatarKind.soul,
                  size: compact ? 54 : 64,
                  mirrored: true,
                  semanticLabel: 'Soul portrait $index',
                ),
              ),
          ],
        ),
      ],
    );
  }
}

class _PageTitle extends StatelessWidget {
  const _PageTitle({required this.eyebrow, required this.title});

  final String eyebrow;
  final String title;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          eyebrow.toUpperCase(),
          style: Theme.of(context).textTheme.labelMedium
              ?.copyWith(color: colors.gold, letterSpacing: 1.2),
        ),
        const SizedBox(height: 7),
        Text(
          title,
          style: Theme.of(context).textTheme.displayMedium
              ?.copyWith(color: colors.foregroundStrong),
        ),
      ],
    );
  }
}

class _SectionHeading extends StatelessWidget {
  const _SectionHeading({required this.label, required this.color});

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Text(
          label.toUpperCase(),
          style: Theme.of(context).textTheme.labelMedium
              ?.copyWith(color: color, letterSpacing: 1),
        ),
        const SizedBox(width: 10),
        const Expanded(child: BalaurStitch()),
      ],
    );
  }
}

void _noop() {}
void _send(String _) {}
void _select(String _) {}
