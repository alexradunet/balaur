import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/widgets.dart';
import 'package:widgetbook/widgetbook.dart';

part 'balaur_pagination.stories.g.dart';

const component = ComponentMeta(
  name: 'Pagination',
  path: 'Design system/Navigation',
);
const meta = Meta(BalaurPagination.new);

final $FirstPage = _Story(
  args: _Args(
    page: IntArg(1),
    total: IntArg(8),
    onChange: Arg.fixed((int _) {}),
  ),
);

final $MiddlePage = _Story(
  args: _Args(
    page: IntArg(5),
    total: IntArg(8),
    onChange: Arg.fixed((int _) {}),
  ),
);
