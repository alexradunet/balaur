import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/domain/household_archive.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/household_archive_view_model.dart';
import 'package:flutter/material.dart';

class HouseholdArchiveScreen extends StatefulWidget {
  const HouseholdArchiveScreen({
    super.key,
    required this.gateway,
    required this.session,
    required this.onBack,
    required this.saveArchive,
  });

  final HouseholdGateway gateway;
  final HouseholdSession session;
  final VoidCallback onBack;
  final Future<bool> Function(HouseholdArchive archive) saveArchive;

  @override
  State<HouseholdArchiveScreen> createState() => _HouseholdArchiveScreenState();
}

class _HouseholdArchiveScreenState extends State<HouseholdArchiveScreen> {
  final _passwordController = TextEditingController();
  late HouseholdArchiveViewModel _viewModel;
  bool _includeCalendarSnapshot = false;

  bool get _isAdministrator =>
      widget.session.member.role == HouseholdMemberRole.administrator;

  @override
  void initState() {
    super.initState();
    _createViewModel();
  }

  @override
  void didUpdateWidget(covariant HouseholdArchiveScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.gateway != widget.gateway ||
        oldWidget.saveArchive != widget.saveArchive) {
      _viewModel.dispose();
      _createViewModel();
    }
  }

  @override
  void dispose() {
    _passwordController.dispose();
    _viewModel.dispose();
    super.dispose();
  }

  void _createViewModel() {
    _viewModel = HouseholdArchiveViewModel(widget.gateway, widget.saveArchive);
  }

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Material(
      color: Theme.of(context).scaffoldBackgroundColor,
      child: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 28),
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 720),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Align(
                    alignment: Alignment.centerLeft,
                    child: BalaurButton(
                      label: 'Back to Household',
                      variant: BalaurButtonVariant.ghost,
                      leading: const Icon(Icons.arrow_back, size: 18),
                      onPressed: widget.onBack,
                    ),
                  ),
                  const SizedBox(height: 20),
                  Text(
                    'Household Archive',
                    style: Theme.of(context).textTheme.headlineMedium
                        ?.copyWith(color: colors.foregroundStrong),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    'Export portable Household State without active credentials or private Conversations.',
                    style: Theme.of(context).textTheme.bodyLarge
                        ?.copyWith(color: colors.foreground),
                  ),
                  const SizedBox(height: 24),
                  if (!_isAdministrator)
                    const BalaurAlert(
                      title: 'Household Administrator access is required',
                      message: 'Only a Household Administrator can export a Household Archive.',
                      tone: BalaurAlertTone.danger,
                    )
                  else
                    ListenableBuilder(
                      listenable: _viewModel,
                      builder: (context, _) => _buildExport(context),
                    ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildExport(BuildContext context) {
    final colors = BalaurColors.of(context);
    final status = _viewModel.status;
    final failed = status == HouseholdArchiveStatus.failed;
    final success = status == HouseholdArchiveStatus.success;
    return BalaurSurface(
      ornate: true,
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            'Confirm your password',
            style: Theme.of(context).textTheme.titleLarge
                ?.copyWith(color: colors.ink),
          ),
          const SizedBox(height: 8),
          Text(
            'Balaur sends the password only to your Household Server for this export.',
            style: TextStyle(color: colors.ink),
          ),
          const SizedBox(height: 16),
          TextField(
            key: const Key('household-archive-password'),
            controller: _passwordController,
            obscureText: true,
            enabled: !_viewModel.isBusy,
            autofillHints: const [AutofillHints.password],
            decoration: const InputDecoration(
              labelText: 'Household Administrator password',
            ),
            onSubmitted: (_) => _export(),
          ),
          const SizedBox(height: 16),
          SwitchListTile.adaptive(
            key: const Key('include-calendar-snapshot'),
            contentPadding: EdgeInsets.zero,
            title: const Text('Include an iCalendar snapshot'),
            subtitle: const Text(
              'The snapshot contains current Calendar Entries but no authorization credential.',
            ),
            value: _includeCalendarSnapshot,
            onChanged: _viewModel.isBusy
                ? null
                : (value) {
                    setState(() {
                      _includeCalendarSnapshot = value;
                    });
                  },
          ),
          if (_viewModel.message case final message?) ...[
            const SizedBox(height: 16),
            BalaurAlert(
              title: success
                  ? 'Export complete'
                  : failed
                  ? 'Export failed'
                  : 'Export status',
              message: _viewModel.fileName == null
                  ? message
                  : '$message ${_viewModel.fileName}',
              tone: success
                  ? BalaurAlertTone.info
                  : failed
                  ? BalaurAlertTone.danger
                  : BalaurAlertTone.warning,
            ),
          ],
          const SizedBox(height: 20),
          Align(
            alignment: Alignment.centerRight,
            child: BalaurButton(
              key: const Key('export-household-archive'),
              label: switch (status) {
                HouseholdArchiveStatus.exporting => 'Creating archive',
                HouseholdArchiveStatus.saving => 'Saving archive',
                _ => 'Export archive',
              },
              onPressed: _viewModel.isBusy ? null : _export,
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _export() async {
    await _viewModel.export(
      password: _passwordController.text,
      includeCalendarSnapshot: _includeCalendarSnapshot,
    );
    if (_viewModel.status == HouseholdArchiveStatus.success) {
      _passwordController.clear();
    }
  }
}
