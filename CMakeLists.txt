cmake_minimum_required(VERSION 3.16)
project(TimeKeeper CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

add_library(tk_core INTERFACE)
target_include_directories(tk_core INTERFACE
    ${CMAKE_CURRENT_SOURCE_DIR}/core)
find_package(nlohmann_json 3.2.0 REQUIRED)
target_link_libraries(tk_core INTERFACE nlohmann_json::nlohmann_json)

find_package(PkgConfig REQUIRED)
pkg_check_modules(GTK4 REQUIRED IMPORTED_TARGET gtk4)
find_package(CURL REQUIRED)

add_executable(timekeeper-gtk gtk/main.cpp)
target_include_directories(timekeeper-gtk PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/core)
target_link_libraries(timekeeper-gtk PRIVATE tk_core PkgConfig::GTK4 CURL::libcurl)