import 'package:flutter/material.dart';
import '../theme/app_theme.dart';
import '../widgets/bento_card.dart';
import '../models/dashboard_stats.dart';
import '../services/api_service.dart';

class DashboardScreen extends StatefulWidget {
  final ApiService apiService;
  const DashboardScreen({Key? key, required this.apiService}) : super(key: key);

  @override
  State<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends State<DashboardScreen> {
  DashboardStats _stats = DashboardStats();
  bool _isLoading = true;

  @override
  void initState() {
    super.initState();
    _fetchStats();
  }

  Future<void> _fetchStats() async {
    setState(() => _isLoading = true);
    final s = await widget.apiService.getDashboardStats();
    if (mounted) {
      setState(() {
        _stats = s;
        _isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Row(
          children: [
            const Text('SiPenDosa'),
            const SizedBox(width: 8),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
              decoration: BoxDecoration(
                color: AppTheme.burnishedGold.withOpacity(0.15),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: AppTheme.burnishedGold.withOpacity(0.3)),
              ),
              child: const Text(
                'v1.1.1',
                style: TextStyle(fontSize: 10, color: AppTheme.burnishedGold, fontWeight: FontWeight.bold),
              ),
            ),
          ],
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh, color: AppTheme.textLight),
            onPressed: _fetchStats,
          ),
        ],
      ),
      body: RefreshIndicator(
        color: AppTheme.crimson,
        backgroundColor: AppTheme.surface,
        onRefresh: _fetchStats,
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            // Onboarding 3-Step Wizard
            BentoCard(
              borderColor: AppTheme.burnishedGold.withOpacity(0.3),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      const Text('⚡', style: TextStyle(fontSize: 14)),
                      const SizedBox(width: 6),
                      Text(
                        'PANDUAN OPERASIONAL CEPAT',
                        style: TextStyle(
                          fontSize: 10,
                          fontWeight: FontWeight.bold,
                          color: AppTheme.burnishedGold,
                          letterSpacing: 1.2,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  const Text(
                    'Cara Kerja SiPenDosa',
                    style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900, color: Colors.white),
                  ),
                  const SizedBox(height: 4),
                  const Text(
                    'Kirim pengingat santun otomatis ke dosen tanpa sungkan dan tanpa repot.',
                    style: TextStyle(fontSize: 12, color: AppTheme.textMuted),
                  ),
                  const SizedBox(height: 16),
                  _buildStepTile('01', 'Tautkan WhatsApp', 'Scan QR atau gunakan 8-digit kode pairing'),
                  _buildStepTile('02', 'Daftarkan Dosen', '${_stats.totalContacts} kontak terdaftar'),
                  _buildStepTile('03', 'Pasang Jadwal Kuliah', '${_stats.activeSchedules} jadwal aktif tersimpan'),
                ],
              ),
            ),
            const SizedBox(height: 16),

            // Success Rate Card
            BentoCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text(
                        'TINGKAT KEBERHASILAN',
                        style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: AppTheme.textMuted),
                      ),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                        decoration: BoxDecoration(
                          color: AppTheme.emerald.withOpacity(0.15),
                          borderRadius: BorderRadius.circular(10),
                          border: Border.all(color: AppTheme.emerald.withOpacity(0.3)),
                        ),
                        child: const Text('STABLE', style: TextStyle(fontSize: 9, color: AppTheme.emerald, fontWeight: FontWeight.bold)),
                      ),
                    ],
                  ),
                  const SizedBox(height: 12),
                  Text(
                    '${_stats.successRate.toStringAsFixed(1)}%',
                    style: const TextStyle(fontSize: 40, fontWeight: FontWeight.w900, color: Colors.white, fontFamily: 'monospace'),
                  ),
                  const SizedBox(height: 4),
                  const Text('Rasio pengiriman pesan sukses otomatis tanpa kendala.', style: TextStyle(fontSize: 11, color: AppTheme.textMuted)),
                ],
              ),
            ),
            const SizedBox(height: 16),

            // Metrics Grid (2x2)
            Row(
              children: [
                Expanded(
                  child: BentoCard(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text('HARI INI', style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: AppTheme.textMuted)),
                        const SizedBox(height: 8),
                        Text(
                          '${_stats.totalSentToday}',
                          style: const TextStyle(fontSize: 28, fontWeight: FontWeight.bold, color: AppTheme.emerald, fontFamily: 'monospace'),
                        ),
                        const SizedBox(height: 4),
                        const Text('Pesan sukses', style: TextStyle(fontSize: 10, color: AppTheme.textMuted)),
                      ],
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: BentoCard(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text('TOTAL KIRIM', style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: AppTheme.textMuted)),
                        const SizedBox(height: 8),
                        Text(
                          '${_stats.totalSentAll}',
                          style: const TextStyle(fontSize: 28, fontWeight: FontWeight.bold, color: Colors.white, fontFamily: 'monospace'),
                        ),
                        const SizedBox(height: 4),
                        const Text('Sepanjang masa', style: TextStyle(fontSize: 10, color: AppTheme.textMuted)),
                      ],
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: BentoCard(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text('ANTREAN', style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: AppTheme.textMuted)),
                        const SizedBox(height: 8),
                        Text(
                          '${_stats.totalPending}',
                          style: const TextStyle(fontSize: 28, fontWeight: FontWeight.bold, color: AppTheme.burnishedGold, fontFamily: 'monospace'),
                        ),
                        const SizedBox(height: 4),
                        const Text('Menunggu jam kirim', style: TextStyle(fontSize: 10, color: AppTheme.textMuted)),
                      ],
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: BentoCard(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text('GAGAL', style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: AppTheme.textMuted)),
                        const SizedBox(height: 8),
                        Text(
                          '${_stats.totalFailedAll}',
                          style: const TextStyle(fontSize: 28, fontWeight: FontWeight.bold, color: AppTheme.crimson, fontFamily: 'monospace'),
                        ),
                        const SizedBox(height: 4),
                        const Text('Ditinjau otomatis', style: TextStyle(fontSize: 10, color: AppTheme.textMuted)),
                      ],
                    ),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildStepTile(String step, String title, String subtitle) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
            decoration: BoxDecoration(
              color: Colors.white.withOpacity(0.05),
              borderRadius: BorderRadius.circular(6),
            ),
            child: Text(
              step,
              style: const TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: AppTheme.burnishedGold, fontFamily: 'monospace'),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: const TextStyle(fontSize: 12, fontWeight: FontWeight.bold, color: Colors.white)),
                Text(subtitle, style: const TextStyle(fontSize: 11, color: AppTheme.textMuted)),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
