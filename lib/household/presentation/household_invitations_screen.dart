import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/domain/household_invitation.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/household_invitations_view_model.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:qr_flutter/qr_flutter.dart';

class HouseholdInvitationsScreen extends StatefulWidget {
  const HouseholdInvitationsScreen({
    super.key,
    required this.gateway,
    required this.session,
    required this.onBack,
  });

  final HouseholdGateway gateway;
  final HouseholdSession session;
  final VoidCallback onBack;

  @override
  State<HouseholdInvitationsScreen> createState() =>
      _HouseholdInvitationsScreenState();
}

class _HouseholdInvitationsScreenState
    extends State<HouseholdInvitationsScreen> {
  late HouseholdInvitationsViewModel _viewModel;
  HouseholdMemberRole _selectedRole = HouseholdMemberRole.member;

  bool get _isAdministrator =>
      widget.session.member.role == HouseholdMemberRole.administrator;

  @override
  void initState() {
    super.initState();
    _viewModel = HouseholdInvitationsViewModel(gateway: widget.gateway);
    if (_isAdministrator) {
      _viewModel.initialize();
    }
  }

  @override
  void didUpdateWidget(covariant HouseholdInvitationsScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.gateway != widget.gateway ||
        oldWidget.session != widget.session) {
      _viewModel.dispose();
      _viewModel = HouseholdInvitationsViewModel(gateway: widget.gateway);
      if (_isAdministrator) {
        _viewModel.initialize();
      }
    }
  }

  @override
  void dispose() {
    _viewModel.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Material(
      color: Theme.of(context).scaffoldBackgroundColor,
      child: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) {
            final padding = constraints.maxWidth < 700 ? 16.0 : 40.0;
            return SingleChildScrollView(
              padding: EdgeInsets.symmetric(horizontal: padding, vertical: 28),
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 880),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Align(
                        alignment: Alignment.centerLeft,
                        child: BalaurButton(
                          label: 'Back to Household',
                          variant: BalaurButtonVariant.ghost,
                          size: BalaurButtonSize.small,
                          leading: const Icon(Icons.arrow_back, size: 18),
                          onPressed: widget.onBack,
                        ),
                      ),
                      const SizedBox(height: 20),
                      Text(
                        'Household Invitations',
                        style: Theme.of(context).textTheme.headlineMedium
                            ?.copyWith(color: colors.foregroundStrong),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        'Create one-time access for another Household Member.',
                        style: Theme.of(context).textTheme.bodyLarge
                            ?.copyWith(color: colors.foreground),
                      ),
                      const SizedBox(height: 24),
                      if (!_isAdministrator)
                        const BalaurAlert(
                          title: 'Household Administrator access is required',
                          message: 'Only a Household Administrator can manage Household Invitations.',
                          tone: BalaurAlertTone.danger,
                        )
                      else
                        ListenableBuilder(
                          listenable: _viewModel,
                          builder: (context, _) => _buildState(context),
                        ),
                    ],
                  ),
                ),
              ),
            );
          },
        ),
      ),
    );
  }

  Widget _buildState(BuildContext context) {
    if (_viewModel.status == HouseholdInvitationsStatus.loading) {
      return const Column(
        children: [
          BalaurSkeleton(variant: BalaurSkeletonVariant.block, height: 150),
          SizedBox(height: 16),
          BalaurSkeleton(variant: BalaurSkeletonVariant.block, height: 120),
        ],
      );
    }
    if (_viewModel.status == HouseholdInvitationsStatus.failed) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          BalaurAlert(
            title: 'Household Invitations are not available',
            message:
                _viewModel.errorMessage ??
                'Balaur could not load the Household Invitations.',
            tone: BalaurAlertTone.danger,
          ),
          const SizedBox(height: 16),
          Align(
            alignment: Alignment.centerRight,
            child: BalaurButton(
              label: 'Try again',
              onPressed: _viewModel.initialize,
            ),
          ),
        ],
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _buildCreatePanel(context),
        if (_viewModel.createdInvitation case final created?) ...[
          const SizedBox(height: 20),
          _buildCreatedInvitation(context, created),
        ],
        if (_viewModel.errorMessage case final error?) ...[
          const SizedBox(height: 20),
          BalaurAlert(
            title: 'The action did not finish',
            message: error,
            tone: BalaurAlertTone.danger,
          ),
        ],
        const SizedBox(height: 28),
        Text(
          'Recent Household Invitations',
          style: Theme.of(context).textTheme.titleLarge
              ?.copyWith(color: BalaurColors.of(context).foregroundStrong),
        ),
        const SizedBox(height: 12),
        if (_viewModel.invitations.isEmpty)
          const BalaurAlert(
            title: 'No Household Invitations',
            message: 'Create a Household Invitation when another person is ready to join.',
            tone: BalaurAlertTone.info,
          )
        else
          ..._viewModel.invitations.map(
            (invitation) => Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: _buildInvitation(context, invitation),
            ),
          ),
      ],
    );
  }

  Widget _buildCreatePanel(BuildContext context) {
    final colors = BalaurColors.of(context);
    return BalaurSurface(
      ornate: true,
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            'Create a Household Invitation',
            style: Theme.of(context).textTheme.titleLarge
                ?.copyWith(color: colors.ink),
          ),
          const SizedBox(height: 8),
          Text(
            'The Household Invitation expires after 24 hours and works one time.',
            style: TextStyle(color: colors.ink),
          ),
          const SizedBox(height: 20),
          DropdownButtonFormField<HouseholdMemberRole>(
            initialValue: _selectedRole,
            isExpanded: true,
            style: TextStyle(color: colors.ink),
            decoration: const InputDecoration(labelText: 'Assigned role'),
            items: const [
              DropdownMenuItem(
                value: HouseholdMemberRole.member,
                child: Text('Household Member'),
              ),
              DropdownMenuItem(
                value: HouseholdMemberRole.administrator,
                child: Text('Household Administrator'),
              ),
            ],
            onChanged: _viewModel.isCreating
                ? null
                : (role) {
                    if (role != null) {
                      setState(() => _selectedRole = role);
                    }
                  },
          ),
          const SizedBox(height: 20),
          Align(
            alignment: Alignment.centerRight,
            child: BalaurButton(
              key: const Key('create-household-invitation'),
              label: _viewModel.isCreating
                  ? 'Creating invitation'
                  : 'Create invitation',
              onPressed: _viewModel.isCreating
                  ? null
                  : () async {
                      await _viewModel.create(_selectedRole);
                    },
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildCreatedInvitation(
    BuildContext context,
    CreatedHouseholdInvitation created,
  ) {
    final colors = BalaurColors.of(context);
    return BalaurSurface(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  'Household Invitation ready',
                  style: Theme.of(context).textTheme.titleLarge
                      ?.copyWith(color: colors.ink),
                ),
              ),
              IconButton(
                tooltip: 'Hide the Household Invitation',
                onPressed: _viewModel.hideCreatedInvitation,
                icon: const Icon(Icons.close),
              ),
            ],
          ),
          const SizedBox(height: 8),
          const BalaurAlert(
            title: 'Share this value now',
            message: 'The Household Server stores only its hash. Balaur cannot show this value again.',
            tone: BalaurAlertTone.info,
          ),
          const SizedBox(height: 20),
          Text(
            'Scan this Quick Response (QR) code to pair another device.',
            textAlign: TextAlign.center,
            style: TextStyle(color: colors.ink),
          ),
          const SizedBox(height: 12),
          Center(
            child: Semantics(
              label: 'Quick Response code for the Household Invitation',
              image: true,
              child: ColoredBox(
                color: Colors.white,
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: QrImageView(
                    data: created.payload.qrValue,
                    size: 220,
                    errorCorrectionLevel: QrErrorCorrectLevel.M,
                  ),
                ),
              ),
            ),
          ),
          const SizedBox(height: 20),
          Text(
            'Household Server',
            style: Theme.of(context).textTheme.labelLarge
                ?.copyWith(color: colors.ink),
          ),
          const SizedBox(height: 4),
          Text(
            created.payload.serverAddress.value,
            style: TextStyle(color: colors.ink),
          ),
          const SizedBox(height: 12),
          Text(
            'Household Invitation value',
            style: Theme.of(context).textTheme.labelLarge
                ?.copyWith(color: colors.ink),
          ),
          const SizedBox(height: 4),
          Text(
            created.payload.value,
            key: const Key('created-household-invitation-value'),
            style: TextStyle(color: colors.ink),
          ),
          const SizedBox(height: 20),
          Align(
            alignment: Alignment.centerRight,
            child: BalaurButton(
              key: const Key('copy-household-invitation'),
              label: 'Copy pairing details',
              variant: BalaurButtonVariant.ghost,
              leading: const Icon(Icons.copy, size: 18),
              onPressed: () async {
                await Clipboard.setData(
                  ClipboardData(
                    text:
                        'Household Server: ${created.payload.serverAddress.value}\n'
                        'Household Invitation: ${created.payload.value}',
                  ),
                );
                if (context.mounted) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(
                      content: Text('Household Invitation copied.'),
                    ),
                  );
                }
              },
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildInvitation(
    BuildContext context,
    HouseholdInvitation invitation,
  ) {
    final statusLabel = switch (invitation.status) {
      HouseholdInvitationStatus.active => 'Active',
      HouseholdInvitationStatus.expired => 'Expired',
      HouseholdInvitationStatus.used => 'Used',
      HouseholdInvitationStatus.canceled => 'Canceled',
    };
    final statusTone = switch (invitation.status) {
      HouseholdInvitationStatus.active => BalaurBadgeTone.teal,
      HouseholdInvitationStatus.used => BalaurBadgeTone.gold,
      HouseholdInvitationStatus.expired ||
      HouseholdInvitationStatus.canceled => BalaurBadgeTone.wood,
    };
    final roleLabel = switch (invitation.role) {
      HouseholdMemberRole.member => 'Household Member',
      HouseholdMemberRole.administrator => 'Household Administrator',
    };
    final colors = BalaurColors.of(context);

    return BalaurSurface(
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Wrap(
            spacing: 8,
            runSpacing: 8,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              BalaurBadge(label: statusLabel, tone: statusTone),
              BalaurBadge(label: roleLabel, tone: BalaurBadgeTone.gold),
            ],
          ),
          const SizedBox(height: 12),
          Text(
            'Created by ${invitation.creatorDisplayName}',
            style: TextStyle(color: colors.ink),
          ),
          const SizedBox(height: 4),
          Text(
            'Expires ${_formatUtc(invitation.expiresAt)}',
            style: TextStyle(color: colors.ink),
          ),
          if (invitation.status == HouseholdInvitationStatus.active) ...[
            const SizedBox(height: 16),
            Align(
              alignment: Alignment.centerRight,
              child: BalaurButton(
                key: Key('cancel-household-invitation-${invitation.id}'),
                label: _viewModel.isCanceling(invitation.id)
                    ? 'Canceling'
                    : 'Cancel invitation',
                variant: BalaurButtonVariant.ghost,
                size: BalaurButtonSize.small,
                onPressed: _viewModel.isCanceling(invitation.id)
                    ? null
                    : () => _confirmCancel(context, invitation),
              ),
            ),
          ],
        ],
      ),
    );
  }

  Future<void> _confirmCancel(
    BuildContext context,
    HouseholdInvitation invitation,
  ) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Cancel this Household Invitation?'),
        content: const Text(
          'The invitation value will stop working immediately.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('Keep invitation'),
          ),
          TextButton(
            key: const Key('confirm-cancel-household-invitation'),
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('Cancel invitation'),
          ),
        ],
      ),
    );
    if (confirmed ?? false) {
      await _viewModel.cancel(invitation.id);
    }
  }

  String _formatUtc(DateTime value) {
    final utc = value.toUtc();
    String twoDigits(int number) => number.toString().padLeft(2, '0');
    return '${utc.year}-${twoDigits(utc.month)}-${twoDigits(utc.day)} '
        '${twoDigits(utc.hour)}:${twoDigits(utc.minute)} UTC';
  }
}
