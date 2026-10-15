#include "session_identity.h"

#include <winrt/Windows.Foundation.h>

#include <cstdio>
#include <stdexcept>

using winrt::Windows::Foundation::IClosable;
using winrt::Windows::Foundation::IStringable;

void require(bool condition, char const* message) {
    if (!condition) throw std::runtime_error(message);
}

struct FakeSession : winrt::implements<FakeSession, IStringable, IClosable> {
    int& released;
    explicit FakeSession(int& count) : released(count) {}
    ~FakeSession() { ++released; }
    winrt::hstring ToString() const { return L"same app"; }
    void Close() const {}
};

struct Entry {
    winrt::com_ptr<::IUnknown> identity;
    std::string instance;
};

int main() {
    try {
        int released = 0;
        uint64_t next = 0;
        auto first = winrt::make<FakeSession>(released);
        auto second = winrt::make<FakeSession>(released);
        auto firstIdentity = first.as<::IUnknown>();
        require(firstIdentity == first.as<IClosable>().as<::IUnknown>(), "interface changed canonical identity");
        std::vector<Entry> current;
        auto a = mediaInstance(firstIdentity, current, next);
        auto b = mediaInstance(second.as<::IUnknown>(), current, next);
        require(a != b, "distinct sessions reused a token");
        current = {{firstIdentity, a}, {second.as<::IUnknown>(), b}};
        require(mediaInstance(second.as<::IUnknown>(), current, next) == b, "enumeration reorder changed token");
        require(mediaInstance(first.as<IClosable>().as<::IUnknown>(), current, next) == a, "same COM identity changed token");

        // A sizing/failed snapshot builds candidates without replacing current.
        auto third = winrt::make<FakeSession>(released);
        auto c = mediaInstance(third.as<::IUnknown>(), current, next);
        require(c != a && c != b && current.size() == 2, "candidate changed committed identities");
        require(mediaInstance(firstIdentity, current, next) == a, "failed snapshot lost previous identity");

        current.erase(current.begin());
        require(mediaInstance(firstIdentity, current, next) != a, "absent session reused retired token");
        first = nullptr;
        firstIdentity = nullptr;
        require(released == 1, "removed identity reference leaked");
        second = nullptr;
        require(released == 1, "committed snapshot did not retain identity");
        current.clear();
        require(released == 2, "snapshot close did not release identity");
        third = nullptr;
        require(released == 3, "candidate identity reference leaked");
        std::puts("Media identity self-test passed (canonical identity, reorder, removal, retry, release).");
        return 0;
    } catch (std::exception const& error) {
        std::fprintf(stderr, "Media identity self-test failed: %s\n", error.what());
        return 1;
    } catch (winrt::hresult_error const& error) {
        std::fprintf(stderr, "Media identity self-test failed: HRESULT 0x%08X\n", static_cast<unsigned>(error.code()));
        return 1;
    }
}
