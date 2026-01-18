package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"
)

type sourcesFile struct {
	Schema  string   `json:"$schema,omitempty"`
	Sources []source `json:"sources"`
}

type source struct {
	Repo    string   `json:"repo"`
	Branch  string   `json:"branch,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

type indexFile struct {
	Schema      string  `json:"$schema,omitempty"`
	GeneratedAt string  `json:"generatedAt,omitempty"`
	Skills      []skill `json:"skills"`
}

type skill struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Repo      string `json:"repo"`
	Head      string `json:"head"`
	UpdatedAt string `json:"updatedAt"`
}

type foundSkill struct {
	Name       string
	SourcePath string
}

type skillMeta struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	Head        string `json:"head,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
	CheckedAt   string `json:"checkedAt,omitempty"`
}

var defaultExclude = []string{
	"node_modules",
	"dist",
	"build",
	"out",
	"coverage",
	".git",
	".github",
	".vscode",
	".idea",
	".next",
	".turbo",
	"vendor",
	"target",
	"tmp",
	"temp",
	"bin",
	"obj",
}

func main() {
	var (
		sourcesPath = flag.String("sources", "sources.skill.json", "path to sources.skill.json")
		indexPath   = flag.String("index", "index.skill.json", "path to index.skill.json")
		sourcesDir  = flag.String("sources-dir", "sources", "directory to clone sources into")
		keepSources = flag.Bool("keep-sources", false, "keep cloned repos after update")
	)
	flag.Parse()

	if err := run(*sourcesPath, *indexPath, *sourcesDir, *keepSources); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(sourcesPath, indexPath, sourcesDir string, keepSources bool) error {
	src, err := loadSources(sourcesPath)
	if err != nil {
		return err
	}
	if len(src.Sources) == 0 {
		return errors.New("sources list is empty")
	}

	existing, err := loadIndex(indexPath)
	if err != nil {
		return err
	}

	existingByRepo := map[string][]skill{}
	existingHead := map[string]string{}
	for _, s := range existing.Skills {
		existingByRepo[s.Repo] = append(existingByRepo[s.Repo], s)
		if existingHead[s.Repo] == "" {
			existingHead[s.Repo] = s.Head
		}
	}

	repoNameSeen := map[string]string{}
	repoSeen := map[string]bool{}
	now := time.Now().UTC().Format(time.RFC3339)

	var updatedSkills []skill
	pathOwners := map[string]string{}
	for _, srcRepo := range src.Sources {
		repo := strings.TrimSpace(srcRepo.Repo)
		if repo == "" {
			return errors.New("source repo is empty")
		}
		if repoSeen[repo] {
			return fmt.Errorf("duplicate repo entry %q", repo)
		}
		repoSeen[repo] = true

		repoName := repoFolderName(repo)
		if repoName == "" {
			return fmt.Errorf("unable to derive repo folder name from %q", repo)
		}
		if prev, ok := repoNameSeen[repoName]; ok && prev != repo {
			return fmt.Errorf("duplicate repo folder name %q for %q and %q", repoName, prev, repo)
		}
		repoNameSeen[repoName] = repo

		head, err := gitRemoteHead(repo, srcRepo.Branch)
		if err != nil {
			return err
		}

		if existingHead[repo] == head && head != "" && !needsSourcePathUpdate(existingByRepo[repo]) {
			for _, s := range existingByRepo[repo] {
				updatedSkills = append(updatedSkills, s)
				pathOwners[destPathForName(s.Name)] = repo
				meta := skillMeta{
					Name:      s.Name,
					Head:      s.Head,
					UpdatedAt: s.UpdatedAt,
					CheckedAt: now,
				}
				if err := enrichMetaFromSkill(destPathForName(s.Name), &meta); err != nil {
					return err
				}
				if err := writeSkillMeta(destPathForName(s.Name), meta); err != nil {
					return err
				}
			}
			continue
		}

		repoSkills, actualHead, err := scanRepo(repo, srcRepo.Branch, sourcesDir, repoName, srcRepo.Exclude)
		if err != nil {
			return err
		}

		seenDest := map[string]bool{}
		for _, rs := range repoSkills {
			if rs.Name == "" {
				return fmt.Errorf("empty skill name in repo %q", repo)
			}
			if seenDest[rs.Name] {
				return fmt.Errorf("duplicate skill name %q in repo %q", rs.Name, repo)
			}
			seenDest[rs.Name] = true

			destPath := destPathForName(rs.Name)
			if owner, ok := pathOwners[destPath]; ok && owner != repo {
				return fmt.Errorf("skill path %q already owned by repo %q", destPath, owner)
			}
		}

		if err := removeRepoSkills(existingByRepo[repo]); err != nil {
			return err
		}

		if err := mirrorSkills("skill", filepath.Join(sourcesDir, repoName), repoSkills); err != nil {
			return err
		}

		for _, rs := range repoSkills {
			destPath := destPathForName(rs.Name)
			pathOwners[destPath] = repo
			entry := skill{
				Name:      rs.Name,
				Path:      rs.SourcePath,
				Repo:      repo,
				Head:      actualHead,
				UpdatedAt: now,
			}
			updatedSkills = append(updatedSkills, entry)
			meta := skillMeta{
				Name:      rs.Name,
				Head:      entry.Head,
				UpdatedAt: entry.UpdatedAt,
				CheckedAt: now,
			}
			if err := enrichMetaFromSkill(destPathForName(rs.Name), &meta); err != nil {
				return err
			}
			if err := writeSkillMeta(destPathForName(rs.Name), meta); err != nil {
				return err
			}
		}

		if !keepSources {
			_ = os.RemoveAll(filepath.Join(sourcesDir, repoName))
		}
	}

	sort.Slice(updatedSkills, func(i, j int) bool {
		if updatedSkills[i].Repo != updatedSkills[j].Repo {
			return updatedSkills[i].Repo < updatedSkills[j].Repo
		}
		if updatedSkills[i].Path != updatedSkills[j].Path {
			return updatedSkills[i].Path < updatedSkills[j].Path
		}
		return updatedSkills[i].Name < updatedSkills[j].Name
	})

	newIndex := indexFile{
		Schema: existing.Schema,
		Skills: updatedSkills,
	}

	if !reflect.DeepEqual(existing.Skills, newIndex.Skills) || !reflect.DeepEqual(existing.Schema, newIndex.Schema) {
		newIndex.GeneratedAt = now
		return writeIndex(indexPath, newIndex)
	}

	return nil
}

func loadSources(path string) (sourcesFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return sourcesFile{}, err
	}
	var src sourcesFile
	if err := json.Unmarshal(data, &src); err != nil {
		return sourcesFile{}, err
	}
	return src, nil
}

