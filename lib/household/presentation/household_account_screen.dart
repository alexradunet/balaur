import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:flutter/material.dart';

class HouseholdAccountScreen extends StatelessWidget {
  const HouseholdAccountScreen({
    super.key,
    required this.session,
    required this.onSignOut,
  });

  final HouseholdSession session;
  final Future<void> Function() onSignOut;

  @override
  Widget build(BuildContext context) {
    return HouseholdAccountView(session: session, onSignOut: onSignOut);
  }
}

class HouseholdAccountView extends StatelessWidget {
  const HouseholdAccountView({
    super.key,
    required this.session,
    required this.onSignOut,
    this.onEnter,
  });

  final HouseholdSession session;
  final Future<void> Function() onSignOut;
  final VoidCallback? onEnter;

  @override
  Widget build(BuildContext context) {
    final colors = BalaurColors.of(context);
    final roleLabel = switch (session.member.role) {
      HouseholdMemberRole.member => 'Household Member',
      HouseholdMemberRole.administrator => 'Household Administrator',
    };
    return SafeArea(
      child: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 28),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 640),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  'Household paired',
                  style: Theme.of(context).textTheme.headlineMedium
                      ?.copyWith(color: colors.foregroundStrong),
                ),
                const SizedBox(height: 8),
                Text(
                  'This device is connected to your Household Server.',
                  style: Theme.of(context).textTheme.bodyLarge
                      ?.copyWith(color: colors.foreground),
                ),
                const SizedBox(height: 24),
                BalaurSurface(
                  ornate: true,
                  padding: const EdgeInsets.all(24),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  session.member.displayName,
                                  key: const Key('household-member-name'),
                                  style: Theme.of(context).textTheme.titleLarge
                                      ?.copyWith(color: colors.ink),
                                ),
                                const SizedBox(height: 4),
                                Text(
                                  session.member.email,
                                  style: TextStyle(color: colors.ink),
                                ),
                              ],
                            ),
                          ),
                          const SizedBox(width: 12),
                          BalaurBadge(
                            label: roleLabel,
                            tone: BalaurBadgeTone.teal,
                          ),
                        ],
                      ),
                      const SizedBox(height: 20),
                      const BalaurStitch(),
                      const SizedBox(height: 20),
                      Text(
                        'Household Server',
                        style: Theme.of(context).textTheme.labelLarge
                            ?.copyWith(color: colors.ink),
                      ),
                      const SizedBox(height: 6),
                      Text(
                        session.serverAddress.value,
                        key: const Key('paired-household-server-address'),
                        style: TextStyle(color: colors.ink),
                      ),
                      const SizedBox(height: 20),
                      Text(
                        'Balaur stores this Household session in secure storage on this device.',
                        style: TextStyle(color: colors.ink),
                      ),
                      const SizedBox(height: 24),
                      Wrap(
                        alignment: WrapAlignment.end,
                        spacing: 12,
                        runSpacing: 12,
                        children: [
                          BalaurButton(
                            key: const Key('sign-out-household'),
                            label: 'Sign out',
                            variant: BalaurButtonVariant.ghost,
                            onPressed: () async {
                              await onSignOut();
                            },
                          ),
                          if (onEnter case final onEnter?)
                            BalaurButton(
                              key: const Key('enter-paired-application'),
                              label: 'Enter Balaur',
                              onPressed: onEnter,
                            ),
                        ],
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
