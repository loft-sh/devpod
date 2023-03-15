// var workspaceIDRegEx1 = regexp.MustCompile(`[^\w\-]`)
// var workspaceIDRegEx2 = regexp.MustCompile(`[^0-9a-z\-]+`)
//
// func ToID(str string) string {

// lowercase everything and replace paths to forward slash
// 	str = strings.ToLower(filepath.ToSlash(str))

// 	// get last element if we find a /
// 	index := strings.LastIndex(str, "/")
// 	if index != -1 {
// 		str = str[index+1:]
//
// 		// remove .git if there is it
// 		str = strings.TrimSuffix(str, ".git")
//
// 		// remove a potential tag / branch name
// 		splitted := strings.Split(str, "@")
// 		if len(splitted) == 2 && !branchRegEx.MatchString(splitted[1]) {
// 			str = splitted[0]
// 		}
// 	}
//
// 	return workspaceIDRegEx2.ReplaceAllString(workspaceIDRegEx1.ReplaceAllString(str, "-"), "")
// }
//
// var branchRegEx = regexp.MustCompile(`[^a-zA-Z0-9\.\-]+`)

use lazy_static::lazy_static;
use regex::Regex;
use std::path::Path;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum NewWorkspaceIDError {
    #[error("unable to find filename for source id `${0}`")]
    NoFileName(String),
}
impl serde::Serialize for NewWorkspaceIDError {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_str(self.to_string().as_ref())
    }
}

lazy_static! {
    static ref UNSUPPORTED_SYMBOLS_REGEX: Regex = Regex::new(r"[^a-zA-Z0-9\.\-]+").unwrap();
}

// TODO: implement other cases from `devpod/pkg/workspace/workspace.go#ToId`
#[tauri::command]
pub fn new_workspace_id(source_name: String) -> Result<String, NewWorkspaceIDError> {
    let source_name = source_name.to_lowercase();
    let source_path = Path::new(&source_name);
    let name = source_path.file_stem();

    print!("{:?}", name.unwrap().to_str());

    name.and_then(|x| x.to_str())
        .and_then(|x| Some(x.to_string()))
        .and_then(|x| {
            Some(
                UNSUPPORTED_SYMBOLS_REGEX
                    .replace_all(x.as_ref(), "")
                    .to_string(),
            )
        })
        .ok_or(NewWorkspaceIDError::NoFileName(source_name))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn should_get_id() {
        let want = "foobar".to_string();
        let got = new_workspace_id("foobar".into()).unwrap();
        assert_eq!(want, got);

        let want = "vscode-course-sample";
        let got =
            new_workspace_id("https://github.com/microsoft/vscode-course-sample".into()).unwrap();
        assert_eq!(want, got);

        let want = "vscode-course-sample";
        let got = new_workspace_id("https://github.com/microsoft/vscode-course-sample.git".into())
            .unwrap();
        assert_eq!(want, got);

        let want = "test-folder".to_string();
        let got = new_workspace_id("../../test-folder".into()).unwrap();
        assert_eq!(want, got);

        #[cfg(target_os = "windows")]
        {
            let want = "test-folder".to_string();
            let got = new_workspace_id(r"..\..\test-folder".into()).unwrap();
            assert_eq!(want, got);
        }

        let want = "why-would-you-do-this".to_string();
        let got = new_workspace_id("why-!would-you-d?o-this#".into()).unwrap();
        assert_eq!(want, got);
    }
}
