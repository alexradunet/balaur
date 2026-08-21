import 'package:balaur/chat/data/chat_gateway.dart';
import 'package:balaur/chat/data/conversation_repository.dart';
import 'package:balaur/chat/data/openai_chat_gateway.dart';
import 'package:balaur/chat/presentation/chat_screen.dart';
import 'package:balaur/design_system/design_system.dart';
import 'package:balaur/household/data/household_credential_store.dart';
import 'package:balaur/household/data/household_gateway.dart';
import 'package:balaur/household/data/pocketbase_household_gateway.dart';
import 'package:balaur/household/domain/household_session.dart';
import 'package:balaur/household/presentation/household_account_screen.dart';
import 'package:balaur/household/presentation/household_invitations_screen.dart';
import 'package:balaur/household/presentation/household_pairing_gate.dart';
import 'package:balaur/household/presentation/household_qr_scanner_screen.dart';
import 'package:balaur/navigation/presentation/balaur_navigation_shell.dart';
import 'package:balaur/settings/presentation/provider_settings_screen.dart';
import 'package:balaur/settings/provider_settings_store.dart';
import 'package:flutter/material.dart';
import 'package:flutter_driver/driver_extension.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:go_router/go_router.dart';

const _householdInvitationsLocation = '/household/invitations';

void main() {
  if (const bool.fromEnvironment('ENABLE_FLUTTER_DRIVER')) {
    enableFlutterDriverExtension();
  }

  WidgetsFlutterBinding.ensureInitialized();
  const secureStorage = FlutterSecureStorage();
  runApp(
    BalaurApp(
      gateway: OpenAiChatGateway(),
      conversationRepository: InMemoryConversationRepository(),
      settingsStore: SecureProviderSettingsStore(secureStorage),
      householdGateway: PocketBaseHouseholdGateway(
        credentialStore: SecureHouseholdCredentialStore(secureStorage),
      ),
    ),
  );
}

class BalaurApp extends StatefulWidget {
  const BalaurApp({
    super.key,
    required this.gateway,
    required this.conversationRepository,
    required this.settingsStore,
    required this.householdGateway,
  });

  final ChatGateway gateway;
  final ConversationRepository conversationRepository;
  final ProviderSettingsStore settingsStore;
  final HouseholdGateway householdGateway;

  @override
  State<BalaurApp> createState() => _BalaurAppState();
}

class _BalaurAppState extends State<BalaurApp> {
  late final GoRouter _router;
  ThemeMode _themeMode = ThemeMode.system;

  @override
  void initState() {
    super.initState();
    _router = GoRouter(
      initialLocation: BalaurDestination.chat.location,
      routes: [
        GoRoute(path: '/', redirect: (_, _) => BalaurDestination.chat.location),
        ShellRoute(
          builder: (context, state, child) {
            final active =
                state.uri.path.startsWith(BalaurDestination.household.location)
                ? BalaurDestination.household
                : BalaurDestination.values.firstWhere(
                    (destination) => destination.location == state.uri.path,
                    orElse: () => BalaurDestination.chat,
                  );
            return BalaurNavigationShell(
              active: active,
              onSelect: (destination) => context.go(destination.location),
              onToggleTheme: _toggleTheme,
              child: child,
            );
          },
          routes: [
            GoRoute(
              path: BalaurDestination.chat.location,
              builder: (context, _) => ChatScreen(
                gateway: widget.gateway,
                conversationRepository: widget.conversationRepository,
                settingsStore: widget.settingsStore,
                onOpenSettings: () =>
                    context.go(BalaurDestination.settings.location),
              ),
            ),
            GoRoute(
              path: BalaurDestination.household.location,
              builder: (context, _) {
                final scope = HouseholdSessionScope.of(context);
                final signOut = scope.onSignOut;
                return HouseholdAccountScreen(
                  session: scope.session,
                  onManageInvitations:
                      scope.session.member.role ==
                          HouseholdMemberRole.administrator
                      ? () => context.go(_householdInvitationsLocation)
                      : null,
                  onSignOut: () async {
                    context.go(BalaurDestination.chat.location);
                    await signOut();
                  },
                );
              },
            ),
            GoRoute(
              path: _householdInvitationsLocation,
              builder: (context, _) {
                final scope = HouseholdSessionScope.of(context);
                return HouseholdInvitationsScreen(
                  gateway: widget.householdGateway,
                  session: scope.session,
                  onBack: () =>
                      context.go(BalaurDestination.household.location),
                );
              },
            ),
            GoRoute(
              path: BalaurDestination.settings.location,
              builder: (_, _) =>
                  ProviderSettingsScreen(settingsStore: widget.settingsStore),
            ),
          ],
        ),
      ],
    );
  }

  @override
  void dispose() {
    _router.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'Balaur',
      debugShowCheckedModeBanner: false,
      theme: BalaurTheme.light(),
      darkTheme: BalaurTheme.dark(),
      themeMode: _themeMode,
      routerConfig: _router,
      builder: (context, child) => HouseholdPairingGate(
        gateway: widget.householdGateway,
        pairedChild: child ?? const SizedBox.shrink(),
        onScanInvitation: _scanHouseholdInvitation,
      ),
    );
  }

  Future<String?> _scanHouseholdInvitation(BuildContext context) {
    return Navigator.of(context).push<String>(
      MaterialPageRoute<String>(
        builder: (_) => const HouseholdQrScannerScreen(),
      ),
    );
  }

  void _toggleTheme() {
    final platformIsDark =
        WidgetsBinding.instance.platformDispatcher.platformBrightness ==
        Brightness.dark;
    final isDark =
        _themeMode == ThemeMode.dark ||
        (_themeMode == ThemeMode.system && platformIsDark);
    setState(() {
      _themeMode = isDark ? ThemeMode.light : ThemeMode.dark;
    });
  }
}
