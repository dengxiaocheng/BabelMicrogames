package issuebridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type GitHubClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

type GitHubUser struct {
	Login string `json:"login"`
}

func (c *GitHubClient) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *GitHubClient) requestJSON(method, path string, payload any, out any) error {
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultAPIBaseURL
	}
	body := io.Reader(nil)
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	request, err := http.NewRequest(method, baseURL+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "babel-runtime-codex-issue-bridge")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(c.Token) != "" {
		request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.Token))
	}

	response, err := c.httpClient().Do(request)
	if err != nil {
		return fmt.Errorf("GitHub API 连接失败: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode >= 300 {
		return fmt.Errorf("GitHub API 失败 %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	if out != nil && len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, out); err != nil {
			return err
		}
	}
	return nil
}

func (c *GitHubClient) paginatedGet(path string, out any) error {
	resultsValue := []json.RawMessage{}
	page := 1
	for {
		pagePath, err := appendPage(path, page)
		if err != nil {
			return err
		}
		var pageItems []json.RawMessage
		if err := c.requestJSON(http.MethodGet, pagePath, nil, &pageItems); err != nil {
			return err
		}
		if len(pageItems) == 0 {
			break
		}
		resultsValue = append(resultsValue, pageItems...)
		if len(pageItems) < 100 {
			break
		}
		page++
	}
	raw, err := json.Marshal(resultsValue)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func (c *GitHubClient) GetUser() (GitHubUser, error) {
	var user GitHubUser
	return user, c.requestJSON(http.MethodGet, "/user", nil, &user)
}

func (c *GitHubClient) CreateIssue(repoFullName string, payload CreateIssueRequest) (Issue, error) {
	var issue Issue
	return issue, c.requestJSON(http.MethodPost, "/repos/"+repoFullName+"/issues", payload, &issue)
}

func (c *GitHubClient) GetIssue(repoFullName string, issueNumber int) (Issue, error) {
	var issue Issue
	return issue, c.requestJSON(http.MethodGet, fmt.Sprintf("/repos/%s/issues/%d", repoFullName, issueNumber), nil, &issue)
}

func (c *GitHubClient) CreateIssueComment(repoFullName string, issueNumber int, body string) (Comment, error) {
	var comment Comment
	payload := map[string]string{"body": body}
	return comment, c.requestJSON(http.MethodPost, fmt.Sprintf("/repos/%s/issues/%d/comments", repoFullName, issueNumber), payload, &comment)
}

func (c *GitHubClient) CloseIssue(repoFullName string, issueNumber int) error {
	payload := map[string]string{
		"state":        "closed",
		"state_reason": "completed",
	}
	return c.requestJSON(http.MethodPatch, fmt.Sprintf("/repos/%s/issues/%d", repoFullName, issueNumber), payload, nil)
}

func (c *GitHubClient) ListIssueComments(repoFullName string, issueNumber int) ([]Comment, error) {
	var comments []Comment
	path := fmt.Sprintf("/repos/%s/issues/%d/comments?per_page=100", repoFullName, issueNumber)
	return comments, c.paginatedGet(path, &comments)
}

func appendPage(path string, page int) (string, error) {
	parsed, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("page", fmt.Sprintf("%d", page))
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
