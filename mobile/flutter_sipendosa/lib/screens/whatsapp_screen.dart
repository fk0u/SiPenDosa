import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import '../theme/app_theme.dart';
import '../widgets/bento_card.dart';
import '../services/api_service.dart';

class WhatsAppScreen extends StatefulWidget {
  final ApiService apiService;
  const WhatsAppScreen({Key? key, required this.apiService}) : super(key: key);

  @override
  State<WhatsAppScreen> createState() => _WhatsAppScreenState();
}

class _WhatsAppScreenState extends State<WhatsAppScreen> {
  final TextEditingController _phoneController = TextEditingController();
  String _pairingCode = '';
  bool _isLoading = false;
  String _status = 'Memeriksa status WhatsApp...';
  bool _isConnected = false;

  @override
  void initState() {
    super.initState();
    _checkStatus();
  }

  Future<void> _checkStatus() async {
    final s = await widget.apiService.getWhatsAppStatus();
    if (mounted) {
      setState(() {
        _isConnected = (s['state'] == 'connected');
        _status = _isConnected ? 'Terhubung (${s['push_name'] ?? s['phone']})' : 'Belum Terhubung';
      });
    }
  }

  Future<void> _requestPairingCode() async {
    final phone = _phoneController.text.trim();
    if (phone.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Nomor telepon tidak boleh kosong')),
      );
      return;
    }

    setState(() => _isLoading = true);
    final code = await widget.apiService.requestPairingCode(phone);
    if (mounted) {
      setState(() {
        _isLoading = false;
        _pairingCode = code;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('WhatsApp Manager'),
        actions: [
          IconButton(icon: const Icon(Icons.refresh), onPressed: _checkStatus),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // Connection Status Card
          BentoCard(
            borderColor: _isConnected ? AppTheme.emerald.withOpacity(0.4) : AppTheme.burnishedGold.withOpacity(0.4),
            child: Row(
              children: [
                Container(
                  width: 12,
                  height: 12,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: _isConnected ? AppTheme.emerald : AppTheme.burnishedGold,
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        _isConnected ? 'WHATSAPP AKTIF' : 'WHATSAPP STANDBY',
                        style: TextStyle(
                          fontSize: 10,
                          fontWeight: FontWeight.bold,
                          color: _isConnected ? AppTheme.emerald : AppTheme.burnishedGold,
                          letterSpacing: 1.0,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(_status, style: const TextStyle(fontSize: 14, fontWeight: FontWeight.bold, color: Colors.white)),
                    ],
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),

          // Method 1: Phone Number Pairing Code
          BentoCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: const [
                    Text('🔢', style: TextStyle(fontSize: 14)),
                    SizedBox(width: 8),
                    Text(
                      'TAUTKAN DENGAN NOMOR TELEPON',
                      style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: AppTheme.burnishedGold),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                const Text(
                  'Gunakan kode 8-digit resmi jika Anda kesulitan memindai QR Code di layar ponsel ini.',
                  style: TextStyle(fontSize: 12, color: AppTheme.textMuted),
                ),
                const SizedBox(height: 14),
                TextField(
                  controller: _phoneController,
                  keyboardType: TextInputType.phone,
                  style: const TextStyle(color: Colors.white),
                  decoration: InputDecoration(
                    hintText: '6281234567890',
                    hintStyle: const TextStyle(color: AppTheme.textMuted),
                    filled: true,
                    fillColor: AppTheme.surface,
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(12),
                      borderSide: const BorderSide(color: AppTheme.border),
                    ),
                    prefixIcon: const Icon(Icons.phone, color: AppTheme.burnishedGold),
                  ),
                ),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  height: 46,
                  child: ElevatedButton(
                    style: ElevatedButton.styleFrom(
                      backgroundColor: AppTheme.crimson,
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                    ),
                    onPressed: _isLoading ? null : _requestPairingCode,
                    child: _isLoading
                        ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(color: Colors.white, strokeWidth: 2))
                        : const Text('Minta Kode Pairing 8-Digit', style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold)),
                  ),
                ),

                if (_pairingCode.isNotEmpty) ...[
                  const SizedBox(height: 18),
                  Container(
                    width: double.infinity,
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      color: AppTheme.surface,
                      borderRadius: BorderRadius.circular(14),
                      border: Border.all(color: AppTheme.burnishedGold.withOpacity(0.5)),
                    ),
                    child: Column(
                      children: [
                        const Text('KODE PAIRING WHATSAPP ANDA', style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: AppTheme.burnishedGold)),
                        const SizedBox(height: 8),
                        Text(
                          _pairingCode,
                          style: const TextStyle(fontSize: 28, fontWeight: FontWeight.w900, letterSpacing: 4, color: Colors.white, fontFamily: 'monospace'),
                        ),
                        const SizedBox(height: 8),
                        TextButton.icon(
                          icon: const Icon(Icons.copy, size: 16, color: AppTheme.burnishedGold),
                          label: const Text('Salin Kode', style: TextStyle(color: AppTheme.burnishedGold)),
                          onPressed: () {
                            Clipboard.setData(ClipboardData(text: _pairingCode));
                            ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(content: Text('✓ Kode pairing disalin!')),
                            );
                          },
                        ),
                      ],
                    ),
                  ),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}
