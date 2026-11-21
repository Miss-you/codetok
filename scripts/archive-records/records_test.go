package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestRenderParseRoundTrip(t *testing.T) {
	in := Records{
		TaskID: "ENG-012",
		Paths:  []string{"workspace/ENG-012/", "docs/plans/2026-06-13-engineering-roadmap.md", "openspec/changes/foo/proposal.md"},
		Inline: []InlineFile{
			{Name: "notes.md", Content: "line one\n- `workspace/not-a-listed-path.md`\n~~~~~\n```go\nx := 1\n```\n"},
			{Name: "no-newline.txt", Content: "tail"},
			{Name: "empty.txt", Content: ""},
		},
	}
	body, err := RenderComment(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(body, Marker("ENG-012")+"\n") {
		t.Fatalf("body does not start with the marker:\n%s", body)
	}
	got, ok, err := ParseComment(body)
	if err != nil || !ok {
		t.Fatalf("ParseComment: ok=%v err=%v\n%s", ok, err, body)
	}
	want := in
	want.Inline = []InlineFile{in.Inline[0], {Name: "no-newline.txt", Content: "tail\n"}, {Name: "empty.txt", Content: "\n"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestParseCommentCRLFAndBacktickFence(t *testing.T) {
	body := "<!-- codetok-records:EBTA-001 -->\r\nFiles:\r\n* `workspace/EBTA-001/final_impl.md` (design)\r\n\r\n<details><summary>inline: a.md</summary>\r\n\r\n```markdown\r\nhello\r\n```\r\n\r\n</details>\r\n"
	got, ok, err := ParseComment(body)
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if !reflect.DeepEqual(got.Paths, []string{"workspace/EBTA-001/final_impl.md"}) || got.Inline[0].Content != "hello\n" {
		t.Fatalf("got %#v", got)
	}
}

func TestParseCommentNoMarker(t *testing.T) {
	_, ok, err := ParseComment("AI review: abc\n- `workspace/x/`")
	if ok || err != nil {
		t.Fatalf("ok=%v err=%v, want no marker and no error", ok, err)
	}
}

func TestInlineMarkerMentionIsNotARecord(t *testing.T) {
	for _, body := range []string{
		"Use `<!-- codetok-records:ENG-012 -->` as the marker.",
		"> quoted <!-- codetok-records:ENG-012 -->\n- `workspace/ENG-012/`",
	} {
		if _, ok, err := ParseComment(body); ok || err != nil {
			t.Errorf("ParseComment(%q): ok=%v err=%v, want no marker", body, ok, err)
		}
	}
	real := Marker("ENG-012") + "\n- `workspace/ENG-012/`"
	mention := "AI review: see `<!-- codetok-records:ENG-012 -->`"
	if _, _, found, _, err := FindRecords([]Comment{comment(1, "OWNER", real), comment(2, "OWNER", mention)}); !found || err != nil {
		t.Fatalf("a mention in another comment broke FindRecords: found=%v err=%v", found, err)
	}
}

func TestParseCommentErrors(t *testing.T) {
	cases := map[string]string{
		"two markers":       Marker("A") + "\n" + Marker("B") + "\n- `workspace/A/`",
		"bad task id":       "<!-- codetok-records:../x -->\n- `workspace/x/`",
		"no files":          Marker("A") + "\nnothing listed",
		"parent dir":        Marker("A") + "\n- `workspace/../cmd/root.go`",
		"absolute":          Marker("A") + "\n- `/etc/passwd`",
		"outside allowlist": Marker("A") + "\n- `cmd/root.go`",
		"archive itself":    Marker("A") + "\n- `docs/changes/A/README.md`",
		"prefix only":       Marker("A") + "\n- `workspace/`",
		"wildcard":          Marker("A") + "\n- `workspace/*.md`",
		"pathspec magic":    Marker("A") + "\n- `:(glob)workspace/x`",
		"dot segment":       Marker("A") + "\n- `workspace/./x.md`",
		"duplicate":         Marker("A") + "\n- `workspace/x.md`\n- `workspace/x.md`",
		"unclosed fence":    Marker("A") + "\n<details><summary>inline: a.md</summary>\n\n~~~~\nhello\n",
		"bad inline name":   Marker("A") + "\n<details><summary>inline: ../a.md</summary>\n\n~~~~\nhi\n~~~~\n\n</details>\n",
		"missing details":   Marker("A") + "\n<details><summary>inline: a.md</summary>\n\n~~~~\nhi\n~~~~\ntrailing\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			_, ok, err := ParseComment(body)
			if !ok || err == nil {
				t.Fatalf("ok=%v err=%v, want a marker with an error", ok, err)
			}
		})
	}
}

func comment(id int64, assoc, body string) Comment {
	c := Comment{ID: id, Body: body, AuthorAssociation: assoc}
	c.User.Login = "user" + assoc
	return c
}

func TestFindRecords(t *testing.T) {
	good := Marker("ENG-012") + "\n- `workspace/ENG-012/`"
	t.Run("none", func(t *testing.T) {
		_, _, found, _, err := FindRecords([]Comment{comment(1, "OWNER", "AI review: abc")})
		if found || err != nil {
			t.Fatalf("found=%v err=%v", found, err)
		}
	})
	t.Run("untrusted marker is ignored", func(t *testing.T) {
		_, _, found, ignored, err := FindRecords([]Comment{comment(1, "NONE", good), comment(2, "CONTRIBUTOR", good)})
		if found || err != nil || len(ignored) != 2 {
			t.Fatalf("found=%v err=%v ignored=%v", found, err, ignored)
		}
	})
	t.Run("one trusted among untrusted", func(t *testing.T) {
		r, c, found, _, err := FindRecords([]Comment{comment(1, "NONE", Marker("EVIL")+"\n- `workspace/x/`"), comment(2, "OWNER", good)})
		if !found || err != nil || r.TaskID != "ENG-012" || c.ID != 2 {
			t.Fatalf("r=%v c=%v found=%v err=%v", r, c.ID, found, err)
		}
	})
	t.Run("two trusted is an error", func(t *testing.T) {
		_, _, found, _, err := FindRecords([]Comment{comment(1, "OWNER", good), comment(2, "MEMBER", good)})
		if !found || err == nil {
			t.Fatalf("found=%v err=%v", found, err)
		}
	})
	t.Run("malformed trusted is an error", func(t *testing.T) {
		_, _, found, _, err := FindRecords([]Comment{comment(1, "COLLABORATOR", Marker("A")+"\n- `cmd/x.go`")})
		if !found || err == nil {
			t.Fatalf("found=%v err=%v", found, err)
		}
	})
}
