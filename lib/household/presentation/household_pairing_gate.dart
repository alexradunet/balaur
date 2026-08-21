import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/household_pairing_view.dart';
import 'package:balaur/household/presentation/household_pairing_view_model.dart';
import 'package:flutter/material.dart';

class HouseholdPairingGate extends StatefulWidget {
  const HouseholdPairingGate({
    super.key,
    required this.gateway,
    required this.pairedChild,
    this.onScanInvitation,
  });

  final HouseholdGateway gateway;
  final Widget pairedChild;
  final HouseholdInvitationScanAction? onScanInvitation;

  @override
  State<HouseholdPairingGate> createState() => _HouseholdPairingGateState();
}

class _HouseholdPairingGateState extends State<HouseholdPairingGate> {
  late HouseholdPairingViewModel _viewModel;
  bool _entered = false;

  @override
  void initState() {
    super.initState();
    _viewModel = HouseholdPairingViewModel(gateway: widget.gateway)
      ..initialize();
  }

  @override
  void didUpdateWidget(covariant HouseholdPairingGate oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.gateway != widget.gateway) {
      _viewModel.dispose();
      _viewModel = HouseholdPairingViewModel(gateway: widget.gateway)
        ..initialize();
      _entered = false;
    }
  }

  @override
  void dispose() {
    _viewModel.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final session = _viewModel.session;
    if (_entered && session != null) {
      return HouseholdSessionScope(
        session: session,
        onSignOut: _signOut,
        child: widget.pairedChild,
      );
    }
    return Navigator(
      onGenerateRoute: (_) => MaterialPageRoute<void>(
        builder: (_) => ListenableBuilder(
          listenable: _viewModel,
          builder: (context, _) {
            final currentSession = _viewModel.session;
            return HouseholdPairingView(
              status: _viewModel.status,
              session: currentSession,
              errorMessage: _viewModel.errorMessage,
              onPair: _viewModel.pair,
              onRedeemInvitation: _viewModel.redeemInvitation,
              onSignOut: _signOut,
              onScanInvitation: widget.onScanInvitation,
              onEnter: currentSession == null
                  ? null
                  : () => setState(() => _entered = true),
            );
          },
        ),
      ),
    );
  }

  Future<void> _signOut() async {
    if (mounted) {
      setState(() => _entered = false);
    }
    await _viewModel.signOut();
  }
}

class HouseholdSessionScope extends InheritedWidget {
  const HouseholdSessionScope({
    super.key,
    required this.session,
    required this.onSignOut,
    required super.child,
  });

  final HouseholdSession session;
  final Future<void> Function() onSignOut;

  static HouseholdSessionScope of(BuildContext context) {
    final scope = context
        .dependOnInheritedWidgetOfExactType<HouseholdSessionScope>();
    assert(scope != null, 'No Household session is available.');
    return scope!;
  }

  @override
  bool updateShouldNotify(HouseholdSessionScope oldWidget) {
    return oldWidget.session != session || oldWidget.onSignOut != onSignOut;
  }
}
