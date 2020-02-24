# GitLab review helper

This utility helps you to effectively conduct a code review using your favorite text editor instead of GitLab web interface

## Installation

1. `go get -u github.com/deff7/gitlab-review`
2. Create personal [GitLab token](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html)
3. Set up environment variables:
  1. `GITLAB_TOKEN` with your personal access token
  2. `GITLAB_BASE_URL` with URL of your GitLab server

## Workflow

1. Your peer sends you a link to a merge request
2. Git checkout to a base branch of the merge request
3. Open project in your favorite editor and start doing the code review
4. Every time you want to start a discussion leave a comment in the source code just above the current line. Start the comment with `CR` prefix:
```
func foo() {
  // CR why do you need this?
  panic(nil)
}
```
5. When you done with the code review then execute the helper in a terminal in root directory of the project: `gitlab-review <link to the merge request>`. Don't forget to provide a link to the merge request
6. In the utility window you can skip comment by pressing `n` or push it by pressing `y`. 
