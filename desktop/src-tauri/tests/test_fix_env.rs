#[cfg(test)]
mod tests {
    #[test]
    fn test_fix_env() {
        std::env::set_var("_TEST_VARIABLE", "test_value");
        fix_env::fix_env("_TEST_VARIABLE").unwrap();
        assert_eq!(std::env::var("_TEST_VARIABLE").unwrap(), "test_value");
    }

    #[test]
    fn test_fix_env_multiline() {
        std::env::set_var("_TEST_VARIABLE", "test_value\nnew_line\nanother_line");
        fix_env::fix_env().unwrap();
        assert_eq!(std::env::var("_TEST_VARIABLE").unwrap(), "test_value\nnew_line\nanother_line");
    }
}