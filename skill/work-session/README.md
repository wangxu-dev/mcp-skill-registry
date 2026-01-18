# Work Session

Work session context preservation for AI assistants.

## Purpose

Maintain development continuity across disconnected conversations by preserving:
- What was accomplished
- Which files were modified
- Current TODO list
- Git state (branch, uncommitted changes)

## Usage

```bash
# At the START of work (restores previous session)
/work-session restore

# At the END of work (saves current session)
/work-session save
```

## Session File

Location: Project root (`.session.md`)

Format:
```md
# Work Session (2026-01-18 18:45)

## Summary
- Completed JWT token refresh logic

## Files Modified
- apps/back/src/modules/auth/routes.ts:75-82

## TODOs
- [ ] Add token expiration error handling

## Next Steps
- Add error handling

## Git State
- Branch: feature/auth-refresh
- Uncommitted changes: true
```

The session file is **automatically deleted after restore** to prevent context pollution.
Do not store secrets, credentials, tokens, personal data, or private file contents in the session file.

## Workflow

```
Day 1 Morning: Start work
→ (no previous session)

Day 1 Evening: End work
→ /work-session save (saves to .session.json)

Day 2 Morning: Start work
→ /work-session restore (reads .session.md, displays summary, deletes file)
→ Continue with context restored
```

## Design Principles

1. **Explicit commands**: Separate `save` and `restore` for clarity
2. **Clean state**: File deleted after restore
3. **Minimal noise**: Concise, factual output
4. **No assumptions**: Don't prompt user during save, let user decide during restore
5. **Portable**: Works across multiple AI assistants
