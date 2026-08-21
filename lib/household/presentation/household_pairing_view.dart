import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/domain/household_invitation.dart';
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

typedef HouseholdRedeemInvitationAction = Future<void> Function({
  required HouseholdInvitationPayload invitation,
  required String displayName,
  required String email,
  required String password,
});

typedef HouseholdInvitationScanAction = Future<String?> Function(
  BuildContext context,
);

enum HouseholdPairingMode { credentials, invitation }

class HouseholdPairingView extends StatefulWidget {
  const HouseholdPairingView({
    super.key,
    required this.status,
    required this.onPair,
    required this.onRedeemInvitation,
    required this.onSignOut,
    this.session,
    this.errorMessage,
    this.onEnter,
    this.onScanInvitation,
    this.initialMode = HouseholdPairingMode.credentials,
    this.allowInsecureLoopback = false,
  });

  final HouseholdPairingStatus status;
  final HouseholdSession? session;
  final String? errorMessage;
  final HouseholdPairAction onPair;
  final HouseholdRedeemInvitationAction onRedeemInvitation;
  final Future<void> Function() onSignOut;
  final VoidCallback? onEnter;
  final HouseholdInvitationScanAction? onScanInvitation;
  final HouseholdPairingMode initialMode;
  final bool allowInsecureLoopback;

  @override
  State<HouseholdPairingView> createState() => _HouseholdPairingViewState();
}

class _HouseholdPairingViewState extends State<HouseholdPairingView> {
  final _credentialsFormKey = GlobalKey<FormState>();
  final _invitationFormKey = GlobalKey<FormState>();
  final _serverAddressController = TextEditingController();
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _invitationServerAddressController = TextEditingController();
  final _invitationController = TextEditingController();
  final _displayNameController = TextEditingController();
  final _invitationEmailController = TextEditingController();
  final _invitationPasswordController = TextEditingController();
  final _passwordConfirmController = TextEditingController();
  late HouseholdPairingMode _mode;
  bool _hidePassword = true;
  String? _scanError;

  @override
  void initState() {
    super.initState();
    _mode = widget.initialMode;
  }

