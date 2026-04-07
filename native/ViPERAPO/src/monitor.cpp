#include <iostream>
#include <csignal>
#include <atomic>
#include <thread>
#include <chrono>

extern "C" int StartSharedMemMonitor(int pollMs);
extern "C" int StopSharedMemMonitor();

static std::atomic_bool stopFlag{false};

void signalHandler(int signum) {
    (void)signum;
    stopFlag.store(true);
    StopSharedMemMonitor();
}

int main(int argc, char** argv) {
    int pollMs = 100;
    if (argc > 1) {
        try { pollMs = std::stoi(argv[1]); } catch (...) {}
    }

    std::signal(SIGINT, signalHandler);
    std::cout << "Viper4Windows SHIM monitor starting (poll=" << pollMs << "ms)" << std::endl;
    StartSharedMemMonitor(pollMs);

    // Blocking wait
    while (!stopFlag.load()) std::this_thread::sleep_for(std::chrono::milliseconds(250));
    std::cout << "Stopping monitor..." << std::endl;
    StopSharedMemMonitor();
    return 0;
}
