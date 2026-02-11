package install

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kaofelix/skulls/internal/gitutil"
)

type DiscoveredSkill struct {
	Name          string
	SkillDirPath  string
	SkillFilePath string
}

// DiscoverSkills discovers skills in a source repository by scanning skills/**/SKILL.md
// and reading YAML frontmatter `name`.
//
// For remote sources it clones to a temp directory and returns a cleanup func that should be called.
// For local path sources cleanup is a no-op.
func DiscoverSkills(source string) ([]DiscoveredSkill, func(), error) {
	cloneURL, err := gitutil.NormalizeSourceToGitURL(source)
	if err != nil {
		return nil, nil, err
	}

	repoDir := cloneURL
	cleanup := func() {}

	if fi, err := os.Stat(cloneURL); err != nil || !fi.IsDir() {
		tmp, err := os.MkdirTemp("", "skulls-discover-*")
		if err != nil {
			return nil, nil, err
		}
		repoDir = filepath.Join(tmp, "repo")
		if err := gitutil.CloneShallowTo(cloneURL, repoDir, io.Discard, io.Discard); err != nil {
			_ = os.RemoveAll(tmp)
			return nil, nil, err
		}
		cleanup = func() { _ = os.RemoveAll(tmp) }
	}

	root := filepath.Join(repoDir, "skills")
	if fi, err := os.Stat(root); err != nil || !fi.IsDir() {
		cleanup()
		return nil, nil, fmt.Errorf("skills directory not found in repo")
	}

	out := make([]DiscoveredSkill, 0, 16)
	seen := map[string]string{}
	walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.EqualFold(d.Name(), "SKILL.md") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		name, ok := parseSkillNameFromFrontmatter(string(b))
		if !ok {
			return nil
		}
		if prev, exists := seen[name]; exists {
			return fmt.Errorf("duplicate skill name %q in %s and %s", name, prev, p)
		}
		seen[name] = p
		out = append(out, DiscoveredSkill{
			Name:          name,
			SkillDirPath:  filepath.Dir(p),
			SkillFilePath: p,
		})
		return nil
	})
	if walkErr != nil {
		cleanup()
		return nil, nil, walkErr
	}
	if len(out) == 0 {
		cleanup()
		return nil, nil, fmt.Errorf("no skills found")
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})

	return out, cleanup, nil
}
