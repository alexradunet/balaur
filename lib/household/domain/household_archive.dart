import 'dart:typed_data';

final class HouseholdArchive {
  HouseholdArchive({required this.fileName, required Uint8List bytes})
    : bytes = Uint8List.fromList(bytes);

  final String fileName;
  final Uint8List bytes;
}
