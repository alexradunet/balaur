import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/domain/calendar_connection.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/calendar_connection_view_model.dart';
import 'package:flutter/material.dart';

class CalendarConnectionScreen extends StatefulWidget {
  const CalendarConnectionScreen({
    super.key,
    required this.gateway,
    required this.session,
    required this.onBack,
    required this.openAuthorization,
  });

  final HouseholdGateway gateway;
  final HouseholdSession session;
  final VoidCallback onBack;
  final Future<bool> Function(Uri uri) openAuthorization;

  @override
  State<CalendarConnectionScreen> createState() =>
      _CalendarConnectionScreenState();
}

class _CalendarConnectionScreenState extends State<CalendarConnectionScreen>
    with WidgetsBindingObserver {
  late CalendarConnectionViewModel _viewModel;

  bool get _isAdministrator =>
      widget.session.member.role == HouseholdMemberRole.administrator;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _createViewModel();
  }

  @override
  void didUpdateWidget(covariant CalendarConnectionScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.gateway != widget.gateway ||
        oldWidget.session != widget.session ||
        oldWidget.openAuthorization != widget.openAuthorization) {
      _viewModel.dispose();
      _createViewModel();
    }
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed &&
        _viewModel.connection?.status == CalendarConnectionStatus.authorizing) {
      _viewModel.refresh();
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _viewModel.dispose();
    super.dispose();
  }

  void _createViewModel() {
    _viewModel = CalendarConnectionViewModel(
      widget.gateway,
      widget.openAuthorization,
    );
    if (_isAdministrator) {
      _viewModel.initialize();
    }
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
                        'Calendar Connection',
                        style: Theme.of(context).textTheme.headlineMedium
                            ?.copyWith(color: colors.foregroundStrong),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        'Connect one shared Google calendar to this Household.',
                        style: Theme.of(context).textTheme.bodyLarge
                            ?.copyWith(color: colors.foreground),
                      ),
                      const SizedBox(height: 24),
                      if (!_isAdministrator)
                        const BalaurAlert(
                          title: 'Household Administrator access is required',
                          message: 'Only a Household Administrator can manage the Calendar Connection.',
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
    if (_viewModel.isLoading) {
      return const Column(
        children: [
          BalaurSkeleton(variant: BalaurSkeletonVariant.block, height: 150),
          SizedBox(height: 16),
          BalaurSkeleton(variant: BalaurSkeletonVariant.block, height: 100),
        ],
      );
    }

    final connection = _viewModel.connection;
    if (connection == null) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          BalaurAlert(
            title: 'The Calendar Connection is not available',
            message:
                _viewModel.errorMessage ??
                'Balaur could not load the Calendar Connection.',
            tone: BalaurAlertTone.danger,
          ),
          const SizedBox(height: 16),
          Align(
            alignment: Alignment.centerRight,
            child: BalaurButton(
              key: const Key('retry-calendar-connection'),
              label: 'Try again',
              onPressed: _viewModel.refresh,
            ),
          ),
        ],
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (_viewModel.errorMessage case final message?) ...[
          BalaurAlert(
            title: 'The Calendar Connection action failed',
            message: message,
            tone: BalaurAlertTone.danger,
          ),
          const SizedBox(height: 16),
        ],
        _buildConnection(context, connection),
      ],
    );
  }

  Widget _buildConnection(BuildContext context, CalendarConnection connection) {
    return switch (connection.status) {
      CalendarConnectionStatus.disconnected => _buildDisconnected(
        context,
        connection,
      ),
      CalendarConnectionStatus.authorizing => _buildAuthorizing(
        context,
        connection,
      ),
      CalendarConnectionStatus.awaitingSelection => _buildSelection(
        context,
        connection,
      ),
      CalendarConnectionStatus.connected => _buildConnected(
        context,
        connection,
      ),
      CalendarConnectionStatus.authorizationFailed => _buildFailed(
        context,
        connection,
      ),
    };
  }

  Widget _buildDisconnected(
    BuildContext context,
    CalendarConnection connection,
  ) {
    return _connectionSurface(
      context,
      connection: connection,
      statusLabel: 'Disconnected',
      statusTone: BalaurBadgeTone.wood,
      body: const Text(
        'Google Calendar has no authorization for this Household.',
      ),
      actions: [
        BalaurButton(
          key: const Key('connect-google-calendar'),
          label: _viewModel.isBusy ? 'Opening browser' : 'Connect Google',
          onPressed: _viewModel.isBusy
              ? null
              : () => _viewModel.connect(replace: false),
        ),
      ],
    );
  }

  Widget _buildAuthorizing(
    BuildContext context,
    CalendarConnection connection,
  ) {
    return _connectionSurface(
      context,
      connection: connection,
      statusLabel: 'Authorization in progress',
      statusTone: BalaurBadgeTone.gold,
      body: const Text(
        'Complete Google authorization in the system browser. Then check the status.',
      ),
      actions: [
        BalaurButton(
          key: const Key('check-calendar-connection-status'),
          label: 'Check status',
          variant: BalaurButtonVariant.wood,
          onPressed: _viewModel.isBusy ? null : _viewModel.refresh,
        ),
        BalaurButton(
          key: const Key('disconnect-calendar-connection'),
          label: 'Disconnect',
          variant: BalaurButtonVariant.ghost,
          onPressed: _viewModel.isBusy
              ? null
              : () => _confirmDisconnect(context),
        ),
      ],
    );
  }

  Widget _buildSelection(BuildContext context, CalendarConnection connection) {
    final colors = BalaurColors.of(context);
    return _connectionSurface(
      context,
      connection: connection,
      statusLabel: 'Select one calendar',
      statusTone: BalaurBadgeTone.gold,
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Text(
            'Google authorization is complete. Select the shared calendar for this Household.',
          ),
          const SizedBox(height: 16),
          for (final calendar in connection.availableCalendars) ...[
            DecoratedBox(
              decoration: BoxDecoration(
                border: Border.all(color: colors.outline),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Row(
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            calendar.name,
                            style: Theme.of(context).textTheme.titleMedium
                                ?.copyWith(color: colors.ink),
                          ),
                          const SizedBox(height: 2),
                          Text(
                            calendar.id,
                            style: TextStyle(color: colors.ink),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(width: 12),
                    BalaurButton(
                      key: Key('select-calendar-${calendar.id}'),
                      label: 'Select',
                      size: BalaurButtonSize.small,
                      onPressed: _viewModel.isBusy
                          ? null
                          : () => _viewModel.selectCalendar(calendar.id),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 12),
          ],
        ],
      ),
      actions: [
        BalaurButton(
          key: const Key('disconnect-calendar-connection'),
          label: 'Disconnect',
          variant: BalaurButtonVariant.ghost,
          onPressed: _viewModel.isBusy
              ? null
              : () => _confirmDisconnect(context),
        ),
      ],
    );
  }

  Widget _buildConnected(BuildContext context, CalendarConnection connection) {
    final selected = connection.selectedCalendar;
    return _connectionSurface(
      context,
      connection: connection,
      statusLabel: 'Connected',
      statusTone: BalaurBadgeTone.teal,
      body: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            selected?.name ?? 'Unknown calendar',
            key: const Key('selected-calendar-name'),
            style: Theme.of(context).textTheme.titleLarge,
          ),
          if (selected != null) ...[
            const SizedBox(height: 4),
            Text(selected.id),
          ],
        ],
      ),
      actions: [
        BalaurButton(
          key: const Key('replace-calendar-connection'),
          label: _viewModel.isBusy ? 'Opening browser' : 'Replace connection',
          variant: BalaurButtonVariant.wood,
          onPressed: _viewModel.isBusy
              ? null
              : () => _viewModel.connect(replace: true),
        ),
        BalaurButton(
          key: const Key('disconnect-calendar-connection'),
          label: 'Disconnect',
          variant: BalaurButtonVariant.ghost,
          onPressed: _viewModel.isBusy
              ? null
              : () => _confirmDisconnect(context),
        ),
      ],
    );
  }

  Widget _buildFailed(BuildContext context, CalendarConnection connection) {
    return _connectionSurface(
      context,
      connection: connection,
      statusLabel: 'Authorization failed',
      statusTone: BalaurBadgeTone.wood,
      body: const Text(
        'Google did not authorize this Calendar Connection. Start authorization again.',
      ),
      actions: [
        BalaurButton(
          key: const Key('reconnect-google-calendar'),
          label: _viewModel.isBusy ? 'Opening browser' : 'Reconnect Google',
          onPressed: _viewModel.isBusy
              ? null
              : () => _viewModel.connect(replace: false),
        ),
        BalaurButton(
          key: const Key('disconnect-calendar-connection'),
          label: 'Disconnect',
          variant: BalaurButtonVariant.ghost,
          onPressed: _viewModel.isBusy
              ? null
              : () => _confirmDisconnect(context),
        ),
      ],
    );
  }

  Widget _connectionSurface(
    BuildContext context, {
    required CalendarConnection connection,
    required String statusLabel,
    required BalaurBadgeTone statusTone,
    required Widget body,
    required List<Widget> actions,
  }) {
    final colors = BalaurColors.of(context);
    return BalaurSurface(
      ornate: true,
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Wrap(
            spacing: 8,
            runSpacing: 8,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              BalaurBadge(label: 'Google Calendar', tone: BalaurBadgeTone.gold),
              BalaurBadge(label: statusLabel, tone: statusTone),
            ],
          ),
          const SizedBox(height: 20),
          DefaultTextStyle.merge(
            style: TextStyle(color: colors.ink),
            child: body,
          ),
          const SizedBox(height: 20),
          const BalaurStitch(),
          const SizedBox(height: 16),
          Text(
            'Household Time Zone',
            style: Theme.of(context).textTheme.labelLarge
                ?.copyWith(color: colors.ink),
          ),
          const SizedBox(height: 4),
          Text(
            connection.householdTimeZone,
            key: const Key('calendar-household-time-zone'),
            style: TextStyle(color: colors.ink),
          ),
          const SizedBox(height: 20),
          Wrap(
            alignment: WrapAlignment.end,
            spacing: 12,
            runSpacing: 12,
            children: actions,
          ),
        ],
      ),
    );
  }

  Future<void> _confirmDisconnect(BuildContext context) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Disconnect this Calendar Connection?'),
        content: const Text(
          'The Household will stop using the selected Google calendar.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('Keep connection'),
          ),
          TextButton(
            key: const Key('confirm-disconnect-calendar-connection'),
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('Disconnect'),
          ),
        ],
      ),
    );
    if (confirmed ?? false) {
      await _viewModel.disconnect();
    }
  }
}
