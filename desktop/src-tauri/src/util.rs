use std::time::{Duration, Instant};

/// `measure`  the duration it took a function to execute.
#[allow(dead_code)]
pub fn measure<F>(f: F) -> Duration
where
    F: Fn(),
{
    let start = Instant::now();
    f();
    let duration = start.elapsed();

    duration
}
