// ==============================================================================
// SiPenDosa — High-Performance Native Bridge (C++17 Implementation)
// ==============================================================================

#include "sipen_bridge.h"

#include <chrono>
#include <cstring>
#include <arpa/inet.h>
#include <fcntl.h>
#include <netdb.h>
#include <netinet/in.h>
#include <signal.h>
#include <sys/select.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <unistd.h>

#if defined(__APPLE__)
#include <libproc.h>
#include <mach/mach.h>
#endif

extern "C" {

const char* sipen_get_bridge_version(void) {
    return "SiPenDosa-NativeBridge/2.0 (macOS C++17/Mach)";
}

int64_t sipen_check_socket_latency_us(const char* host, int port, int timeout_ms) {
    if (!host || port <= 0 || port > 65535) {
        return -1;
    }

    auto start_time = std::chrono::steady_clock::now();

    int sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock < 0) {
        return -1;
    }

    // Set non-blocking mode
    int flags = fcntl(sock, F_GETFL, 0);
    if (flags < 0 || fcntl(sock, F_SETFL, flags | O_NONBLOCK) < 0) {
        close(sock);
        return -1;
    }

    struct sockaddr_in server_addr;
    std::memset(&server_addr, 0, sizeof(server_addr));
    server_addr.sin_family = AF_INET;
    server_addr.sin_port = htons(static_cast<uint16_t>(port));

    if (inet_pton(AF_INET, host, &server_addr.sin_addr) <= 0) {
        // Resolve hostname jika bukan IP
        struct hostent* he = gethostbyname(host);
        if (!he || !he->h_addr_list[0]) {
            close(sock);
            return -1;
        }
        std::memcpy(&server_addr.sin_addr, he->h_addr_list[0], sizeof(struct in_addr));
    }

    int res = connect(sock, reinterpret_cast<struct sockaddr*>(&server_addr), sizeof(server_addr));
    if (res < 0) {
        if (errno != EINPROGRESS) {
            close(sock);
            return -1;
        }

        fd_set write_fds;
        FD_ZERO(&write_fds);
        FD_SET(sock, &write_fds);

        struct timeval tv;
        tv.tv_sec = timeout_ms / 1000;
        tv.tv_usec = (timeout_ms % 1000) * 1000;

        int sel = select(sock + 1, nullptr, &write_fds, nullptr, &tv);
        if (sel <= 0) {
            // Timeout atau error
            close(sock);
            return -1;
        }

        int so_error = 0;
        socklen_t len = sizeof(so_error);
        if (getsockopt(sock, SOL_SOCKET, SO_ERROR, &so_error, &len) < 0 || so_error != 0) {
            close(sock);
            return -1;
        }
    }

    auto end_time = std::chrono::steady_clock::now();
    close(sock);

    auto duration = std::chrono::duration_cast<std::chrono::microseconds>(end_time - start_time);
    return static_cast<int64_t>(duration.count());
}

bool sipen_is_process_alive(int pid) {
    if (pid <= 0) return false;
    // Signal 0 melakukan pengecekan keberadaan proses tanpa mengirim sinyal fisik
    return (kill(pid, 0) == 0);
}

bool sipen_terminate_pid(int pid, bool force) {
    if (pid <= 0) return false;
    int sig = force ? SIGKILL : SIGTERM;
    return (kill(pid, sig) == 0);
}

int64_t sipen_get_process_memory_rss(int pid) {
    if (pid <= 0) return -1;

#if defined(__APPLE__)
    struct proc_taskinfo pti;
    int ret = proc_pidinfo(pid, PROC_PIDTASKINFO, 0, &pti, sizeof(pti));
    if (ret == sizeof(pti)) {
        return static_cast<int64_t>(pti.pti_resident_size);
    }
#endif

    return -1;
}

} // extern "C"
