# mcp-skill-registry

Minimal registry repository for skills.

## What it does
- Syncs from a list of source repos
- Scans for folders containing `SKILL.md`
- Mirrors each skill folder into `skill/`
- Generates `index.skill.json` with source repo/path/head/updatedAt

## Layout
- `cmd/skill-indexer/` Go updater
- `sources.skill.json` Source repo list
- `schema/` JSON Schema for editor validation
- `skill/` Mirrored skill folders
- `index.skill.json` Generated index
  - `repo`/`path` are the source repo/path for traceability
  - installers fetch from this registry repo and use `skill/<name>`

## Usage

Build:
```
go build -o ./main.exe ./cmd/skill-indexer
```

Run:
```
./main.exe -sources sources.skill.json -index index.skill.json -sources-dir sources
```

Keep cloned sources for debugging:
```
./main.exe -sources sources.skill.json -index index.skill.json -sources-dir sources -keep-sources
```

## sources.skill.json

Example:
```
{
  "$schema": "./schema/sources.skill.schema.json",
  "sources": [
    {
      "repo": "https://github.com/vercel-labs/agent-skills",
      "branch": "main",
      "exclude": ["docs", "examples"]
    }
  ]
}
```
