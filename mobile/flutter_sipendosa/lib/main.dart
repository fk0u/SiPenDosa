import 'package:flutter/material.dart';
import 'theme/app_theme.dart';
import 'services/api_service.dart';
import 'screens/main_shell.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  final apiService = ApiService(baseUrl: 'http://127.0.0.1:8473');
  runApp(SiPenDosaApp(apiService: apiService));
}

class SiPenDosaApp extends StatelessWidget {
  final ApiService apiService;
  const SiPenDosaApp({Key? key, required this.apiService}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'SiPenDosa',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.darkTheme,
      home: MainShell(apiService: apiService),
    );
  }
}
