package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const gradeValuesJSON = `[
	{"GradeObjectName":"Midterm","DisplayedGrade":"82 %","PointsNumerator":41,"PointsDenominator":50,
	 "Comments":{"Text":"","Html":"<p>Good <b>work</b></p>"},"LastModified":"2026-09-20T10:00:00.000Z"},
	null,
	{"GradeObjectName":"Participation","DisplayedGrade":"Pass","PointsNumerator":null,"PointsDenominator":null,"Comments":null,"LastModified":null},
	{"GradeObjectName":"Not graded","DisplayedGrade":"","PointsNumerator":null,"PointsDenominator":10,"Comments":{"Text":"","Html":""}},
	{"GradeObjectName":"Quiz","DisplayedGrade":"","PointsNumerator":1,"PointsDenominator":3,"Comments":"plain"}]`

func fakeGrades(t *testing.T, final string) Connect {
	return fakeBrightspace(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/1001/grades/values/myGradeValues/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(gradeValuesJSON))
		case strings.HasSuffix(r.URL.Path, "/1001/grades/final/values/myGradeValue") && final != "":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(final))
		default:
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func callGetGrades(t *testing.T, bs Connect, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := connect(t, bs).CallTool(context.Background(), &mcp.CallToolParams{Name: "get_grades", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func grades(t *testing.T, res *mcp.CallToolResult) Grades {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var g Grades
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return g
}

func TestGetGrades(t *testing.T) {
	g := grades(t, callGetGrades(t, fakeGrades(t, `{"GradeObjectName":"","DisplayedGrade":"B","PointsNumerator":null,"PointsDenominator":null}`), map[string]any{"courseId": 1001}))

	var names []string
	for _, e := range g.Items {
		names = append(names, e.Name)
	}
	if got, want := strings.Join(names, ","), "Midterm,Participation,Quiz"; got != want {
		t.Fatalf("items = %s, want %s (null and ungraded dropped)", got, want)
	}
	mid := g.Items[0]
	if mid.Grade != "82 %" || *mid.Points != 41 || *mid.OutOf != 50 || *mid.Percent != 82 || mid.Comment != "Good work" || mid.Updated != "2026-09-20T10:00:00Z" {
		t.Errorf("midterm = %+v", mid)
	}
	if pass := g.Items[1]; pass.Grade != "Pass" || pass.Points != nil || pass.Percent != nil {
		t.Errorf("participation = %+v", pass)
	}
	if quiz := g.Items[2]; quiz.Grade != "" || *quiz.Percent != 33.3 || quiz.Comment != "plain" {
		t.Errorf("quiz = %+v, want points-only entry with percent 33.3", quiz)
	}
	if g.Final == nil || g.Final.Grade != "B" || g.Final.Name != "Final grade" {
		t.Errorf("final = %+v", g.Final)
	}
	if !strings.HasSuffix(g.URL, "/d2l/lms/grades/my_grades/main.d2l?ou=1001") {
		t.Errorf("url = %q", g.URL)
	}
}

func TestGetGradesWithoutFinalGrade(t *testing.T) {
	g := grades(t, callGetGrades(t, fakeGrades(t, ""), map[string]any{"courseId": 1001}))

	if g.Final != nil || len(g.Items) != 3 {
		t.Errorf("grades = %+v, want the items and no final grade", g)
	}
}

func TestGetGradesErrors(t *testing.T) {
	bs := fakeGrades(t, "")
	if msg := errorText(t, callGetGrades(t, bs, map[string]any{"courseId": 0})); !strings.Contains(msg, "courseId is required") {
		t.Errorf("missing courseId: error %q", msg)
	}
	if msg := errorText(t, callGetGrades(t, bs, map[string]any{"courseId": 42})); !strings.Contains(msg, "course 42 not found") {
		t.Errorf("unknown course: error %q", msg)
	}
}
