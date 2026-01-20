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

Maintain development continuity across disconnected conversations by preserving progress, modified files, TODOs, and git state in a durable session file, with minimal console output.

## Commands

| Command | When to Use | Behavior |
|---------|-------------|----------|
| `/work-session save` | End of work session | Write current state to `.session.md` (primary output). Console output should be a short confirmation only. |
| `/work-session restore` | Start of new session | If `.session.md` exists: read → display summary to user → load context → **delete file** |

## How It Works

**End of session (with context):**
```
/work-session save
→ Analyze conversation → Save to .session.md (rich detail) → Print brief confirmation only
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

`.session.md` is stored at project root (gitignored). It should be detailed and practical, capturing concrete work done and what remains. Keep it concise enough to scan, but not overly compressed.

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

When saving, analyze the full working context (conversation + commands run + files touched + decisions made) and extract:

1. **Summary**: What was accomplished (2-5 sentences)
2. **Developer-Style Notes**: Capture the developer's phrasing, priorities, and intent when clear; keep tone plain and factual
3. **Detailed Progress**: Concrete steps taken, decisions made, partial implementations, and current status
3. **Files Modified**: List files with line numbers if applicable
4. **Current TODOs**: Capture any tracked TODOs
5. **Next Steps Hint**: Predict what might come next based on context
6. **Open Questions**: Any unresolved questions, blockers, or ambiguity
7. **Git State**: Current branch and uncommitted status

**Output format (file content):**
```
Session saved at 18:45

Summary:
- Completed JWT token refresh logic in auth module
- Modified apps/back/src/modules/auth/routes.ts:75-82

Developer-Style Notes:
- Keep responses terse, avoid console spam; focus on practical next actions

Detailed Progress:
- Added refresh flow with rotation
- Updated middleware to read new token location
- Left TODO comments for error edge cases

TODOs preserved:
- [ ] Add token expiration error handling

Next session may continue with error handling implementation.

Open Questions:
- Decide whether to expire refresh token on login or on first use

Git State:
- Branch: feature/auth-refresh
- Uncommitted changes: true
```

**Console output (save):**
```
Session saved to .session.md
```

## Restore Behavior (Start of Work)

When restoring, display the saved context, load it into the assistant's working context, state that it was consumed, and immediately delete the file.

**Output format:**
```
Restoring session from 18:45 (2 days ago)

Previous work:
- Completed JWT token refresh logic in auth module
- Modified apps/back/src/modules/auth/routes.ts:75-82

Detailed Progress:
- Added refresh flow with rotation
- Updated middleware to read new token location
- Left TODO comments for error edge cases

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
- **Save**: Automatically analyze and summarize without asking questions; write a detailed, practical record to `.session.md`
- **Restore**: Load context, display it, delete file, then wait for user direction
- **Privacy**: If sensitive data appears in context, omit it and keep only safe, high-level summaries
- **Tone**: Neutral, concise, factual
- **Focus**: Maximum context preservation with minimum noise; avoid console spam

### Accuracy and Voice
- Prefer concrete facts over guesswork; if unsure, mark as "uncertain".
- Preserve the developer's intent and preferred style when it is clearly expressed.
- Avoid flattery, moralizing, or euphemisms; keep it plain and direct.

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