func loadIndex(path string) (indexFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return indexFile{}, nil
		}
		return indexFile{}, err
	}
	var idx indexFile
	if err := json.Unmarshal(data, &idx); err != nil {
		return indexFile{}, err
	}
	return idx, nil
}

func writeIndex(path string, idx indexFile) error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func gitRemoteHead(repo, branch string) (string, error) {
	ref := branch
	if ref == "" {
		ref = "HEAD"
	}
	out, err := runGit("", "ls-remote", repo, ref)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return "", fmt.Errorf("git ls-remote returned no data for %q %q", repo, ref)
	}
	return fields[0], nil
}

func scanRepo(repo, branch, sourcesDir, repoName string, extraExclude []string) ([]foundSkill, string, error) {
	if err := os.MkdirAll(sourcesDir, 0755); err != nil {
		return nil, "", err
	}
	dest := filepath.Join(sourcesDir, repoName)
	_ = os.RemoveAll(dest)

	cloneArgs := []string{"clone", "--depth", "1"}
	if branch != "" {
		cloneArgs = append(cloneArgs, "--branch", branch)
	}
	cloneArgs = append(cloneArgs, repo, dest)
	if _, err := runGit("", cloneArgs...); err != nil {
		return nil, "", err
	}

	head, err := runGit(dest, "rev-parse", "HEAD")
	if err != nil {
		return nil, "", err
	}
	head = strings.TrimSpace(head)

	excludeSet := buildExcludeSet(extraExclude)
	var skills []foundSkill
	err = filepath.WalkDir(dest, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if shouldSkipDir(name, excludeSet) {
				return fs.SkipDir
			}
			return nil
		}
		if strings.EqualFold(d.Name(), "SKILL.md") {
			dir := filepath.Dir(path)
			rel, err := filepath.Rel(dest, dir)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			skills = append(skills, foundSkill{
				Name:       filepath.Base(dir),
				SourcePath: rel,
			})
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	if len(skills) == 0 {
		return nil, head, nil
	}

	sort.Slice(skills, func(i, j int) bool {
		if skills[i].SourcePath != skills[j].SourcePath {
			return skills[i].SourcePath < skills[j].SourcePath
		}
		return skills[i].Name < skills[j].Name
	})

	return skills, head, nil
}

func buildExcludeSet(extra []string) map[string]bool {
	set := map[string]bool{}
	for _, name := range defaultExclude {
		set[strings.ToLower(name)] = true
	}
	for _, name := range extra {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		set[strings.ToLower(name)] = true
	}
	return set
}

func shouldSkipDir(name string, exclude map[string]bool) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	return exclude[strings.ToLower(name)]
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(string(out)), nil
}

