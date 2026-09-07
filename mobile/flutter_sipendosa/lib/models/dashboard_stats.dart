class DashboardStats {
  final int totalSentToday;
  final int totalSentAll;
  final int totalFailedAll;
  final int totalPending;
  final int totalSchedules;
  final int activeSchedules;
  final int totalContacts;
  final double successRate;

  DashboardStats({
    this.totalSentToday = 0,
    this.totalSentAll = 0,
    this.totalFailedAll = 0,
    this.totalPending = 0,
    this.totalSchedules = 0,
    this.activeSchedules = 0,
    this.totalContacts = 0,
    this.successRate = 100.0,
  });

  factory DashboardStats.fromJson(Map<String, dynamic>? json) {
    if (json == null) return DashboardStats();
    return DashboardStats(
      totalSentToday: json['total_sent_today'] ?? 0,
      totalSentAll: json['total_sent_all'] ?? 0,
      totalFailedAll: json['total_failed_all'] ?? 0,
      totalPending: json['total_pending'] ?? 0,
      totalSchedules: json['total_schedules'] ?? 0,
      activeSchedules: json['active_schedules'] ?? json['total_schedules'] ?? 0,
      totalContacts: json['total_contacts'] ?? 0,
      successRate: (json['success_rate'] as num?)?.toDouble() ?? 100.0,
    );
  }
}
