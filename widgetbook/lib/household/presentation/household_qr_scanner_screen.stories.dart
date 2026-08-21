import 'package:balaur/household/presentation/household_qr_scanner_screen.dart';
import 'package:flutter/material.dart';
import 'package:widgetbook/widgetbook.dart';

part 'household_qr_scanner_screen.stories.g.dart';

const component = ComponentMeta(
  name: 'Household invitation scanner',
  path: 'Screens/Household',
);
const meta = Meta(HouseholdQrScannerScreen.new);

final $Active = _Story(
  args: _Args.fixed(
    state: HouseholdQrScannerState.active,
    cameraPreview: const ColoredBox(
      color: Colors.black,
      child: Center(
        child: Icon(Icons.camera_alt, color: Colors.white, size: 72),
      ),
    ),
  ),
);

final $CameraError = _Story(
  args: _Args.fixed(state: HouseholdQrScannerState.cameraError),
);

final $Unavailable = _Story(
  args: _Args.fixed(state: HouseholdQrScannerState.unavailable),
);
