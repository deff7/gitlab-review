package main

import (
	"time"

	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
)

type gitlabClient struct {
	client *gitlab.Client

	projectID      string
	mergeRequestID int

	// state
	baseSHA, headSHA, startSHA string
}

func newClient(token, baseURL string) *gitlabClient {
	c := gitlab.NewClient(nil, token)
	if baseURL != "" {
		c.SetBaseURL(baseURL)
	}

	return &gitlabClient{
		client: c,
	}
}

func (c *gitlabClient) init(projectID string, mergeRequestID int) error {
	mr, _, err := c.client.MergeRequests.GetMergeRequest(projectID, mergeRequestID, nil)
	if err != nil {
		return errors.Wrap(err, "get merge request info")
	}
	c.projectID = projectID
	c.mergeRequestID = mergeRequestID
	c.baseSHA = mr.DiffRefs.BaseSha
	c.headSHA = mr.DiffRefs.HeadSha
	c.startSHA = mr.DiffRefs.StartSha
	return nil
}

func (c *gitlabClient) pushComment(fileName string, comm comment) error {
	now := time.Now()
	opt := gitlab.CreateMergeRequestDiscussionOptions{
		CreatedAt: &now,
		Body:      &comm.text,
		Position: &gitlab.NotePosition{
			BaseSHA:      c.baseSHA,
			StartSHA:     c.startSHA,
			HeadSHA:      c.headSHA,
			PositionType: "text",
			NewPath:      fileName,
			OldPath:      fileName,
			NewLine:      comm.line,
		},
	}

	_, _, err := c.client.Discussions.CreateMergeRequestDiscussion(c.projectID, c.mergeRequestID, &opt)
	if err != nil {
		return errors.Wrap(err, "push comment")
	}
	return nil
}
