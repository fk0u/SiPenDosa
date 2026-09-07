class Contact {
  final int id;
  final String name;
  final String title;
  final String phone;
  final String type; // dosen, komti, mahasiswa, group

  Contact({
    required this.id,
    required this.name,
    this.title = '',
    required this.phone,
    this.type = 'dosen',
  });

  factory Contact.fromJson(Map<String, dynamic> json) {
    return Contact(
      id: json['id'] ?? 0,
      name: json['name'] ?? '',
      title: json['title'] ?? '',
      phone: json['phone'] ?? '',
      type: json['type'] ?? 'dosen',
    );
  }

  String get displayName => title.isNotEmpty ? '$name, $title' : name;
}
