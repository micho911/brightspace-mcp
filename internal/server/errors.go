package server

import (
	"errors"
	"fmt"

	"github.com/micho911/brightspace-mcp/internal/auth"
	"github.com/micho911/brightspace-mcp/internal/brightspace"
)

// sessionError turns a missing or expired session into an instruction the
// assistant can pass on to the user. Other errors are returned unchanged.
func sessionError(err error, baseURL string) error {
	switch {
	case errors.Is(err, auth.ErrNoSession):
		return errors.New("not logged in to Brightspace: ask the user to run `brightspace-mcp login <their Brightspace URL>` in a terminal, then try again")
	case errors.Is(err, brightspace.ErrSessionExpired):
		return fmt.Errorf("the Brightspace session has expired: ask the user to run `brightspace-mcp login %s` in a terminal, then try again", baseURL)
	}
	return err
}

// errCourseRequired is returned when a course tool gets no course ID.
var errCourseRequired = errors.New("courseId is required: get it from list_courses")

// courseError turns an error from a course-scoped call into a message the
// assistant can act on. what names the thing asked for, e.g. "grades".
func courseError(err error, baseURL string, courseID int64, what string) error {
	switch {
	case errors.Is(err, brightspace.ErrNotFound):
		return fmt.Errorf("course %d not found, or the user cannot see its %s: check the ID with list_courses", courseID, what)
	case errors.Is(err, brightspace.ErrForbidden):
		return fmt.Errorf("Brightspace does not let the user see %s in course %d: the teacher may have turned it off or hidden it", what, courseID)
	}
	return sessionError(err, baseURL)
}

// clampLimit applies a default and a maximum to a caller's limit.
func clampLimit(n, def, maxN int) int {
	if n <= 0 {
		n = def
	}
	return min(n, maxN)
}
