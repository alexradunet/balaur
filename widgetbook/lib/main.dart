import 'package:flutter_driver/driver_extension.dart';
import 'package:widgetbook/widgetbook.dart';

import 'widgetbook.config.dart';

const _enableFlutterDriver = bool.fromEnvironment('ENABLE_FLUTTER_DRIVER');

void main() {
  if (_enableFlutterDriver) {
    enableFlutterDriverExtension();
  }
  runWidgetbook(config);
}
