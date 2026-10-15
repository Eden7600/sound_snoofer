#pragma once

#include <windows.h>
#include <winrt/Windows.Foundation.h>

#include <chrono>

template <typename Operation>
auto mediaResult(Operation const& operation) {
    if (operation.wait_for(std::chrono::seconds(2)) == winrt::Windows::Foundation::AsyncStatus::Started) {
        try {
            operation.Cancel();
        } catch (winrt::hresult_error const&) {
            // A rejected cancellation must still report the timeout that caused it.
        }
        throw winrt::hresult_error(HRESULT_FROM_WIN32(ERROR_TIMEOUT));
    }
    return operation.get();
}