  @override
  void didUpdateWidget(covariant HouseholdPairingView oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.status != HouseholdPairingStatus.paired &&
        widget.status == HouseholdPairingStatus.paired) {
      _passwordController.clear();
      _invitationPasswordController.clear();
      _passwordConfirmController.clear();
    }
  }

  @override
  void dispose() {
    _serverAddressController.dispose();
    _emailController.dispose();
    _passwordController.dispose();
    _invitationServerAddressController.dispose();
    _invitationController.dispose();
    _displayNameController.dispose();
    _invitationEmailController.dispose();
    _invitationPasswordController.dispose();
    _passwordConfirmController.dispose();
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
    final colors = BalaurColors.of(context);
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
                      style: Theme.of(context).textTheme.headlineMedium
                          ?.copyWith(color: colors.foregroundStrong),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      _mode == HouseholdPairingMode.credentials
                          ? 'Connect with existing Household Member credentials.'
                          : 'Use a one-time Household Invitation to join.',
                      style: Theme.of(context).textTheme.bodyLarge
                          ?.copyWith(color: colors.foreground),
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
                    if (_scanError case final message?) ...[
                      BalaurAlert(
                        title: 'The invitation code is not valid',
                        message: message,
                        tone: BalaurAlertTone.danger,
                      ),
                      const SizedBox(height: 16),
                    ],
                    if (_mode == HouseholdPairingMode.credentials)
                      _buildCredentialsForm(context)
                    else
                      _buildInvitationForm(context),
                  ],
                ),
              ),
            ),
          );
        },
      ),
    );
  }

  Widget _buildCredentialsForm(BuildContext context) {
    return BalaurSurface(
      ornate: true,
      padding: const EdgeInsets.all(24),
      child: AutofillGroup(
        child: Form(
          key: _credentialsFormKey,
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
                decoration: const InputDecoration(labelText: 'Email address'),
                keyboardType: TextInputType.emailAddress,
                textInputAction: TextInputAction.next,
                autofillHints: const [AutofillHints.email],
                validator: _validateEmail,
              ),
              const SizedBox(height: 16),
              TextFormField(
                key: const Key('household-password'),
                controller: _passwordController,
                decoration: _passwordDecoration('Password'),
                obscureText: _hidePassword,
                textInputAction: TextInputAction.done,
                autofillHints: const [AutofillHints.password],
                onFieldSubmitted: (_) => _pair(),
                validator: _required,
              ),
              const SizedBox(height: 16),
              Text(
                'Balaur stores the server address and member session in secure storage on this device.',
                style: TextStyle(color: BalaurColors.of(context).ink),
              ),
              const SizedBox(height: 24),
              Wrap(
                alignment: WrapAlignment.spaceBetween,
                crossAxisAlignment: WrapCrossAlignment.center,
                spacing: 12,
                runSpacing: 12,
                children: [
                  BalaurButton(
                    key: const Key('use-household-invitation'),
                    label: 'Use an invitation',
                    variant: BalaurButtonVariant.ghost,
                    onPressed: () => setState(() {
                      _mode = HouseholdPairingMode.invitation;
                      _scanError = null;
                    }),
                  ),
                  BalaurButton(
                    key: const Key('pair-household'),
                    label: 'Pair this device',
                    onPressed: _pair,
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildInvitationForm(BuildContext context) {
    return BalaurSurface(
      ornate: true,
      padding: const EdgeInsets.all(24),
      child: AutofillGroup(
        child: Form(
          key: _invitationFormKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (widget.onScanInvitation != null) ...[
                BalaurButton(
                  key: const Key('scan-household-invitation'),
                  label: 'Scan invitation code',
                  variant: BalaurButtonVariant.wood,
                  leading: const Icon(Icons.qr_code_scanner, size: 20),
                  onPressed: _scanInvitation,
                ),
                const SizedBox(height: 20),
                Row(
                  children: [
                    Expanded(
                      child: Divider(color: BalaurColors.of(context).outline),
                    ),
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 12),
                      child: Text(
                        'OR ENTER MANUALLY',
                        style: TextStyle(color: BalaurColors.of(context).ink),
                      ),
                    ),
                    Expanded(
                      child: Divider(color: BalaurColors.of(context).outline),
                    ),
                  ],
                ),
                const SizedBox(height: 20),
              ],
              TextFormField(
                key: const Key('invitation-household-server-address'),
                controller: _invitationServerAddressController,
                decoration: const InputDecoration(
                  labelText: 'Household Server address',
                  hintText: 'https://household.example.com',
                ),
                keyboardType: TextInputType.url,
                textInputAction: TextInputAction.next,
                autofillHints: const [AutofillHints.url],
                validator: _validateServerAddress,
              ),
              const SizedBox(height: 16),
              TextFormField(
                key: const Key('household-invitation-value'),
                controller: _invitationController,
                decoration: const InputDecoration(
                  labelText: 'Household Invitation value',
                ),
                autocorrect: false,
                enableSuggestions: false,
                textInputAction: TextInputAction.next,
                validator: _validateInvitation,
              ),
              const SizedBox(height: 16),
              TextFormField(
                key: const Key('invited-household-display-name'),
                controller: _displayNameController,
                decoration: const InputDecoration(labelText: 'Your name'),
                textCapitalization: TextCapitalization.words,
                textInputAction: TextInputAction.next,
                autofillHints: const [AutofillHints.name],
                validator: _required,
              ),
              const SizedBox(height: 16),
              TextFormField(
                key: const Key('invited-household-email'),
                controller: _invitationEmailController,
                decoration: const InputDecoration(labelText: 'Email address'),
                keyboardType: TextInputType.emailAddress,
                textInputAction: TextInputAction.next,
                autofillHints: const [AutofillHints.newUsername],
                validator: _validateEmail,
              ),
              const SizedBox(height: 16),
              TextFormField(
                key: const Key('invited-household-password'),
                controller: _invitationPasswordController,
                decoration: _passwordDecoration('Create a password'),
                obscureText: _hidePassword,
                textInputAction: TextInputAction.next,
                autofillHints: const [AutofillHints.newPassword],
                validator: _validateNewPassword,
              ),
              const SizedBox(height: 16),
              TextFormField(
                key: const Key('invited-household-password-confirm'),
                controller: _passwordConfirmController,
                decoration: const InputDecoration(
                  labelText: 'Confirm password',
                ),
                obscureText: _hidePassword,
                textInputAction: TextInputAction.done,
                autofillHints: const [AutofillHints.newPassword],
                onFieldSubmitted: (_) => _redeemInvitation(),
                validator: _validatePasswordConfirmation,
              ),
              const SizedBox(height: 16),
              Text(
                'The Household Invitation works one time. Balaur stores the new member session in secure storage.',
                style: TextStyle(color: BalaurColors.of(context).ink),
              ),
              const SizedBox(height: 24),
              Wrap(
                alignment: WrapAlignment.spaceBetween,
                crossAxisAlignment: WrapCrossAlignment.center,
                spacing: 12,
                runSpacing: 12,
                children: [
                  BalaurButton(
                    key: const Key('use-household-credentials'),
                    label: 'Use existing credentials',
                    variant: BalaurButtonVariant.ghost,
                    onPressed: () => setState(() {
                      _mode = HouseholdPairingMode.credentials;
                      _scanError = null;
                    }),
                  ),
                  BalaurButton(
                    key: const Key('redeem-household-invitation'),
                    label: 'Join Household',
                    onPressed: _redeemInvitation,
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  InputDecoration _passwordDecoration(String label) {
    return InputDecoration(
      labelText: label,
      suffixIcon: IconButton(
        onPressed: () => setState(() => _hidePassword = !_hidePassword),
        tooltip: _hidePassword ? 'Show password' : 'Hide password',
        icon: Icon(_hidePassword ? Icons.visibility : Icons.visibility_off),
      ),
    );
  }

  String? _validateServerAddress(String? value) {
    if (_required(value) case final error?) {
      return error;
    }
    try {
      _parseServerAddress(value!);
      return null;
    } on FormatException {
      return widget.allowInsecureLoopback
          ? 'Use HTTPS or an HTTP loopback address.'
          : 'Enter a stable HTTPS address.';
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

  String? _validateInvitation(String? value) {
    if (_required(value) case final error?) {
      return error;
    }
    if (!HouseholdInvitationPayload.isValidValue(value!.trim())) {
      return 'Enter the complete Household Invitation value.';
    }
    return null;
  }

  String? _validateNewPassword(String? value) {
    if (value == null || value.length < 8 || value.length > 71) {
      return 'Use a password with 8 to 71 characters.';
    }
    return null;
  }

  String? _validatePasswordConfirmation(String? value) {
    if (value != _invitationPasswordController.text) {
      return 'The passwords do not match.';
    }
    return null;
  }

  String? _required(String? value) {
    return value == null || value.trim().isEmpty ? 'Enter a value.' : null;
  }

  Future<void> _pair() async {
    if (!_credentialsFormKey.currentState!.validate()) {
      return;
    }
    TextInput.finishAutofillContext(shouldSave: false);
    await widget.onPair(
      serverAddress: _parseServerAddress(_serverAddressController.text),
      email: _emailController.text.trim(),
      password: _passwordController.text,
    );
  }

  Future<void> _scanInvitation() async {
    final rawValue = await widget.onScanInvitation?.call(context);
    if (!mounted || rawValue == null) {
      return;
    }
    try {
      final payload = HouseholdInvitationPayload.parseQrValue(
        rawValue,
        allowInsecureLoopback: widget.allowInsecureLoopback,
      );
      setState(() {
        _invitationServerAddressController.text = payload.serverAddress.value;
        _invitationController.text = payload.value;
        _scanError = null;
      });
    } on FormatException {
      setState(() {
        _scanError = 'Scan the code shown by a Household Administrator. You can also enter the details manually.';
      });
    }
  }

  HouseholdServerAddress _parseServerAddress(String value) {
    return HouseholdServerAddress.parse(
      value,
      allowInsecureLoopback: widget.allowInsecureLoopback,
    );
  }

  Future<void> _redeemInvitation() async {
    if (!_invitationFormKey.currentState!.validate()) {
      return;
    }
    TextInput.finishAutofillContext(shouldSave: false);
    await widget.onRedeemInvitation(
      invitation: HouseholdInvitationPayload(
        serverAddress: _parseServerAddress(
          _invitationServerAddressController.text,
        ),
        value: _invitationController.text.trim(),
      ),
      displayName: _displayNameController.text.trim(),
      email: _invitationEmailController.text.trim(),
      password: _invitationPasswordController.text,
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
