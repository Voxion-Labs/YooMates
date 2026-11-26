#ifndef YOOMATES_LOCK_HPP
#define YOOMATES_LOCK_HPP

#include <atomic>

namespace yoomates {

// High-performance, low-latency spinlock for real-time audio packet routing
// Fully portable across MinGW, GCC, Clang, and MSVC without OS threading overhead.
class FastMutex {
private:
    std::atomic_flag flag_ = ATOMIC_FLAG_INIT;

public:
    FastMutex() = default;
    FastMutex(const FastMutex&) = delete;
    FastMutex& operator=(const FastMutex&) = delete;

    void lock() noexcept {
        while (flag_.test_and_set(std::memory_order_acquire)) {
            // Low-overhead spin
        }
    }

    void unlock() noexcept {
        flag_.clear(std::memory_order_release);
    }
};

class FastLockGuard {
private:
    FastMutex& mutex_;

public:
    explicit FastLockGuard(FastMutex& m) noexcept : mutex_(m) {
        mutex_.lock();
    }

    ~FastLockGuard() noexcept {
        mutex_.unlock();
    }

    FastLockGuard(const FastLockGuard&) = delete;
    FastLockGuard& operator=(const FastLockGuard&) = delete;
};

} // namespace yoomates

#endif // YOOMATES_LOCK_HPP
