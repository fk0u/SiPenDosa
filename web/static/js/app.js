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

    // Initialize on DOM load
    document.addEventListener('DOMContentLoaded', initWebSocket);
})();
