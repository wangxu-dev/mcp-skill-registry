---
name: work-session
description: Preserve work progress across disconnected sessions. Use '/work-session save' when ending work, '/work-session restore' when starting fresh.
license: Apache-2.0
metadata:
  author: wangxu-dev
  version: "0.0.1"
  argument-hint: <save|restore>
---

# Work Session Context Preservation

Maintain development continuity across disconnected conversations by preserving progress, modified files, TODOs, and git state.

## Commands

| Command | When to Use | Behavior |
|---------|-------------|----------|
| `/work-session save` | End of work session | Write current state to `.session.md` |
| `/work-session restore` | Start of new session | If `.session.md` exists: read → display summary → **delete file** |

## How It Works

**End of session (with context):**
```
/work-session save
→ Analyze conversation → Save to .session.md
```

**Start of new session (file exists):**
```
/work-session restore
→ Read .session.md → Display summary → Delete file
```

**Start of new session (no file found):**
```
/work-session restore
→ Report "No saved session found" and explain how to save at end of work
```

## Session File Format

`.session.md` is stored at project root (gitignored):

```md
# Work Session (2026-01-18 18:45)

## Summary
- Completed JWT token refresh logic in auth module

## Files Modified
- apps/back/src/modules/auth/routes.ts:75-82
- apps/back/src/modules/auth/middleware.ts

## TODOs
- [ ] Add token expiration error handling

## Next Steps
- Implement error handling for expired tokens

## Git State
- Branch: feature/auth-refresh
- Uncommitted changes: true
```

## Save Behavior (End of Work)

When saving, analyze the conversation history and extract:

1. **Summary**: What was accomplished (1-2 sentences)
2. **Files Modified**: List files with line numbers if applicable
3. **Current TODOs**: Capture any tracked TODOs
4. **Next Steps Hint**: Predict what might come next based on context
5. **Git State**: Current branch and uncommitted status

**Output format:**
```
Session saved at 18:45

Summary:
- Completed JWT token refresh logic in auth module
- Modified apps/back/src/modules/auth/routes.ts:75-82

TODOs preserved:
- [ ] Add token expiration error handling

Next session may continue with error handling implementation.
```

## Restore Behavior (Start of Work)

When restoring, display the saved context, state that it was consumed, and immediately delete the file.

**Output format:**
```
Restoring session from 18:45 (2 days ago)

Previous work:
- Completed JWT token refresh logic in auth module
- Modified apps/back/src/modules/auth/routes.ts:75-82

TODOs:
- [ ] Add token expiration error handling

Context hints:
- Branch: feature/auth-refresh
- Uncommitted changes detected

Session record consumed and deleted.
Ready to continue with error handling, or something else?
```

**If no session file is found:**
```
No saved session found.
Tip: Use /work-session save at the end of work to preserve context.
```

## Strict Guidelines

### Prohibited Behaviors
- Do **not** use flattery or self-aggrandizing language
- Do **not** ask "Do you want to do this?" repeatedly
- Do **not** output irrelevant or tangential information
- Do **not** make assumptions about user preferences during save
- Do **not** keep the session file after restore (must delete)
- Do **not** store secrets, credentials, tokens, personal data, or private file contents

### Required Behaviors
- **Save**: Automatically analyze and summarize without asking questions
- **Restore**: Load context, display it, delete file, then wait for user direction
- **Privacy**: If sensitive data appears in context, omit it and keep only safe, high-level summaries
- **Tone**: Neutral, concise, factual
- **Focus**: Maximum context preservation with minimum noise

## Usage

```bash
# End of work (save current session)
/work-session save

# Start of new work (restore previous session)
/work-session restore
```

**Workflow:**
```
Day 1 Evening: /work-session save
Day 2 Morning: /work-session restore
```
