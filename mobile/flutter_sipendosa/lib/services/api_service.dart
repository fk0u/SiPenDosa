import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/dashboard_stats.dart';
import '../models/contact.dart';
import '../models/schedule.dart';

class ApiService {
  String baseUrl;

  ApiService({this.baseUrl = 'http://127.0.0.1:8473'});

  void updateBaseUrl(String url) {
    baseUrl = url.replaceAll(RegExp(r'/+$'), '');
  }

  Future<DashboardStats> getDashboardStats() async {
    try {
      final res = await http.get(Uri.parse('$baseUrl/api/stats')).timeout(const Duration(seconds: 4));
      if (res.statusCode == 200) {
        final data = jsonDecode(res.body);
        return DashboardStats.fromJson(data);
      }
    } catch (_) {}
    return DashboardStats();
  }

  Future<Map<String, dynamic>> getWhatsAppStatus() async {
    try {
      final res = await http.get(Uri.parse('$baseUrl/api/whatsapp/status')).timeout(const Duration(seconds: 4));
      if (res.statusCode == 200) {
        return jsonDecode(res.body) as Map<String, dynamic>;
      }
    } catch (_) {}
    return {'state': 'disconnected', 'phone': '', 'push_name': '', 'qr': ''};
  }

  Future<String> requestPairingCode(String phone) async {
    try {
      final res = await http.post(
        Uri.parse('$baseUrl/api/whatsapp/pair-phone'),
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: {'phone': phone},
      ).timeout(const Duration(seconds: 10));
      return res.body;
    } catch (e) {
      return 'ERR: ${e.toString()}';
    }
  }

  Future<List<Contact>> getContacts() async {
    try {
      final res = await http.get(Uri.parse('$baseUrl/api/contacts')).timeout(const Duration(seconds: 5));
      if (res.statusCode == 200) {
        final List list = jsonDecode(res.body);
        return list.map((e) => Contact.fromJson(e)).toList();
      }
    } catch (_) {}
    return [];
  }

  Future<List<Schedule>> getSchedules() async {
    try {
      final res = await http.get(Uri.parse('$baseUrl/api/schedules')).timeout(const Duration(seconds: 5));
      if (res.statusCode == 200) {
        final List list = jsonDecode(res.body);
        return list.map((e) => Schedule.fromJson(e)).toList();
      }
    } catch (_) {}
    return [];
  }
}
