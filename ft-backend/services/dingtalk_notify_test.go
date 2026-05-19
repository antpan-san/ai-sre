package services

import (
	"strings"
	"testing"

	"ft-backend/models"
)

func TestEnsureDingTalkKeyword(t *testing.T) {
	got := ensureDingTalkKeyword("hello", "操")
	if !strings.Contains(got, "操") {
		t.Fatalf("missing keyword: %q", got)
	}
	dup := ensureDingTalkKeyword("已有操内容", "操")
	if dup != "已有操内容" {
		t.Fatalf("duplicate keyword: %q", dup)
	}
}

func TestBuildAutoIterationDingTalkMarkdownContainsKeywordHook(t *testing.T) {
	title, md := buildAutoIterationDingTalkMarkdown(DingTalkKindWorkerCompleted, autoIterationDingTalkFields{
		Title:  "配置钉钉",
		TaskID: "8bcf4c96-145e-4786-9dfe-06d2ec78b803",
		Source: "页面手动",
		Topic:  "auto dev",
		Status: models.AutoIterationStatusCompleted,
		Summary: "实现完成",
	})
	if title == "" || !strings.Contains(md, "配置钉钉") {
		t.Fatalf("title=%q md=%q", title, md)
	}
	if !strings.Contains(md, "8bcf4c96") {
		t.Fatal("missing task id")
	}
	withKw := ensureDingTalkKeyword(md, "操")
	if !strings.Contains(withKw, "操") {
		t.Fatal("keyword not injected")
	}
}

func TestEscapeDingTalkMarkdownInline(t *testing.T) {
	got := escapeDingTalkMarkdownInline("a*b`c#d")
	if got != "a\\*b'c\\#d" {
		t.Fatalf("escape: %q", got)
	}
}
