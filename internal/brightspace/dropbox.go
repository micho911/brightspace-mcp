package brightspace

import (
	"context"
	"strconv"
	"time"
)

// FileRef is a file attached to something in Brightspace.
type FileRef struct {
	ID   int64  `json:"FileId"`
	Name string `json:"FileName"`
	Size int64  `json:"Size"`
}

// DropboxFolder is an assignment folder.
type DropboxFolder struct {
	ID             int64      `json:"Id"`
	Name           string     `json:"Name"`
	Instructions   RichText   `json:"CustomInstructions"`
	Attachments    []FileRef  `json:"Attachments"`
	DueDate        *time.Time `json:"DueDate"`
	IsHidden       bool       `json:"IsHidden"`
	IsAnonymous    bool       `json:"IsAnonymous"`
	GroupTypeID    *int64     `json:"GroupTypeId"`
	GradeItemID    *int64     `json:"GradeItemId"`
	SubmissionType FlexText   `json:"SubmissionType"`
	CompletionType FlexText   `json:"CompletionType"`
	Availability   struct {
		StartDate *time.Time `json:"StartDate"`
		EndDate   *time.Time `json:"EndDate"`
	} `json:"Availability"`
	Assessment struct {
		ScoreDenominator *float64 `json:"ScoreDenominator"`
	} `json:"Assessment"`
}

// DropboxFeedback is the teacher's feedback on a submission.
type DropboxFeedback struct {
	Score       *float64  `json:"Score"`
	Feedback    RichText  `json:"Feedback"`
	IsGraded    bool      `json:"IsGraded"`
	GradedLabel FlexText  `json:"GradedSymbol"`
	Files       []FileRef `json:"Files"`
}

// DropboxEntity is the user's (or their group's) standing in one folder.
type DropboxEntity struct {
	Entity struct {
		ID          int64  `json:"EntityId"`
		DisplayName string `json:"DisplayName"`
	} `json:"Entity"`
	// Status is 0 unsubmitted, 1 submitted, 2 draft, 3 feedback published.
	Status         FlexText            `json:"Status"`
	Feedback       *DropboxFeedback    `json:"Feedback"`
	CompletionDate *time.Time          `json:"CompletionDate"`
	Submissions    []DropboxSubmission `json:"Submissions"`
}

// DropboxSubmission is one submission made by the user.
type DropboxSubmission struct {
	ID      int64      `json:"Id"`
	Date    *time.Time `json:"SubmissionDate"`
	Comment RichText   `json:"Comment"`
	Files   []FileRef  `json:"Files"`
}

func dropboxPath(orgUnitID int64) string {
	return "/d2l/api/le/" + leVersion + "/" + strconv.FormatInt(orgUnitID, 10) + "/dropbox/folders/"
}

// DropboxFolders returns the assignment folders in a course.
func (c *Client) DropboxFolders(ctx context.Context, orgUnitID int64) ([]DropboxFolder, error) {
	var folders []DropboxFolder
	err := c.get(ctx, dropboxPath(orgUnitID), &folders)
	return folders, err
}

// DropboxFolder returns one assignment folder.
func (c *Client) DropboxFolder(ctx context.Context, orgUnitID, folderID int64) (DropboxFolder, error) {
	var folder DropboxFolder
	err := c.get(ctx, dropboxPath(orgUnitID)+strconv.FormatInt(folderID, 10), &folder)
	return folder, err
}

// MySubmissions returns the user's own submissions in a folder (one entity
// for the user, or for the user's group in a group folder).
func (c *Client) MySubmissions(ctx context.Context, orgUnitID, folderID int64) ([]DropboxEntity, error) {
	var entities []DropboxEntity
	err := c.get(ctx, dropboxPath(orgUnitID)+strconv.FormatInt(folderID, 10)+"/submissions/mysubmissions/", &entities)
	return entities, err
}
