import 'dart:convert';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';
import '../models/dashboard_stats.dart';
import '../models/contact.dart';
import '../models/schedule.dart';

/// REST client untuk server SiPenDosa.
///
/// Sejak server mewajibkan autentikasi pada /api/*, client ini login sekali
/// lewat [login] (POST /api/auth/login), menyimpan bearer token di
/// SharedPreferences, lalu melampirkannya pada setiap permintaan.
class ApiService {
  String baseUrl;
  String? _token;

  ApiService({this.baseUrl = 'http://127.0.0.1:8473'});

  void updateBaseUrl(String url) {
    baseUrl = url.replaceAll(RegExp(r'/+$'), '');
  }

  /// Login dan simpan token. Panggil setelah server URL diatur.
  /// Return null jika sukses, atau pesan error.
  Future<String?> login(String username, String password) async {
    try {
      final res = await http
          .post(
            Uri.parse('$baseUrl/api/auth/login'),
            headers: {'Content-Type': 'application/json'},
            body: jsonEncode({'username': username, 'password': password}),
          )
          .timeout(const Duration(seconds: 8));
      if (res.statusCode == 200) {
        final data = jsonDecode(res.body);
        _token = data['token'] as String?;
        final prefs = await SharedPreferences.getInstance();
        await prefs.setString('sipendosa_token', _token ?? '');
        await prefs.setString('sipendosa_username', username);
        return null;
      }
      return 'Login gagal (${res.statusCode})';
    } catch (e) {
      return 'ERR: ${e.toString()}';
    }
  }

  /// Muat token yang tersimpan (panggil saat aplikasi start).
  Future<void> restoreSession() async {
    final prefs = await SharedPreferences.getInstance();
    _token = prefs.getString('sipendosa_token');
    final savedUrl = prefs.getString('sipendosa_base_url');
    if (savedUrl != null && savedUrl.isNotEmpty) {
      baseUrl = savedUrl;
    }
  }

  Future<void> updateBaseUrlAndPersist(String url) async {
    updateBaseUrl(url);
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('sipendosa_base_url', baseUrl);
  }

  Map<String, String> get _headers => {
        if (_token != null && _token!.isNotEmpty)
          'Authorization': 'Bearer $_token',
      };

  Future<http.Response?> _get(String path, {Duration? timeout}) async {
    try {
      final res = await http
          .get(Uri.parse('$baseUrl$path'), headers: _headers)
          .timeout(timeout ?? const Duration(seconds: 5));
      if (res.statusCode == 401) return null; // token invalid/expired
      return res;
    } catch (_) {
      return null;
    }
  }

  Future<DashboardStats> getDashboardStats() async {
    final res = await _get('/api/stats', timeout: const Duration(seconds: 4));
    if (res != null && res.statusCode == 200) {
      return DashboardStats.fromJson(jsonDecode(res.body));
    }
    return DashboardStats();
  }

  Future<Map<String, dynamic>> getWhatsAppStatus() async {
    final res =
        await _get('/api/whatsapp/status', timeout: const Duration(seconds: 4));
    if (res != null && res.statusCode == 200) {
      return jsonDecode(res.body) as Map<String, dynamic>;
    }
    return {'state': 'disconnected', 'phone': '', 'push_name': '', 'qr': ''};
  }

  Future<String> requestPairingCode(String phone) async {
    try {
      final res = await http
          .post(
            Uri.parse('$baseUrl/api/wa/pair-phone'),
            headers: {
              ..._headers,
              'Content-Type': 'application/x-www-form-urlencoded'
            },
            body: {'phone': phone},
          )
          .timeout(const Duration(seconds: 10));
      return res.body;
    } catch (e) {
      return 'ERR: ${e.toString()}';
    }
  }

  Future<List<Contact>> getContacts() async {
    final res = await _get('/api/contacts');
    if (res != null && res.statusCode == 200) {
      final List list = jsonDecode(res.body);
      return list.map((e) => Contact.fromJson(e)).toList();
    }
    return [];
  }

  Future<List<Schedule>> getSchedules() async {
    final res = await _get('/api/schedules');
    if (res != null && res.statusCode == 200) {
      final List list = jsonDecode(res.body);
      return list.map((e) => Schedule.fromJson(e)).toList();
    }
    return [];
  }
}
