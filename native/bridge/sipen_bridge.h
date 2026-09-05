// ==============================================================================
// SiPenDosa — High-Performance Native Bridge (C++ / POSIX)
// Menyediakan fungsi low-level untuk socket healthcheck, latency timing, dan PID mgmt.
// ==============================================================================

#ifndef SIPEN_BRIDGE_H
#define SIPEN_BRIDGE_H

#include <stdint.h>
#include <stdbool.h>

#ifdef __cplusplus
extern "C" {
#endif

/// Menguji apakah server backend SiPenDosa sedang mendengarkan pada host:port.
/// Mengembalikan latency dalam mikrodetik jika sukses, atau -1 jika koneksi gagal / timeout.
int64_t sipen_check_socket_latency_us(const char* host, int port, int timeout_ms);

/// Memeriksa apakah suatu proses dengan PID tertentu masih aktif.
bool sipen_is_process_alive(int pid);

/// Menghentikan proses dengan aman (SIGTERM jika force=false, SIGKILL jika force=true).
bool sipen_terminate_pid(int pid, bool force);

/// Mendapatkan total memori fisik yang digunakan proses (RSS dalam byte) via mach kernel task API.
int64_t sipen_get_process_memory_rss(int pid);

/// Mengembalikan string versi bridge C++.
const char* sipen_get_bridge_version(void);

#ifdef __cplusplus
}
#endif

#endif // SIPEN_BRIDGE_H
