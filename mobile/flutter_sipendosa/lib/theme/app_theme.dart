import 'package:flutter/material.dart';

class AppTheme {
  // Brand Color Tokens
  static const Color background = Color(0xFF08090D);
  static const Color surface = Color(0xFF0D1017);
  static const Color card = Color(0xFF131823);
  static const Color border = Color(0xFF1E293B);
  static const Color crimson = Color(0xFFE11D48);
  static const Color burnishedGold = Color(0xFFF59E0B);
  static const Color emerald = Color(0xFF10B981);
  static const Color textMuted = Color(0xFF64748B);
  static const Color textLight = Color(0xFF94A3B8);

  static ThemeData get darkTheme {
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.dark,
      scaffoldBackgroundColor: background,
      colorScheme: const ColorScheme.dark(
        primary: crimson,
        secondary: burnishedGold,
        surface: surface,
        background: background,
        error: crimson,
        onPrimary: Colors.white,
        onSecondary: Colors.black,
        onSurface: Colors.white,
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: surface,
        elevation: 0,
        centerTitle: false,
        titleTextStyle: TextStyle(
          color: Colors.white,
          fontSize: 18,
          fontWeight: FontWeight.w900,
          letterSpacing: -0.5,
        ),
      ),
      cardTheme: CardTheme(
        color: card,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(20),
          side: const BorderSide(color: border, width: 1),
        ),
      ),
    );
  }
}
