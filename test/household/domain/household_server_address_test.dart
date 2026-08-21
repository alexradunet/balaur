import 'package:balaur/household/domain/household_server_address.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('HouseholdServerAddress', () {
    test('normalizes a stable HTTPS address', () {
      final address = HouseholdServerAddress.parse(
        '  https://Household.Example.com/base/  ',
      );

      expect(address.value, 'https://household.example.com/base');
    });

    test('rejects an insecure address', () {
      expect(
        () => HouseholdServerAddress.parse('http://household.example.com'),
        throwsFormatException,
      );
    });

    test('rejects credentials, queries, and fragments', () {
      for (final value in [
        'https://member@household.example.com',
        'https://household.example.com?token=secret',
        'https://household.example.com/#setup',
      ]) {
        expect(
          () => HouseholdServerAddress.parse(value),
          throwsFormatException,
          reason: value,
        );
      }
    });

    test('permits insecure loopback when explicitly enabled', () {
      final address = HouseholdServerAddress.parse(
        'http://localhost:8090',
        allowInsecureLoopback: true,
      );

      expect(address.value, 'http://localhost:8090');
      expect(
        () => HouseholdServerAddress.parse(
          'http://household.example.com',
          allowInsecureLoopback: true,
        ),
        throwsFormatException,
      );
    });
  });
}
