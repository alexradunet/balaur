import 'package:balaur/design_system/design_system.dart';
import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('uses the root asset key in the application bundle', () async {
    final key = await BalaurAssets.soulAvatar(5)
        .obtainKey(ImageConfiguration.empty);

    expect(key.name, 'assets/design_system/avatars/soul-05.png');
  });
}
