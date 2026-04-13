package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/leonardomso/gone/internal/checker"
	"github.com/leonardomso/gone/internal/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_Defaults(t *testing.T) {
	t.Parallel()

	model := New("", nil, nil, false, nil, nil)
	assert.Equal(t, ".", model.path)
	assert.Equal(t, []string{"md"}, model.fileTypes)
	assert.Equal(t, stateScanning, model.state)
	assert.Equal(t, filterAll, model.filter)
}

func TestHandleFilesFound_TransitionsToExtracting(t *testing.T) {
	t.Parallel()

	model := New(".", nil, []string{"md"}, false, nil, nil)
	updated, cmd := model.handleFilesFound(FilesFoundMsg{Files: []string{"README.md"}})
	require.NotNil(t, updated)
	require.NotNil(t, cmd)
	assert.Equal(t, stateExtracting, model.state)
	assert.Equal(t, []string{"README.md"}, model.files)
}

func TestHandleLinksExtracted_AppliesFilterAndStartsChecking(t *testing.T) {
	t.Parallel()

	urlFilter, err := filter.New(filter.Config{Domains: []string{"ignored.example"}})
	require.NoError(t, err)

	model := New(".", urlFilter, []string{"md"}, false, nil, nil)
	updated, cmd := model.handleLinksExtracted(LinksExtractedMsg{
		Links: []checker.Link{
			{URL: "https://allowed.example", FilePath: "README.md", Line: 1},
			{URL: "https://ignored.example/path", FilePath: "README.md", Line: 2},
		},
	})
	require.NotNil(t, updated)
	require.NotNil(t, cmd)
	assert.Equal(t, stateChecking, model.state)
	assert.Len(t, model.links, 1)
	assert.Equal(t, 1, model.uniqueURLs)
}

func TestHandleLinkChecked_CategorizesResults(t *testing.T) {
	t.Parallel()

	model := New(".", nil, []string{"md"}, false, nil, nil)

	_, _ = model.handleLinkChecked(LinkCheckedMsg{Result: checker.Result{Status: checker.StatusAlive}})
	_, _ = model.handleLinkChecked(LinkCheckedMsg{Result: checker.Result{Status: checker.StatusRedirect}})
	_, _ = model.handleLinkChecked(LinkCheckedMsg{Result: checker.Result{Status: checker.StatusDead}})
	_, _ = model.handleLinkChecked(LinkCheckedMsg{Result: checker.Result{Status: checker.StatusDuplicate}})

	assert.Equal(t, 4, model.checked)
	assert.Len(t, model.aliveLinks, 1)
	assert.Len(t, model.warningLinks, 1)
	assert.Len(t, model.deadLinks, 1)
	assert.Len(t, model.duplicateLinks, 1)
}

func TestHandleAllChecksComplete_UpdatesList(t *testing.T) {
	t.Parallel()

	model := New(".", nil, []string{"md"}, false, nil, nil)
	model.warningLinks = []checker.Result{{Link: checker.Link{URL: "https://warn.example"}}}
	model.deadLinks = []checker.Result{{Link: checker.Link{URL: "https://dead.example"}}}

	updated, cmd := model.handleAllChecksComplete()
	require.NotNil(t, updated)
	assert.Nil(t, cmd)
	assert.Equal(t, stateResults, model.state)
	assert.Len(t, model.list.Items(), 2)
}

func TestUpdate_WindowSizeAndKeys(t *testing.T) {
	t.Parallel()

	model := New(".", nil, []string{"md"}, false, nil, nil)
	updated, cmd := model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	require.NotNil(t, updated)
	assert.Nil(t, cmd)
	updatedModel, ok := updated.(Model)
	require.True(t, ok)
	assert.Equal(t, 100, updatedModel.width)
	assert.Equal(t, 30, updatedModel.height)

	updated, cmd = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	require.NotNil(t, updated)
	assert.Nil(t, cmd)
	updatedModel, ok = updated.(Model)
	require.True(t, ok)
	assert.True(t, updatedModel.showHelp)
}

func TestUpdate_FilterCycleAndQuitCancel(t *testing.T) {
	t.Parallel()

	model := New(".", nil, []string{"md"}, false, nil, nil)
	model.state = stateResults
	model.warningLinks = []checker.Result{{Link: checker.Link{URL: "https://warn.example"}}}
	model.deadLinks = []checker.Result{{Link: checker.Link{URL: "https://dead.example"}}}

	filterKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}
	updated, cmd := model.Update(filterKey)
	require.NotNil(t, updated)
	assert.Nil(t, cmd)
	updatedModel, ok := updated.(Model)
	require.True(t, ok)
	assert.Equal(t, filterWarnings, updatedModel.filter)

	cancelCalled := false
	updatedModel.state = stateChecking
	updatedModel.checkerState.CancelFunc = func() { cancelCalled = true }
	updated, cmd = updatedModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, updated)
	require.NotNil(t, cmd)
	updatedModel, ok = updated.(Model)
	require.True(t, ok)
	assert.True(t, cancelCalled)
	assert.True(t, updatedModel.quitting)
}

func TestGetFilteredResultsAndView(t *testing.T) {
	t.Parallel()

	model := New(".", nil, []string{"md"}, false, nil, nil)
	model.state = stateResults
	model.files = []string{"README.md"}
	model.links = []checker.Link{{URL: "https://alive.example"}}
	model.aliveLinks = []checker.Result{{Status: checker.StatusAlive, Link: checker.Link{URL: "https://alive.example"}}}
	model.warningLinks = []checker.Result{{
		Status: checker.StatusRedirect,
		Link:   checker.Link{URL: "https://warn.example"},
	}}
	model.deadLinks = []checker.Result{{Status: checker.StatusDead, Link: checker.Link{URL: "https://dead.example"}}}
	model.duplicateLinks = []checker.Result{{
		Status: checker.StatusDuplicate,
		Link:   checker.Link{URL: "https://dup.example"},
	}}

	assert.Len(t, model.getFilteredResults(), 3)
	model.filter = filterDead
	assert.Len(t, model.getFilteredResults(), 1)

	view := model.View()
	assert.Contains(t, view, "Filter:")
	assert.Contains(t, view, "✓ 1 alive")
	assert.Contains(t, view, "⚠ 1 warnings")
}

func TestKeyMapHelp(t *testing.T) {
	t.Parallel()

	keys := DefaultKeyMap()
	assert.Len(t, keys.ShortHelp(), 5)
	assert.Len(t, keys.FullHelp(), 3)
	assert.True(t, key.Matches(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}, keys.Filter))
}
