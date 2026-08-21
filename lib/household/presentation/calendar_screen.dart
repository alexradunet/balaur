import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/domain/calendar_entry.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/calendar_view_model.dart';
import 'package:flutter/material.dart';

class CalendarScreen extends StatefulWidget {
  const CalendarScreen({
    super.key,
    required this.gateway,
    required this.session,
    required this.openAuthorization,
    this.onManageConnection,
    this.now,
  });

  final HouseholdGateway gateway;
  final HouseholdSession session;
  final Future<bool> Function(Uri uri) openAuthorization;
  final VoidCallback? onManageConnection;
  final DateTime Function()? now;

  @override
  State<CalendarScreen> createState() => _CalendarScreenState();
}

class _CalendarScreenState extends State<CalendarScreen> {
  late CalendarViewModel _viewModel;

  @override
  void initState() {
    super.initState();
    _createViewModel();
  }

  @override
  void didUpdateWidget(covariant CalendarScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.gateway != widget.gateway ||
        oldWidget.session != widget.session ||
        oldWidget.openAuthorization != widget.openAuthorization) {
      _viewModel.dispose();
      _createViewModel();
    }
  }

  @override
  void dispose() {
    _viewModel.dispose();
    super.dispose();
  }

  void _createViewModel() {
    _viewModel = CalendarViewModel(
      widget.gateway,
      widget.session.member.role == HouseholdMemberRole.administrator,
      widget.openAuthorization,
      now: widget.now,
    );
    _viewModel.initialize();
  }

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    return Material(
      color: Theme.of(context).scaffoldBackgroundColor,
      child: SafeArea(
        child: ListenableBuilder(
          listenable: _viewModel,
          builder: (context, _) {
            return LayoutBuilder(
              builder: (context, constraints) {
                final wide = constraints.maxWidth >= 900;
                final padding = constraints.maxWidth < 700 ? 4.0 : 40.0;
                return SingleChildScrollView(
                  padding: EdgeInsets.symmetric(
                    horizontal: padding,
                    vertical: 28,
                  ),
                  child: Center(
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(maxWidth: 1200),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          _buildHeader(context, colors),
                          const SizedBox(height: 24),
                          _buildBody(context, wide: wide),
                        ],
                      ),
                    ),
                  ),
                );
              },
            );
          },
        ),
      ),
    );
  }

  Widget _buildHeader(BuildContext context, BalaurColors colors) {
    return Wrap(
      alignment: WrapAlignment.spaceBetween,
      crossAxisAlignment: WrapCrossAlignment.center,
      spacing: 16,
      runSpacing: 12,
      children: [
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Household Calendar',
              style: Theme.of(context).textTheme.headlineMedium
                  ?.copyWith(color: colors.foregroundStrong),
            ),
            const SizedBox(height: 4),
            Text(
              _viewModel.state.householdTimeZone,
              key: const Key('calendar-time-zone'),
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: colors.foregroundStrong,
                fontWeight: FontWeight.w600,
              ),
            ),
          ],
        ),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            if (widget.onManageConnection case final manage?)
              BalaurButton(
                key: const Key('manage-calendar-connection'),
                label: 'Connection',
                variant: BalaurButtonVariant.ghost,
                onPressed: manage,
              ),
            BalaurButton(
              key: const Key('refresh-calendar'),
              label: _viewModel.state.isRefreshing ? 'Refreshing' : 'Refresh',
              variant: BalaurButtonVariant.wood,
              onPressed: _viewModel.state.isRefreshing
                  ? null
                  : _viewModel.refresh,
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildBody(BuildContext context, {required bool wide}) {
    final status = _viewModel.state.status;
    if (status == CalendarVisibleStatus.loading) {
      return const Column(
        children: [
          BalaurSkeleton(variant: BalaurSkeletonVariant.block, height: 420),
          SizedBox(height: 16),
          BalaurSkeleton(variant: BalaurSkeletonVariant.block, height: 180),
        ],
      );
    }
    if (status == CalendarVisibleStatus.disconnected ||
        status == CalendarVisibleStatus.authorizationFailure ||
        status == CalendarVisibleStatus.serverFailure) {
      return _buildUnavailable(context, status);
    }

    final month = _buildMonth(context);
    final agenda = _buildAgenda(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (status == CalendarVisibleStatus.refreshFailure) ...[
          BalaurAlert(
            title: 'Refresh failed',
            message:
                _viewModel.state.message ??
                'Balaur kept the current Calendar Entries.',
            tone: BalaurAlertTone.warning,
          ),
          const SizedBox(height: 16),
        ],
        if (wide)
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(flex: 3, child: month),
              const SizedBox(width: 24),
              Expanded(flex: 2, child: agenda),
            ],
          )
        else ...[
          month,
          const SizedBox(height: 20),
          agenda,
        ],
      ],
    );
  }

  Widget _buildUnavailable(BuildContext context, CalendarVisibleStatus status) {
    final authorization = status == CalendarVisibleStatus.authorizationFailure;
    final server = status == CalendarVisibleStatus.serverFailure;
    final title = server
        ? 'The Household Server is not available'
        : authorization
        ? 'Calendar authorization failed'
        : 'The Calendar Connection is disconnected';
    final memberMessage = server
        ? 'Balaur could not load Calendar Entries from the Household Server.'
        : _viewModel.isAdministrator
        ? 'Reconnect the shared Calendar Source to continue.'
        : 'Ask a Household Administrator to reconnect the shared Calendar Source.';
    return BalaurSurface(
      ornate: true,
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          BalaurAlert(
            title: title,
            message: server || authorization
                ? _viewModel.state.message ?? memberMessage
                : memberMessage,
            tone: server ? BalaurAlertTone.danger : BalaurAlertTone.warning,
          ),
          const SizedBox(height: 20),
          Wrap(
            alignment: WrapAlignment.end,
            spacing: 12,
            runSpacing: 12,
            children: [
              if (server)
                BalaurButton(
                  key: const Key('retry-calendar'),
                  label: 'Try again',
                  onPressed: _viewModel.refresh,
                ),
              if (!server && _viewModel.isAdministrator)
                BalaurButton(
                  key: const Key('reconnect-calendar'),
                  label: 'Reconnect calendar',
                  onPressed: _viewModel.reconnect,
                ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildMonth(BuildContext context) {
    const weekdays = ['MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT', 'SUN'];
    final colors = BalaurColors.of(context);
    final month = _viewModel.state.displayMonth;
    final selected = _viewModel.state.selectedDate;
    final today = _viewModel.today;
    final days = _viewModel.visibleDays;
    return BalaurSurface(
      key: const Key('calendar-month-panel'),
      ornate: true,
      padding: const EdgeInsets.all(4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Wrap(
            alignment: WrapAlignment.spaceBetween,
            crossAxisAlignment: WrapCrossAlignment.center,
            spacing: 12,
            runSpacing: 8,
            children: [
              Text(
                _monthLabel(month),
                key: const Key('visible-calendar-month'),
                style: Theme.of(context).textTheme.titleLarge
                    ?.copyWith(color: colors.ink),
              ),
              Wrap(
                spacing: 8,
                children: [
                  BalaurButton(
                    key: const Key('previous-calendar-month'),
                    label: 'Previous',
                    variant: BalaurButtonVariant.ghost,
                    onPressed: _viewModel.previousMonth,
                  ),
                  BalaurButton(
                    key: const Key('calendar-today'),
                    label: 'Today',
                    variant: BalaurButtonVariant.wood,
                    onPressed: _viewModel.goToToday,
                  ),
                  BalaurButton(
                    key: const Key('next-calendar-month'),
                    label: 'Next',
                    variant: BalaurButtonVariant.ghost,
                    onPressed: _viewModel.nextMonth,
                  ),
                ],
              ),
            ],
          ),
          const SizedBox(height: 16),
          Row(
            children: [
              for (final weekday in weekdays)
                Expanded(
                  child: Center(
                    child: Text(
                      weekday,
                      style: Theme.of(context).textTheme.labelMedium?.copyWith(
                        color: colors.ink,
                        fontWeight: FontWeight.w700,
                        fontSize: 12,
                      ),
                    ),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 8),
          GridView.builder(
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            itemCount: days.length,
            gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
              crossAxisCount: 7,
              crossAxisSpacing: 0,
              mainAxisSpacing: 0,
            ),
            itemBuilder: (context, index) {
              final date = days[index];
              final count = _viewModel.entryCountFor(date);
              final isToday = _sameDate(date, today);
              final isSelected = _sameDate(date, selected);
              return BalaurCalendarCell(
                key: Key('calendar-day-${_dateKey(date)}'),
                day: date.day,
                pips: count,
                today: isToday,
                selected: isSelected,
                dimmed: date.month != month.month,
                semanticLabel: _dateSemanticLabel(
                  date,
                  count: count,
                  today: isToday,
                  selected: isSelected,
                ),
                onPressed: () => _viewModel.selectDate(date),
              );
            },
          ),
        ],
      ),
    );
  }

  Widget _buildAgenda(BuildContext context) {
    final colors = BalaurColors.of(context);
    final entries = _viewModel.selectedEntries;
    final status = _viewModel.state.status;
    return BalaurSurface(
      key: const Key('calendar-agenda-panel'),
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            _fullDate(_viewModel.state.selectedDate),
            key: const Key('selected-calendar-date'),
            style: Theme.of(context).textTheme.titleLarge
                ?.copyWith(color: colors.ink),
          ),
          const SizedBox(height: 16),
          if (status == CalendarVisibleStatus.emptyMonth)
            const BalaurEmptyState(
              title: 'No Calendar Entries this month',
              message: 'The shared calendar has no entries in this month.',
            )
          else if (status == CalendarVisibleStatus.emptyDay)
            const BalaurEmptyState(
              title: 'No Calendar Entries today',
              message: 'Select another day or refresh the calendar.',
            )
          else
            for (var index = 0; index < entries.length; index++)
              BalaurDayEntry(
                time: _viewModel.entryTimeLabel(entries[index]),
                title: entries[index].title,
                detail: entries[index].location,
                tone: _tone(entries[index]),
                last: index == entries.length - 1,
              ),
        ],
      ),
    );
  }

  BalaurDayEntryTone _tone(CalendarEntry entry) {
    final value = entry.colorHex.codeUnits.fold<int>(
      0,
      (sum, unit) => sum + unit,
    );
    return BalaurDayEntryTone.values[value % BalaurDayEntryTone.values.length];
  }

  String _monthLabel(DateTime date) {
    const months = [
      'January',
      'February',
      'March',
      'April',
      'May',
      'June',
      'July',
      'August',
      'September',
      'October',
      'November',
      'December',
    ];
    return '${months[date.month - 1]} ${date.year}';
  }

  String _fullDate(DateTime date) {
    const weekdays = [
      'Monday',
      'Tuesday',
      'Wednesday',
      'Thursday',
      'Friday',
      'Saturday',
      'Sunday',
    ];
    return '${weekdays[date.weekday - 1]}, ${_monthLabel(date).split(' ').first} ${date.day}, ${date.year}';
  }

  String _dateSemanticLabel(
    DateTime date, {
    required int count,
    required bool today,
    required bool selected,
  }) {
    final parts = <String>[
      _fullDate(date),
      '$count Calendar ${count == 1 ? 'Entry' : 'Entries'}',
      if (today) 'today',
      if (selected) 'selected',
    ];
    return parts.join(', ');
  }

  String _dateKey(DateTime date) {
    return '${date.year}-${date.month.toString().padLeft(2, '0')}-${date.day.toString().padLeft(2, '0')}';
  }

  bool _sameDate(DateTime left, DateTime right) {
    return left.year == right.year &&
        left.month == right.month &&
        left.day == right.day;
  }
}
