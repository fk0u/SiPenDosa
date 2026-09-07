// SiPenDosa — Advanced WhatsApp Assistant Bot Client Script
(function () {
    let ws = null;
    let wsReconnectTimeout = null;

    function initWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;

        ws = new WebSocket(wsUrl);

        ws.onopen = function () {
            console.log('[SiPenDosa WS] Terhubung ke server realtime');
            if (wsReconnectTimeout) {
                clearTimeout(wsReconnectTimeout);
                wsReconnectTimeout = null;
            }
        };

        ws.onmessage = function (event) {
            try {
                const data = JSON.parse(event.data);
                handleEvent(data);
            } catch (e) {
                console.error('[SiPenDosa WS] Gagal mem-parse pesan:', e);
            }
        };

        ws.onclose = function () {
            console.warn('[SiPenDosa WS] Terputus dari server. Mencoba rekoneksi dalam 3 detik...');
            wsReconnectTimeout = setTimeout(initWebSocket, 3000);
        };

        ws.onerror = function (err) {
            console.error('[SiPenDosa WS] Terjadi kesalahan:', err);
            ws.close();
        };
    }

    function handleEvent(event) {
        switch (event.type) {
            case 'wa_status':
                updateWhatsAppStatus(event.payload);
                break;
            case 'wa_qr':
                updateWhatsAppQR(event.payload);
                break;
            case 'wa_pairing_code':
                if (event.payload && event.payload.code) {
                    renderPairingCode(event.payload.code);
                }
                break;
            case 'countdown':
                updateCountdown(event.payload);
                break;
            case 'toast':
                showToast(event.payload.level, event.payload.message);
                break;
            case 'queue_update':
                // Auto reload queue or stats if current page is queue or overview
                const queueTable = document.getElementById('queue-table-body');
                if (queueTable && window.htmx) {
                    window.location.reload();
                }
                break;
        }
    }

    function updateWhatsAppStatus(payload) {
        const statusBadge = document.getElementById('wa-status-badge');
        const statusText = document.getElementById('wa-status-text');
        const statusDot = document.getElementById('wa-status-dot');
        const qrBtn = document.getElementById('btn-scan-qr');
        const phoneText = document.getElementById('wa-phone-text');

        if (!statusBadge) return;

        const state = payload.state;
        statusBadge.className = 'badge badge-sm font-semibold gap-1.5 transition-all';

        if (state === 'connected') {
            statusBadge.classList.add('badge-success');
            if (statusText) statusText.innerText = 'WhatsApp Terhubung';
            if (statusDot) statusDot.className = 'w-2 h-2 rounded-full bg-emerald-400 animate-pulse';
            if (qrBtn) qrBtn.classList.add('hidden');
            if (phoneText && payload.phone) phoneText.innerText = payload.phone;
        } else if (state === 'need_qr') {
            statusBadge.classList.add('badge-warning');
            if (statusText) statusText.innerText = 'Perlu Scan QR';
            if (statusDot) statusDot.className = 'w-2 h-2 rounded-full bg-amber-400 animate-ping';
            if (qrBtn) qrBtn.classList.remove('hidden');
        } else if (state === 'connecting') {
            statusBadge.classList.add('badge-warning');
            if (statusText) statusText.innerText = 'Menyambungkan...';
            if (statusDot) statusDot.className = 'w-2 h-2 rounded-full bg-amber-400 animate-pulse';
            if (qrBtn) qrBtn.classList.add('hidden');
        } else {
            statusBadge.classList.add('badge-error');
            if (statusText) statusText.innerText = 'Terputus';
            if (statusDot) statusDot.className = 'w-2 h-2 rounded-full bg-rose-400';
            if (qrBtn) qrBtn.classList.remove('hidden');
        }
    }

    function updateWhatsAppQR(qrDataUrl) {
        const qrImg = document.getElementById('modal-qr-image');
        const qrPlaceholder = document.getElementById('modal-qr-placeholder');
        if (qrImg) {
            if (qrDataUrl) {
                qrImg.src = qrDataUrl;
                qrImg.classList.remove('hidden');
                if (qrPlaceholder) qrPlaceholder.classList.add('hidden');
            } else {
                qrImg.classList.add('hidden');
                if (qrPlaceholder) qrPlaceholder.classList.remove('hidden');
            }
        }
    }

    function updateCountdown(info) {
        const countdownTimer = document.getElementById('countdown-timer');
        const countdownMatkul = document.getElementById('countdown-matkul');
        const countdownRecipient = document.getElementById('countdown-recipient');
        const countdownPreview = document.getElementById('countdown-preview');

        if (!countdownTimer) return;

        if (!info) {
            countdownTimer.innerText = 'Tidak ada jadwal aktif';
            if (countdownMatkul) countdownMatkul.innerText = '-';
            if (countdownRecipient) countdownRecipient.innerText = '-';
            if (countdownPreview) countdownPreview.innerText = 'Semua jadwal kuliah nonaktif atau belum dikonfigurasi.';
            return;
        }

        countdownTimer.innerText = info.remaining_human;
        if (countdownMatkul) countdownMatkul.innerText = info.matkul;
        if (countdownRecipient) countdownRecipient.innerText = info.recipient_name;
        if (countdownPreview) countdownPreview.innerText = info.message_preview;
    }

    window.showToast = function (level, message) {
        const container = document.getElementById('toast-container');
        if (!container) return;

        const toast = document.createElement('div');
        let alertClass = 'alert-info';
        if (level === 'success') alertClass = 'alert-success';
        if (level === 'warning') alertClass = 'alert-warning';
        if (level === 'error') alertClass = 'alert-error';

        toast.className = `alert ${alertClass} shadow-xl text-sm font-medium transition-all transform translate-y-2 opacity-0 flex items-center justify-between gap-3`;
        toast.innerHTML = `
            <span>${message}</span>
            <button class="btn btn-ghost btn-xs" onclick="this.parentElement.remove()">✕</button>
        `;

        container.appendChild(toast);

        // Animate in
        setTimeout(() => {
            toast.classList.remove('translate-y-2', 'opacity-0');
        }, 50);

        // Auto remove after 4 seconds
        setTimeout(() => {
            toast.classList.add('opacity-0', '-translate-y-2');
            setTimeout(() => toast.remove(), 300);
        }, 4000);
    };

    window.openWhatsAppModal = function () {
        const modal = document.getElementById('qr_modal');
        if (!modal) return;
        modal.showModal();
        loadWhatsAppQR(false);
    };

    window.switchWaTab = function (tab) {
        const btnQr = document.getElementById('tab-btn-qr');
        const btnPhone = document.getElementById('tab-btn-phone');
        const contentQr = document.getElementById('tab-content-qr');
        const contentPhone = document.getElementById('tab-content-phone');

        if (!btnQr || !btnPhone || !contentQr || !contentPhone) return;

        if (tab === 'phone') {
            btnPhone.className = 'py-2 px-3 rounded-xl transition-all duration-200 bg-rose-600 text-white shadow-md';
            btnQr.className = 'py-2 px-3 rounded-xl transition-all duration-200 text-slate-400 hover:text-white';
            contentPhone.classList.remove('hidden');
            contentQr.classList.add('hidden');
            const phoneInput = document.getElementById('wa-pairing-phone');
            if (phoneInput) setTimeout(() => phoneInput.focus(), 100);
        } else {
            btnQr.className = 'py-2 px-3 rounded-xl transition-all duration-200 bg-rose-600 text-white shadow-md';
            btnPhone.className = 'py-2 px-3 rounded-xl transition-all duration-200 text-slate-400 hover:text-white';
            contentQr.classList.remove('hidden');
            contentPhone.classList.add('hidden');
            loadWhatsAppQR(false);
        }
    };

    window.loadWhatsAppQR = function (forceReconnect = false) {
        const qrImg = document.getElementById('modal-qr-image');
        const qrPlaceholder = document.getElementById('modal-qr-placeholder');
        const loadingText = document.getElementById('modal-qr-loading-text');

        if (loadingText) loadingText.innerText = forceReconnect ? 'Menghubungi server untuk QR baru...' : 'Mengambil QR Code WhatsApp...';
        if (qrImg) qrImg.classList.add('hidden');
        if (qrPlaceholder) qrPlaceholder.classList.remove('hidden');

        const url = forceReconnect ? '/api/wa/reconnect' : '/api/wa/qr';
        const method = forceReconnect ? 'POST' : 'GET';

        fetch(url, { method: method })
            .then(res => res.json())
            .then(data => {
                if (data.qr) {
                    updateWhatsAppQR(data.qr);
                } else if (data.state === 'connected') {
                    if (qrPlaceholder) {
                        qrPlaceholder.innerHTML = `
                            <span class="text-3xl">✓</span>
                            <span class="text-emerald-400 font-bold text-sm">WhatsApp Sudah Terhubung!</span>
                            <span class="text-xs text-slate-400 font-mono">${data.phone || ''}</span>
                        `;
                    }
                }
                if (data.pairing_code) {
                    renderPairingCode(data.pairing_code);
                }
            })
            .catch(err => {
                console.error('[SiPenDosa QR] Gagal memuat QR:', err);
                if (loadingText) loadingText.innerText = 'Gagal memuat QR. Periksa koneksi internet/server.';
            });
    };

    window.submitWhatsAppPairing = function () {
        const phoneInput = document.getElementById('wa-pairing-phone');
        const btnText = document.getElementById('pairing-btn-text');
        const btnSpinner = document.getElementById('pairing-btn-spinner');
        const btn = document.getElementById('btn-submit-pairing');

        const phone = phoneInput ? phoneInput.value.trim() : '';
        if (!phone) {
            alert('Silakan masukkan nomor WhatsApp Anda terlebih dahulu.');
            if (phoneInput) phoneInput.focus();
            return;
        }

        if (btn) btn.disabled = true;
        if (btnText) btnText.innerText = 'Menghubungi WhatsApp Engine...';
        if (btnSpinner) btnSpinner.classList.remove('hidden');

        fetch('/api/wa/pair-phone', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ phone: phone })
        })
            .then(r => r.json())
            .then(res => {
                if (btn) btn.disabled = false;
                if (btnSpinner) btnSpinner.classList.add('hidden');
                if (btnText) btnText.innerText = 'Dapatkan Kode Pairing (8 Digit)';

                if (res.success && res.code) {
                    renderPairingCode(res.code);
                    showToast('success', 'Kode pairing WhatsApp berhasil dibuat!');
                } else {
                    alert('Gagal mendapatkan kode pairing: ' + (res.error || 'Terjadi kesalahan'));
                }
            })
            .catch(err => {
                if (btn) btn.disabled = false;
                if (btnSpinner) btnSpinner.classList.add('hidden');
                if (btnText) btnText.innerText = 'Dapatkan Kode Pairing (8 Digit)';
                alert('Gagal mengirim permintaan: ' + err);
            });
    };

    function renderPairingCode(code) {
        const resultContainer = document.getElementById('wa-pairing-result');
        const display = document.getElementById('wa-pairing-code-display');
        if (resultContainer) resultContainer.classList.remove('hidden');
        if (display) {
            if (code.length === 8 && !code.includes('-')) {
                display.innerText = code.slice(0, 4) + ' - ' + code.slice(4);
            } else {
                display.innerText = code;
            }
        }
    }

    window.copyPairingCode = function () {
        const display = document.getElementById('wa-pairing-code-display');
        if (!display) return;
        const rawCode = display.innerText.replace(/\s+/g, '').replace('-', '');
        navigator.clipboard.writeText(rawCode).then(() => {
            showToast('success', '✓ Kode pairing (' + rawCode + ') disalin ke clipboard!');
        }).catch(() => {
            prompt('Salin kode pairing:', rawCode);
        });
    };

    // Initialize on DOM load
    document.addEventListener('DOMContentLoaded', initWebSocket);
})();
