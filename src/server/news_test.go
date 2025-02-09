package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/letsblockit/letsblockit/src/news"
	"github.com/letsblockit/letsblockit/src/pages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/blog/atom"
)

var exampleReleases = []*news.Release{{
	CreatedAt:   fixedNow.Add(time.Hour),
	PublishedAt: fixedNow.Add(time.Hour),
}, {
	CreatedAt: fixedNow.Add(time.Hour),
}, {
	CreatedAt:   fixedNow,
	PublishedAt: fixedNow.Add(2 * time.Hour),
}, {
	CreatedAt:   fixedNow.Add(-1 * time.Hour),
	PublishedAt: fixedNow,
}}

func (s *ServerTestSuite) TestNews_Anonymous() {
	s.user = ""
	req := httptest.NewRequest(http.MethodGet, "/news", nil)
	s.releases = exampleReleases
	s.expectRender("news", pages.ContextData{
		"releases":    exampleReleases,
		"newReleases": make(map[string]bool),
	})
	s.runRequest(req, assertOk)
}

func (s *ServerTestSuite) TestNews_LoggedIn() {
	req := httptest.NewRequest(http.MethodGet, "/news", nil)
	s.releases = exampleReleases
	s.expectRender("news", pages.ContextData{
		"releases": exampleReleases,
		"newReleases": map[string]bool{
			"0": true,
			"1": true,
		},
	})
	s.runRequest(req, assertOk)

	// Test news cursor has been updated
	pref, err := s.server.preferences.Get(s.c, s.user)
	require.NoError(s.T(), err)
	require.True(s.T(), exampleReleases[0].CreatedAt.Equal(pref.NewsCursor))
}

func (s *ServerTestSuite) TestNews_NoNews() {
	require.NoError(s.T(), s.server.preferences.UpdateNewsCursor(s.c, s.user, exampleReleases[0].CreatedAt))
	req := httptest.NewRequest(http.MethodGet, "/news", nil)
	s.releases = exampleReleases
	s.expectRender("news", pages.ContextData{
		"releases":    exampleReleases,
		"newReleases": map[string]bool{},
	})
	s.runRequest(req, assertOk)
}

func (s *ServerTestSuite) TestNewsAtom() {
	req := httptest.NewRequest(http.MethodGet, "/news.atom", nil)
	s.releases = exampleReleases
	s.runRequest(req, func(t *testing.T, rec *httptest.ResponseRecorder) {
		assert.Equal(t, http.StatusOK, rec.Code, rec.Body)
		assert.Equal(t, "application/atom+xml", rec.Header().Get("Content-Type"))
		assert.Equal(t, newsETag, rec.Header().Get("ETag"))

		var feed atom.Feed
		assert.NoError(t, xml.Unmarshal(rec.Body.Bytes(), &feed))
		assert.Len(t, feed.Entry, 4)
		assert.Equal(t, atom.Time(fixedNow.Add(2*time.Hour)), feed.Updated)
	})
}

func (s *ServerTestSuite) TestNewsAtom_NotModified() {
	req := httptest.NewRequest(http.MethodGet, "/news.atom", nil)
	req.Header.Set("If-None-Match", newsETag)
	s.releases = exampleReleases
	s.runRequest(req, func(t *testing.T, rec *httptest.ResponseRecorder) {
		assert.Equal(t, http.StatusNotModified, rec.Code)
		assert.Empty(t, rec.Body)
	})
}
