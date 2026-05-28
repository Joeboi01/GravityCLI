package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDashboardHandlesBackMsgBeforeDelegatingToSubview(t *testing.T) {
	model := NewDashboardModel()
	model.active = viewClone

	updated, cmd := model.Update(BackMsg{})
	got := updated.(DashboardModel)

	if got.active != viewDashboard {
		t.Fatalf("active view = %v, want %v", got.active, viewDashboard)
	}
	if !got.loading {
		t.Fatal("expected dashboard to reload profile after returning")
	}
	if cmd == nil {
		t.Fatal("expected profile reload command")
	}
}

func TestCloneSuccessQReturnsBackMsg(t *testing.T) {
	model := NewCloneModel()
	model.step = cloneStepSuccess

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected q on clone success to return a command")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatal("expected q on clone success to return BackMsg")
	}
}

func TestPRSuccessQReturnsBackMsg(t *testing.T) {
	model := NewPRModel()
	model.step = prStepSuccess

	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("expected q on PR success to return a command")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Fatal("expected q on PR success to return BackMsg")
	}
}
