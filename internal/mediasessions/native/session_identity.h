#pragma once

#include <unknwn.h>
#include <winrt/base.h>

#include <cstdint>
#include <string>
#include <vector>

// Only canonical IUnknown identity is stable across different COM interfaces.
// Entries retain that reference, preventing address reuse while a token is live.
template <typename Entry>
std::string mediaInstance(winrt::com_ptr<::IUnknown> const& identity,
                          std::vector<Entry> const& previous, uint64_t& next) {
    for (auto const& entry : previous) {
        if (entry.identity == identity) return entry.instance;
    }
    return std::to_string(++next);
}
