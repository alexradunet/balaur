import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/household_account_screen.dart';
import 'package:balaur/household/presentation/household_pairing_view_model.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

typedef HouseholdPairAction = Future<void> Function({
  required HouseholdServerAddress serverAddress,
  required String email,
  required String password,
});

class HouseholdPairingView extends StatefulWidget {
  const HouseholdPairingView({
    super.key,
    required this.status,
    required this.onPair,
    required this.onSignOut,
    this.session,
    this.errorMessage,
    this.onEnter,
  });

  final HouseholdPairingStatus status;
  final HouseholdSession? session;
  final String? errorMessage;
  final HouseholdPairAction onPair;
  final Future<void> Function() onSignOut;
  final VoidCallback? onEnter;

  @override
  State<HouseholdPairingView> createState() => _HouseholdPairingViewState();
}

class _HouseholdPairingViewState extends State<HouseholdPairingView> {
  final _formKey = GlobalKey<FormState>();
  final _serverAddressController = TextEditingController();
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _hidePassword = true;

  @override
  void didUpdateWidget(covariant HouseholdPairingView oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.status != HouseholdPairingStatus.paired &&
        widget.status == HouseholdPairingStatus.paired) {
      _passwordController.clear();
    }
  }

  @override
  void dispose() {
    _serverAddressController.dispose();
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final session = widget.session;
    final child = switch (widget.status) {
      HouseholdPairingStatus.connecting => const _ConnectingView(),
      HouseholdPairingStatus.paired when session != null =>
        HouseholdAccountView(
          session: session,
          onSignOut: widget.onSignOut,
          onEnter: widget.onEnter,
        ),
      HouseholdPairingStatus.unpaired ||
      HouseholdPairingStatus.failed => _buildPairingForm(context),
      HouseholdPairingStatus.paired => _buildPairingForm(context),
    };
    return Material(
      color: Theme.of(context).scaffoldBackgroundColor,
      child: child,
    );
  }

  Widget _buildPairingForm(BuildContext context) {
    return SafeArea(
      child: LayoutBuilder(
        builder: (context, constraints) {
          final horizontalPadding = constraints.maxWidth < 700 ? 16.0 : 40.0;
          return SingleChildScrollView(
            padding: EdgeInsets.symmetric(
              horizontal: horizontalPadding,
              vertical: 28,
            ),
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 640),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text(
                      'Pair this device',
                      style: Theme.of(context).textTheme.headlineMedium,
                    ),
                    const SizedBox(height: 8),
                    Text(
                      'Connect this device to your Household Server.',
                      style: Theme.of(context).textTheme.bodyLarge,
                    ),
                    const SizedBox(height: 24),
                    if (widget.errorMessage case final message?) ...[
                      BalaurAlert(
                        title: 'Pairing failed',
                        message: message,
                        tone: BalaurAlertTone.danger,
                      ),
                      const SizedBox(height: 16),
                    ],
                    BalaurSurface(
                      ornate: true,
                      padding: const EdgeInsets.all(24),
                      child: AutofillGroup(
                        child: Form(
                          key: _formKey,
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.stretch,
                            children: [
                              TextFormField(
                                key: const Key('household-server-address'),
                                controller: _serverAddressController,
                                decoration: const InputDecoration(
                                  labelText: 'Household Server address',
                                  hintText: 'https://household.example.com',
                                  helperText: 'Use the stable HTTPS address from the Household operator.',
                                ),
                                keyboardType: TextInputType.url,
                                textInputAction: TextInputAction.next,
                                autofillHints: const [AutofillHints.url],
                                validator: _validateServerAddress,
                              ),
                              const SizedBox(height: 16),
                              TextFormField(
                                key: const Key('household-email'),
                                controller: _emailController,
                                decoration: const InputDecoration(
                                  labelText: 'Email address',
                                ),
                                keyboardType: TextInputType.emailAddress,
                                textInputAction: TextInputAction.next,
                                autofillHints: const [AutofillHints.email],
                                validator: _validateEmail,
                              ),
                              const SizedBox(height: 16),
                              TextFormField(
                                key: const Key('household-password'),
                                controller: _passwordController,
                                decoration: InputDecoration(
                                  labelText: 'Password',
                                  suffixIcon: IconButton(
                                    onPressed: () => setState(() {
                                      _hidePassword = !_hidePassword;
                                    }),
                                    tooltip: _hidePassword
                                        ? 'Show password'
                                        : 'Hide password',
                                    icon: Icon(
                                      _hidePassword
                                          ? Icons.visibility
                                          : Icons.visibility_off,
                                    ),
                                  ),
                                ),
                                obscureText: _hidePassword,
                                textInputAction: TextInputAction.done,
                                autofillHints: const [AutofillHints.password],
                                onFieldSubmitted: (_) => _pair(),
                                validator: _required,
                              ),
                              const SizedBox(height: 16),
                              const Text(
                                'Balaur stores the server address and member session in secure storage on this device.',
                              ),
                              const SizedBox(height: 24),
                              Align(
                                alignment: Alignment.centerRight,
                                child: BalaurButton(
                                  key: const Key('pair-household'),
                                  label: 'Pair this device',
                                  onPressed: _pair,
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          );
        },
      ),
    );
  }

  String? _validateServerAddress(String? value) {
    if (_required(value) case final error?) {
      return error;
    }
    try {
      HouseholdServerAddress.parse(value!);
      return null;
    } on FormatException {
      return 'Enter a stable HTTPS address.';
    }
  }

  String? _validateEmail(String? value) {
    if (_required(value) case final error?) {
      return error;
    }
    if (!RegExp(r'^[^\s@]+@[^\s@]+\.[^\s@]+$').hasMatch(value!.trim())) {
      return 'Enter a valid email address.';
    }
    return null;
  }

  String? _required(String? value) {
    return value == null || value.trim().isEmpty ? 'Enter a value.' : null;
  }

  Future<void> _pair() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }
    TextInput.finishAutofillContext(shouldSave: false);
    await widget.onPair(
      serverAddress: HouseholdServerAddress.parse(
        _serverAddressController.text,
      ),
      email: _emailController.text.trim(),
      password: _passwordController.text,
    );
  }
}

class _ConnectingView extends StatelessWidget {
  const _ConnectingView();

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Center(
        child: Semantics(
          liveRegion: true,
          label: 'Connecting to the Household Server',
          child: BalaurSurface(
            ornate: true,
            padding: const EdgeInsets.all(28),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const SizedBox.square(
                  dimension: 32,
                  child: CircularProgressIndicator(strokeWidth: 3),
                ),
                const SizedBox(height: 16),
                Text(
                  'Connecting to the Household Server…',
                  style: Theme.of(context).textTheme.titleMedium,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
