import 'package:balaur/household/domain/household_invitation.dart';
import 'package:balaur/household/domain/household_server_address.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('HouseholdInvitationPayload', () {
    test('round-trips the Household Server and invitation through QR data', () {
      final payload = HouseholdInvitationPayload(
        serverAddress: HouseholdServerAddress.parse(
          'https://household.example.com',
        ),
        value: 'InvitationValue'.padRight(48, 'A'),
      );

      expect(HouseholdInvitationPayload.parseQrValue(payload.qrValue), payload);
      expect(payload.qrValue, contains('server='));
      expect(payload.qrValue, contains('invitation='));
    });

    test('rejects an insecure Household Server in QR data', () {
      final qrValue = Uri(
        scheme: 'balaur',
        host: 'household',
        path: '/join',
        queryParameters: {
          'server': 'http://household.example.com',
          'invitation': 'InvitationValue'.padRight(48, 'A'),
        },
      ).toString();

      expect(
        () => HouseholdInvitationPayload.parseQrValue(qrValue),
        throwsFormatException,
      );
    });

    test('rejects malformed or short invitation values', () {
      expect(
        () => HouseholdInvitationPayload(
          serverAddress: HouseholdServerAddress.parse(
            'https://household.example.com',
          ),
          value: 'short',
        ),
        throwsFormatException,
      );
      expect(
        () => HouseholdInvitationPayload.parseQrValue('not-a-pairing-code'),
        throwsFormatException,
      );
    });
  });
}
