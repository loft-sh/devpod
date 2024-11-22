use std::time::{Duration, Instant};
use log::{info, error};

// Exit code for the window to signal that the application was quit by the user through the system tray
// and event handlers may not use prevent_exit().
pub const QUIT_EXIT_CODE: i32 = 1337;

/// `measure`  the duration it took a function to execute.
#[allow(dead_code)]
pub fn measure<F>(f: F) -> Duration
where
    F: Fn(),
{
    let start = Instant::now();
    f();

    start.elapsed()
}

/// Kills all child processes of a pid on windows, does nothing on all the other OSs.
pub fn kill_child_processes(parent_pid: u32) {
    #[cfg(windows)]
    {
        use windows::Win32::System::Threading::*;
        use windows::Win32::System::Diagnostics::ToolHelp::*;
        use windows::Win32::Foundation::*;

        info!("Trying to kill child processes of PID {}.", parent_pid);

        let snapshot: HANDLE = unsafe { CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0).unwrap() };

        if snapshot.is_invalid() {
            info!("Failed to take process snapshot.");
            return;
        }

        info!("Obtained process snapshot.");

        let mut process_entry: PROCESSENTRY32 = unsafe { std::mem::zeroed() };
        process_entry.dwSize = std::mem::size_of::<PROCESSENTRY32>() as u32;

        unsafe {
            if Process32First(snapshot, &mut process_entry).as_bool() {
                loop {
                    // Check if the process we're looking at is a *direct* child process.
                    if process_entry.th32ParentProcessID == parent_pid {
                        let pid = process_entry.th32ProcessID;

                        // Extract zero-terminated string for the executable.
                        let exe_name = process_entry.szExeFile.iter()
                            .take_while(|&&ch| ch != 0)
                            .map(|&ch| ch as u8 as char)
                            .collect::<String>();

                        info!(
                            "Found process with PID {} as child of PID {} ({}).",
                            pid,
                            parent_pid,
                            exe_name,
                        );

                        // Special exception: We do not clean up tauri's webviews. For now.
                        if exe_name == "msedgewebview2.exe" {
                            info!("Ignoring process PID {}.", pid);
                        } else {
                            // Recursively terminate children of children.
                            kill_child_processes(pid);

                            // Obtain handle for the child process.
                            let child_process_handle: windows::core::Result<HANDLE> = OpenProcess(PROCESS_TERMINATE, false, pid);

                            if child_process_handle.is_err() {
                                error!("Unable to open process {}: {:?}", pid, child_process_handle.unwrap_err());
                            } else {
                                let child_process_handle: HANDLE = child_process_handle.unwrap();

                                // Attempt to terminate the child process.
                                let close_result = TerminateProcess(child_process_handle, 1);

                                // Clean up the handle.
                                CloseHandle(child_process_handle);

                                if !close_result.as_bool() {
                                    error!("Unable to terminate process {}", pid);
                                } else {
                                    info!("Terminated process {}.", pid);
                                }
                            }
                        }
                    }

                    // Obtain next process or end the loop if there is none available.
                    if !Process32Next(snapshot, &mut process_entry).as_bool() {
                        break;
                    }
                }
            }

            // Clean up the snapshot.
            CloseHandle(snapshot);
        }
    }
}
