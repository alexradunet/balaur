import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_breadcrumb.stories.g.dart';

const component = ComponentMeta(
  name: 'Breadcrumb',
  path: 'Design system/Navigation',
);
const meta = Meta(BalaurBreadcrumb.new);

final $Path = _Story(
  args: _Args.fixed(
    items: [
      BalaurBreadcrumbItem(label: 'Memory', onPressed: _noop),
      BalaurBreadcrumbItem(label: 'People', onPressed: _noop),
      BalaurBreadcrumbItem(label: 'Ana — sister'),
    ],
  ),
);

void _noop() {}
