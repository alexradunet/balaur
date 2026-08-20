import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_knowledge_card.stories.g.dart';

const component = ComponentMeta(
  name: 'Knowledge card',
  path: 'Design system/Knowledge',
);
const meta = Meta(BalaurKnowledgeCard.new);

final $Proposed = _Story(
  args: _Args(
    title: StringArg('Prefers Markdown exports'),
    kind: EnumArg(
      BalaurKnowledgeKind.preference,
      values: BalaurKnowledgeKind.values,
    ),
    status: EnumArg(
      BalaurKnowledgeStatus.proposed,
      values: BalaurKnowledgeStatus.values,
    ),
    body: NullableStringArg('Asked twice for notes as .md files.'),
    whenToUse: NullableStringArg('when exporting notes'),
    importance: NullableIntArg(3),
    onApprove: Arg.fixed(() {}),
    onDismiss: Arg.fixed(() {}),
  ),
);

final $Active = _Story(
  args: _Args(
    title: StringArg('Ana — sister'),
    kind: EnumArg(
      BalaurKnowledgeKind.person,
      values: BalaurKnowledgeKind.values,
    ),
    status: EnumArg(
      BalaurKnowledgeStatus.active,
      values: BalaurKnowledgeStatus.values,
    ),
    body: NullableStringArg('Birthday May 3rd. Lives in Cluj.'),
    importance: NullableIntArg(4),
    usedCount: NullableIntArg(6),
    onArchive: Arg.fixed(() {}),
  ),
);

final $Archived = _Story(
  args: _Args(
    title: StringArg('Old apartment Wi-Fi password'),
    kind: EnumArg(BalaurKnowledgeKind.fact, values: BalaurKnowledgeKind.values),
    status: EnumArg(
      BalaurKnowledgeStatus.archived,
      values: BalaurKnowledgeStatus.values,
    ),
    importance: NullableIntArg(1),
    onRestore: Arg.fixed(() {}),
  ),
);
