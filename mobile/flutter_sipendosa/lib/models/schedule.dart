class Schedule {
  final int id;
  final String courseName;
  final String lecturerName;
  final int dayOfWeek; // 0=Minggu, 1=Senin, ...
  final String startTime;
  final String room;
  final bool isActive;
  final String reminderMode; // h-1, hari-h

  Schedule({
    required this.id,
    required this.courseName,
    required this.lecturerName,
    required this.dayOfWeek,
    required this.startTime,
    this.room = '',
    this.isActive = true,
    this.reminderMode = 'h-1',
  });

  factory Schedule.fromJson(Map<String, dynamic> json) {
    return Schedule(
      id: json['id'] ?? 0,
      courseName: json['course_name'] ?? '',
      lecturerName: json['lecturer_name'] ?? '',
      dayOfWeek: json['day_of_week'] ?? 1,
      startTime: json['start_time'] ?? '08:00',
      room: json['room'] ?? '',
      isActive: json['is_active'] == 1 || json['is_active'] == true,
      reminderMode: json['reminder_mode'] ?? 'h-1',
    );
  }

  String get dayName {
    const days = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu'];
    if (dayOfWeek >= 0 && dayOfWeek < days.length) return days[dayOfWeek];
    return '-';
  }
}
