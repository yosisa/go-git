package git

import (
	"reflect"
	"testing"
)

func TestCandidateRefs(t *testing.T) {
	for _, tc := range []struct {
		name     string
		expected []string
	}{
		{
			"master",
			[]string{"master", "refs/tags/master", "refs/heads/master", "refs/remotes/master"},
		},
		{
			"v1",
			[]string{"v1", "refs/tags/v1", "refs/heads/v1", "refs/remotes/v1"},
		},
		{
			"origin/master",
			[]string{"origin/master", "refs/tags/origin/master", "refs/heads/origin/master", "refs/remotes/origin/master"},
		},
		{
			"heads/v1",
			[]string{"heads/v1", "refs/heads/v1", "refs/tags/heads/v1", "refs/heads/heads/v1", "refs/remotes/heads/v1"},
		},
		{
			"refs/remotes/origin/master",
			[]string{"refs/remotes/origin/master", "refs/tags/refs/remotes/origin/master", "refs/heads/refs/remotes/origin/master", "refs/remotes/refs/remotes/origin/master"},
		},
	} {
		r := candidateRefs(tc.name)
		if !reflect.DeepEqual(r, tc.expected) {
			t.Errorf("Name: %v, Expected: %v, Got: %v", tc.name, tc.expected, r)
		}
	}
}