func repoFolderName(repo string) string {
	repo = strings.TrimSpace(repo)
	repo = strings.TrimSuffix(repo, "/")
	repo = strings.TrimSuffix(repo, ".git")

	if strings.HasPrefix(repo, "git@") && strings.Contains(repo, ":") {
		parts := strings.Split(repo, ":")
		repo = parts[len(parts)-1]
	}
	if strings.Contains(repo, "://") {
		if idx := strings.Index(repo, "://"); idx >= 0 {
			repo = repo[idx+3:]
		}
		if idx := strings.Index(repo, "/"); idx >= 0 {
			repo = repo[idx+1:]
		}
	}
	repo = strings.TrimSuffix(repo, "/")
	repo = strings.TrimSuffix(repo, ".git")

	repo = filepath.ToSlash(repo)
	parts := strings.Split(repo, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func removeRepoSkills(entries []skill) error {
	for _, s := range entries {
		if s.Name == "" {
			continue
		}
		target, ok := safeSkillPath(destPathForName(s.Name))
		if !ok {
			return fmt.Errorf("refusing to remove unexpected path for %q", s.Name)
		}
		if err := os.RemoveAll(target); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func mirrorSkills(skillRoot, repoDir string, entries []foundSkill) error {
	if err := os.MkdirAll(skillRoot, 0755); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.Name == "" {
			return errors.New("skill name is empty")
		}
		if seen[entry.Name] {
			return fmt.Errorf("duplicate skill name %q", entry.Name)
		}
		seen[entry.Name] = true

		src := filepath.Join(repoDir, filepath.FromSlash(entry.SourcePath))
		dst := filepath.Join(skillRoot, entry.Name)
		_ = os.RemoveAll(dst)
		if err := copyDir(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func writeSkillMeta(skillPath string, meta skillMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(skillPath, "skill.meta.json"), data, 0644)
}

func enrichMetaFromSkill(skillPath string, meta *skillMeta) error {
	version, description, err := readSkillFrontmatter(filepath.Join(skillPath, "SKILL.md"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	meta.Version = version
	meta.Description = description
	return nil
}

func readSkillFrontmatter(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	lines := strings.Split(string(data), "\n")
	var body []string
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for i := 1; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "---" {
				body = lines[1:i]
				break
			}
		}
	}
	if body == nil {
		limit := 40
		if len(lines) < limit {
			limit = len(lines)
		}
		body = lines[:limit]
	}

	var (
		version     string
		description string
	)
	for _, line := range body {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "description:") && description == "" {
			value := strings.TrimSpace(trimmed[len("description:"):])
			description = trimQuoted(value)
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "version:") && version == "" {
			value := strings.TrimSpace(trimmed[len("version:"):])
			version = trimQuoted(value)
		}
	}

	return version, description, nil
}

func trimQuoted(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", src)
	}
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode())
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	if _, err := out.ReadFrom(in); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
}

func safeSkillPath(rel string) (string, bool) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) {
		return "", false
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", false
	}
	parts := strings.Split(clean, string(os.PathSeparator))
	if len(parts) == 0 || parts[0] != "skill" {
		return "", false
	}
	return clean, true
}

func destPathForName(name string) string {
	return filepath.ToSlash(filepath.Join("skill", name))
}

func needsSourcePathUpdate(entries []skill) bool {
	for _, entry := range entries {
		if entry.Name == "" {
			continue
		}
		if entry.Path == destPathForName(entry.Name) {
			return true
		}
	}
	return false
}
