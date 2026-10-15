#include "async_wait.h"

#include <cstdio>
#include <initializer_list>
#include <stdexcept>

using winrt::Windows::Foundation::AsyncStatus;
using winrt::Windows::Foundation::TimeSpan;

void require(bool condition, char const* message) {
    if (!condition) throw std::runtime_error(message);
}

// A running fake refuses get(): reaching it would block on a frozen player.
struct FakeOperation {
    AsyncStatus status;
    bool rejectCancel = false;
    mutable bool waited = false, cancelled = false, read = false;

    AsyncStatus wait_for(TimeSpan timeout) const {
        require(timeout == std::chrono::seconds(2), "wait must be exactly two seconds");
        waited = true;
        return status;
    }

    void Cancel() const {
        cancelled = true;
        if (rejectCancel) throw winrt::hresult_error(E_ACCESSDENIED);
    }

    int get() const {
        read = true;
        switch (status) {
        case AsyncStatus::Completed: return 7;
        case AsyncStatus::Error: throw winrt::hresult_error(E_FAIL);
        case AsyncStatus::Canceled: throw winrt::hresult_canceled();
        default: throw std::runtime_error("get() reached a still-running operation");
        }
    }
};

void expectError(FakeOperation const& operation, HRESULT expected) {
    try {
        (void)mediaResult(operation);
    } catch (winrt::hresult_error const& error) {
        require(error.code() == expected, "wrong HRESULT returned");
        return;
    }
    throw std::runtime_error("operation did not report an error");
}

int main() {
    try {
        FakeOperation complete{AsyncStatus::Completed};
        require(mediaResult(complete) == 7, "completed result changed");
        require(complete.read && !complete.cancelled, "completed operation was cancelled");

        FakeOperation failed{AsyncStatus::Error};
        expectError(failed, E_FAIL);
        require(failed.read && !failed.cancelled, "failed operation lost its result");

        FakeOperation cancelled{AsyncStatus::Canceled};
        expectError(cancelled, HRESULT_FROM_WIN32(ERROR_CANCELLED));
        require(cancelled.read && !cancelled.cancelled, "cancelled operation was cancelled again");

        for (bool rejectCancel : {false, true}) {
            FakeOperation running{AsyncStatus::Started, rejectCancel};
            expectError(running, HRESULT_FROM_WIN32(ERROR_TIMEOUT));
            require(running.waited && running.cancelled && !running.read, "timeout did not cancel without reading");
        }
        require(complete.waited && failed.waited && cancelled.waited, "an async result bypassed the bounded wait");
        std::puts("Media async wait self-test passed (completed, failed, cancelled, timeout).");
        return 0;
    } catch (std::exception const& error) {
        std::fprintf(stderr, "Media async wait self-test failed: %s\n", error.what());
        return 1;
    } catch (winrt::hresult_error const& error) {
        std::fprintf(stderr, "Media async wait self-test failed: HRESULT 0x%08X\n", static_cast<unsigned>(error.code()));
        return 1;
    }
}
