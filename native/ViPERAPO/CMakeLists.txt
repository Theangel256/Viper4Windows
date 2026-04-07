cmake_minimum_required(VERSION 3.16)
project(ViPERAPO LANGUAGES C CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_WINDOWS_EXPORT_ALL_SYMBOLS ON)

option(USE_SYSTEM_VIPERDSP "Link against a local ViPERDSP checkout" ON)

if(USE_SYSTEM_VIPERDSP)
  # Expect ViPERDSP to be available next to the project root as ../ViPERDSP
  if(EXISTS "${CMAKE_SOURCE_DIR}/../../ViPERDSP/CMakeLists.txt")
    add_subdirectory(${CMAKE_SOURCE_DIR}/../../ViPERDSP ${CMAKE_BINARY_DIR}/ViPERDSP)
  else()
    message(WARNING "ViPERDSP not found at ../ViPERDSP — build will produce a stub DLL.")
  endif()
endif()

# If no external ViPERDSP target was added, provide a small internal implementation
if(NOT TARGET ViPERDSP)
  add_library(ViPERDSP STATIC
    src/ViPERDSP.cpp
  )
  target_include_directories(ViPERDSP PUBLIC ${CMAKE_SOURCE_DIR}/include)
  message(STATUS "Using internal ViPERDSP implementation for prototyping.")
endif()

add_library(Hydrogen_Inst SHARED
  src/ViperAPO.cpp
  src/APOSkeleton.cpp
)

add_library(SharedMemReader STATIC
  src/SharedMemReader.cpp
)

add_executable(viper_shim_monitor
  src/monitor.cpp
  src/SharedMemReader.cpp
)

if(MSVC)
  target_link_libraries(viper_shim_monitor PRIVATE ws2_32)
endif()

if(TARGET ViPERDSP)
  target_link_libraries(viper_shim_monitor PRIVATE ViPERDSP)
endif()

target_include_directories(Hydrogen_Inst PRIVATE
  ${CMAKE_SOURCE_DIR}/include
)

target_link_libraries(Hydrogen_Inst PRIVATE SharedMemReader)

if(TARGET ViPERDSP)
  target_link_libraries(Hydrogen_Inst PRIVATE ViPERDSP)
endif()

if(MSVC)
  target_compile_options(Hydrogen_Inst PRIVATE /W3 /WX-)
else()
  target_compile_options(Hydrogen_Inst PRIVATE -Wall -Wextra)
endif()

set_target_properties(Hydrogen_Inst PROPERTIES
  OUTPUT_NAME "Hydrogen_Inst"
)

message(STATUS "ViPER APO scaffolding configured. Build with a Visual Studio generator or Ninja.")
