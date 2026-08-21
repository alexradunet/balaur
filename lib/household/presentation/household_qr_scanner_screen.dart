import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/domain/household_invitation.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

enum HouseholdQrScannerState { automatic, active, unavailable, cameraError }

class HouseholdQrScannerScreen extends StatefulWidget {
  const HouseholdQrScannerScreen({
    super.key,
    this.state = HouseholdQrScannerState.automatic,
    this.cameraPreview,
  });

  final HouseholdQrScannerState state;
  final Widget? cameraPreview;

  static bool get isSupported {
    if (kIsWeb) {
      return false;
    }
    return defaultTargetPlatform == TargetPlatform.android ||
        defaultTargetPlatform == TargetPlatform.iOS;
  }

  @override
  State<HouseholdQrScannerScreen> createState() =>
      _HouseholdQrScannerScreenState();
}

class _HouseholdQrScannerScreenState extends State<HouseholdQrScannerScreen> {
  bool _finished = false;

  @override
  Widget build(BuildContext context) {
    final state = switch (widget.state) {
      HouseholdQrScannerState.automatic =>
        HouseholdQrScannerScreen.isSupported
            ? HouseholdQrScannerState.active
            : HouseholdQrScannerState.unavailable,
      final state => state,
    };
    final body = switch (state) {
      HouseholdQrScannerState.active => _buildScanner(context),
      HouseholdQrScannerState.cameraError => _buildCameraError(context),
      HouseholdQrScannerState.unavailable => _buildUnavailable(context),
      HouseholdQrScannerState.automatic => const SizedBox.shrink(),
    };
    return Scaffold(
      appBar: AppBar(title: const Text('Scan a Household Invitation')),
      body: SafeArea(child: body),
    );
  }

  Widget _buildScanner(BuildContext context) {
    return Stack(
      fit: StackFit.expand,
      children: [
        widget.cameraPreview ??
            MobileScanner(
              onDetect: (capture) => _useCapture(context, capture),
              errorBuilder: (context, _) => _buildCameraError(context),
            ),
        IgnorePointer(
          child: Center(
            child: Semantics(
              label: 'Place the Household Invitation code in this frame.',
              child: Container(
                width: 260,
                height: 260,
                decoration: BoxDecoration(
                  border: Border.all(color: Colors.white, width: 4),
                  borderRadius: BorderRadius.circular(20),
                ),
              ),
            ),
          ),
        ),
        const Positioned(
          left: 24,
          right: 24,
          bottom: 32,
          child: Text(
            'Place the complete invitation code inside the frame.',
            textAlign: TextAlign.center,
            style: TextStyle(
              color: Colors.white,
              fontSize: 18,
              fontWeight: FontWeight.w600,
              shadows: [Shadow(blurRadius: 6)],
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildUnavailable(BuildContext context) {
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 520),
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: BalaurAlert(
            title: 'Code scanning is not available',
            message: 'Use an Android phone or iPhone to scan. You can enter the Household Invitation manually on this device.',
            tone: BalaurAlertTone.info,
          ),
        ),
      ),
    );
  }

  Widget _buildCameraError(BuildContext context) {
    return ColoredBox(
      color: Colors.black,
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 520),
          child: const Padding(
            padding: EdgeInsets.all(24),
            child: BalaurAlert(
              title: 'The camera is not available',
              message: 'Give Balaur camera access. You can also enter the Household Invitation manually.',
              tone: BalaurAlertTone.danger,
            ),
          ),
        ),
      ),
    );
  }

  void _useCapture(BuildContext context, BarcodeCapture capture) {
    if (_finished) {
      return;
    }
    for (final barcode in capture.barcodes) {
      final value = barcode.rawValue;
      if (value == null || value.trim().isEmpty) {
        continue;
      }
      try {
        HouseholdInvitationPayload.parseQrValue(value);
        _finished = true;
        Navigator.of(context).pop(value);
        return;
      } on FormatException {
        continue;
      }
    }
  }
}
